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
	"errors"
	"fmt"
	"time"
)

const minimalRequiredDockerAPIVersion = 1.22

type Config struct {
	// Endpoint The URL of the docker server. Default is "unix:///var/run/docker.sock"
	Endpoint string `mapstructure:"endpoint"`

	// Timeout The maximum amount of time to wait for docker API responses. Default is 5s
	Timeout time.Duration `mapstructure:"timeout"`

	// DockerAPIVersion Docker client API version. Default & minimal supported version is 1.22
	DockerAPIVersion float64 `mapstructure:"api_version"`

	// ExpiryDuration Expiry duration for removed containers in the cache. Defaults is 5s
	ExpiryDuration time.Duration `mapstructure:"expiry_duration"`
}

// NewDefaultConfig creates a new config with default values
// to be used when creating a docker client
func NewDefaultConfig() Config {
	return Config{
		Endpoint:         "unix:///var/run/docker.sock",
		Timeout:          5 * time.Second,
		DockerAPIVersion: minimalRequiredDockerAPIVersion,
		ExpiryDuration:   5 * time.Second,
	}
}

func (config Config) Validate() error {
	if config.Endpoint == "" {
		return errors.New("the endpoint must be specified")
	}
	if config.DockerAPIVersion < minimalRequiredDockerAPIVersion {
		return fmt.Errorf("the Docker API version must be at least %v", minimalRequiredDockerAPIVersion)
	}
	return nil
}
