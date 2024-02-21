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

package gnmi

import (
	"context"

	"github.com/openconfig/clio/gnmi/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory returns a new factory for the gnmipath gnmipath.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		Addr:       ":6030",
		TargetName: "poodle",
		BufferSize: 1000,
		Sep:        "/",
	}
}

func createMetricsExporter(ctx context.Context, set exporter.CreateSettings, cfg component.Config) (exporter.Metrics, error) {
	g, err := NewGNMIExporter(set.Logger, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetricsExporter(ctx, set, cfg,
		g.storeMetric,
		exporterhelper.WithStart(g.Start),
		exporterhelper.WithShutdown(g.Stop),
	)
}
