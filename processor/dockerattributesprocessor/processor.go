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

package dockerattributesprocessor

import (
	"context"
	"fmt"
	"strings"

	dclient "github.com/docker/docker/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.9.0"
	"go.uber.org/zap"

	dcommon "github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/docker"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/dockerattributesprocessor/internal/docker"
)

type dockerProcessor struct {
	logger    *zap.Logger
	config    *Config
	buildInfo component.BuildInfo

	client *docker.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func newDockerProcessor(logger *zap.Logger, cfg *Config, buildInfo component.BuildInfo) *dockerProcessor {
	return &dockerProcessor{
		logger:    logger,
		config:    cfg,
		buildInfo: buildInfo,
	}
}

func (p *dockerProcessor) Start(_ context.Context, _ component.Host) error {
	var err error
	p.client, err = docker.NewDockerClient(p.logger, &p.config.Config,
		dclient.WithHTTPHeaders(map[string]string{"User-Agent": fmt.Sprintf("%v %v", typeStr, p.buildInfo.Version)}),
	)
	if err != nil {
		return err
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())

	if err := p.client.InitializeContainersCache(p.ctx); err != nil {
		p.cancel()
		return err
	}

	go p.client.WatchEvents(p.ctx)

	return nil
}

func (p *dockerProcessor) Shutdown(_ context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *dockerProcessor) processLogs(ctx context.Context, logs plog.Logs) (plog.Logs, error) {
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		p.processResource(ctx, resourceLogs.At(i).Resource())
	}
	return logs, nil
}

func (p *dockerProcessor) processMetrics(ctx context.Context, metrics pmetric.Metrics) (pmetric.Metrics, error) {
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		p.processResource(ctx, resourceMetrics.At(i).Resource())
	}
	return metrics, nil
}

func (p *dockerProcessor) processTraces(ctx context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		p.processResource(ctx, resourceSpans.At(i).Resource())
	}
	return traces, nil
}

func (p *dockerProcessor) processResource(_ context.Context, resource pcommon.Resource) {
	attributes := resource.Attributes()
	containerID, found := attributes.Get(semconv.AttributeContainerID)
	if !found {
		p.logger.Debug("no container id found")
		return
	}

	containers := p.client.CachedContainers()
	container, found := containers[containerID.Str()]
	if !found {
		p.logger.Debug("couldn't find container id", zap.String("id", containerID.Str()))
		return
	}

	imageRef, err := dcommon.ParseImageName(container.Image)
	if err != nil {
		p.logger.Debug("failed to parse image name", zap.String("image", container.Image))
	}

	putIfAbsent(attributes, semconv.AttributeContainerRuntime, "docker")
	putIfAbsent(attributes, semconv.AttributeContainerName, strings.TrimPrefix(container.Name, "/"))
	putIfAbsent(attributes, semconv.AttributeContainerImageName, imageRef.Repository)
	putIfAbsent(attributes, semconv.AttributeContainerImageTag, imageRef.Tag)
}

func putIfAbsent(attr pcommon.Map, key, value string) {
	if _, found := attr.Get(key); !found {
		attr.PutStr(key, value)
	}
}
