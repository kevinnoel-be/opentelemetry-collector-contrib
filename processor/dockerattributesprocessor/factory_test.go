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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/dockerattributesprocessor/internal/docker"
)

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, component.Type(typeStr), factory.Type())
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &Config{
		Config: docker.Config{
			Endpoint:         "unix:///var/run/docker.sock",
			Timeout:          5 * time.Second,
			DockerAPIVersion: 1.22,
			ExpiryDuration:   5 * time.Second,
		},
	}, cfg)
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}
