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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestInvalidEndpoint(t *testing.T) {
	config := &Config{
		Endpoint: "$notavalidendpoint*",
	}
	cli, err := NewDockerClient(zap.NewNop(), config)
	assert.Nil(t, cli)
	require.Error(t, err)
	assert.Equal(t, "could not create docker client: unable to parse docker host `$notavalidendpoint*`", err.Error())
}

func TestStuff(t *testing.T) {
	//    t.Skip("requires a docker socket")
	cfg := NewDefaultConfig()
	client, err := NewDockerClient(zaptest.NewLogger(t), &cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	_ = client.InitializeContainersCache(context.Background())
	assert.Greater(t, len(client.CachedContainers()), 0)

	client.WatchEvents(context.Background())
}
