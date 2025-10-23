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

// Program clio is an OpenTelemetry Collector binary.
package main

import (
	"log"
	"path/filepath"

	"github.com/openconfig/clio/collector"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

func main() {
	info := component.BuildInfo{
		Command:     "gnmi-collector",
		Description: "An otel collector that provides telemetry over gNMI.",
		Version:     "1.0.0",
	}

	settings := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: collector.Components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{filepath.Join("../config", "config.yaml")},
				ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), yamlprovider.NewFactory()},
				DefaultScheme:     "file",
			},
		},
	}
	if err := runInteractive(settings); err != nil {
		log.Fatal(err)
	}
}

func runInteractive(params otelcol.CollectorSettings) error {
	cmd := otelcol.NewCommand(params)
	if err := cmd.Execute(); err != nil {
		log.Fatalf("collector server run finished with error: %v", err)
	}

	return nil
}
