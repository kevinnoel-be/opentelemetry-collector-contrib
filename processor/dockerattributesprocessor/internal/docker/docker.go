// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	dtypes "github.com/docker/docker/api/types"
	dfilters "github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Container struct {
	*dtypes.ContainerJSON
}

type Client struct {
	logger *zap.Logger
	config *Config

	client         *docker.Client
	containers     map[string]Container
	containersLock sync.Mutex
}

func NewDockerClient(logger *zap.Logger, config *Config, opts ...docker.Opt) (*Client, error) {
	client, err := docker.NewClientWithOpts(
		append([]docker.Opt{
			docker.WithHost(config.Endpoint),
			docker.WithVersion(fmt.Sprintf("v%v", config.DockerAPIVersion)),
		}, opts...)...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %w", err)
	}

	dc := &Client{
		client:         client,
		config:         config,
		logger:         logger,
		containers:     make(map[string]Container),
		containersLock: sync.Mutex{},
	}
	return dc, nil
}

// CachedContainers get all cached containers per container id
func (dc *Client) CachedContainers() map[string]Container {
	dc.containersLock.Lock()
	defer dc.containersLock.Unlock()

	containers := make(map[string]Container, len(dc.containers))
	for cid, container := range dc.containers {
		containers[cid] = container
	}
	return containers
}

// InitializeContainersCache will initialize the cache with all containers
func (dc *Client) InitializeContainersCache(ctx context.Context) error {
	// Build initial container maps before starting loop
	options := dtypes.ContainerListOptions{
		All: true,
	}

	listCtx, cancel := context.WithTimeout(ctx, dc.config.Timeout)
	containerList, err := dc.client.ContainerList(listCtx, options)
	defer cancel()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, c := range containerList {
		wg.Add(1)
		go func(container dtypes.Container) {
			defer wg.Done()
			dc.add(ctx, container.ID)
		}(c)
	}
	wg.Wait()
	return nil
}

func (dc *Client) WatchEvents(ctx context.Context) {
	filters := dfilters.NewArgs([]dfilters.KeyValuePair{
		{Key: "type", Value: "container"},
		{Key: "event", Value: "destroy"},
		{Key: "event", Value: "die"},
		{Key: "event", Value: "pause"},
		{Key: "event", Value: "rename"},
		{Key: "event", Value: "stop"},
		{Key: "event", Value: "start"},
		{Key: "event", Value: "unpause"},
		{Key: "event", Value: "update"},
	}...,
	)
	lastTime := time.Now() // TODO minus removed cache time

EVENT_LOOP:
	for {
		options := dtypes.EventsOptions{
			Filters: filters,
			Since:   lastTime.Format(time.RFC3339Nano),
		}
		eventCh, errCh := dc.client.Events(ctx, options)

		for {
			select {
			case <-ctx.Done():
				return

			case event := <-eventCh:
				switch event.Action {
				case "destroy":
					dc.logger.Debug("container destroyed", zap.String("id", event.Actor.ID))
					dc.expire(event.Actor.ID)
				default:
					dc.logger.Debug("container event received", zap.String("id", event.Actor.ID), zap.String("action", event.Action))
					dc.add(ctx, event.Actor.ID)
				}

				if event.TimeNano > lastTime.UnixNano() {
					lastTime = time.Unix(0, event.TimeNano)
				}

			case err := <-errCh:
				// We are only interested when the context hasn't been canceled since requests made
				// with a closed context are guaranteed to fail.
				if ctx.Err() == nil {
					dc.logger.Error("error watching container events", zap.Error(err))
					// Either decoding or connection error has occurred, so we should resume the event loop after
					// waiting a moment.  In cases of extended daemon unavailability this will retry until
					// collector teardown or background context is closed.
					select {
					case <-time.After(3 * time.Second):
						continue EVENT_LOOP
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

// add queries the inspect api and adds the container in the cache.
func (dc *Client) add(ctx context.Context, cid string) {
	inspectCtx, cancel := context.WithTimeout(ctx, dc.config.Timeout)
	container, err := dc.client.ContainerInspect(inspectCtx, cid)
	defer cancel()
	if err != nil {
		dc.logger.Error("could not inspect container", zap.String("id", cid), zap.Error(err))
	}

	// TODO check dead, check not running/paused/..., set expiry if applicable
	if container.State.Dead {
		dc.expire(cid)
	}

	dc.containersLock.Lock()
	defer dc.containersLock.Unlock()

	dc.containers[cid] = Container{ContainerJSON: &container}
	dc.logger.Debug("added container", zap.String("id", cid))
}

// expire
func (dc *Client) expire(cid string) {
	dc.containersLock.Lock()
	defer dc.containersLock.Unlock()
	// TODO change to a TTL instead of directly removing
	delete(dc.containers, cid)
	dc.logger.Debug("container will expire", zap.String("id", cid), zap.Duration("expiry", dc.config.ExpiryDuration))
}
