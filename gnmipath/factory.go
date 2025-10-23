// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnmipath

import (
	"context"

	"github.com/openconfig/clio/gnmipath/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var processorCapabilities = consumer.Capabilities{MutatesData: false}

// NewFactory returns a new factory for the gnmipath gnmipath.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithMetrics(createMetrics, metadata.MetricsStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		OldSep: ".",
		NewSep: "/",
	}
}

func createMetrics(ctx context.Context, set processor.Settings, cfg component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	p := NewPather(cfg.(*Config))
	return processorhelper.NewMetrics(ctx, set, cfg, nextConsumer,
		p.processMetrics, processorhelper.WithCapabilities(processorCapabilities))
}
