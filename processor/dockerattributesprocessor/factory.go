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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/dockerattributesprocessor/internal/docker"
)

const (
	// The value of "type" key in configuration.
	typeStr = "dockerattributes"
	// The stability level of the exporter.
	stability = component.StabilityLevelAlpha
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// NewFactory returns a new factory for the dockerattributes processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithTraces(createTracesProcessor, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Config: docker.NewDefaultConfig(),
	}
}

func createLogsProcessor(ctx context.Context,
	param processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	p := newDockerProcessor(param.Logger, oCfg, param.BuildInfo)

	return processorhelper.NewLogsProcessor(
		ctx,
		param,
		cfg,
		nextConsumer,
		p.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
	)
}

func createMetricsProcessor(ctx context.Context,
	param processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	p := newDockerProcessor(param.Logger, oCfg, param.BuildInfo)

	return processorhelper.NewMetricsProcessor(
		ctx,
		param,
		cfg,
		nextConsumer,
		p.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
	)
}

func createTracesProcessor(ctx context.Context,
	param processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	p := newDockerProcessor(param.Logger, oCfg, param.BuildInfo)

	return processorhelper.NewTracesProcessor(
		ctx,
		param,
		cfg,
		nextConsumer,
		p.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
	)
}
