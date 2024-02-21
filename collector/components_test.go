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

// Code generated by "go.opentelemetry.io/collector/cmd/builder". DO NOT EDIT.

package collector

import (
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestValidateConfigs(t *testing.T) {
	factories, err := Components()
	if err != nil {
		t.Fatal(err)
	}

	for k, factory := range factories.Receivers {
		if k != factory.Type() {
			t.Errorf("factory is mapped incorrectly: key: %v, actual type: %v", k, factory.Type())
		}
		if err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig()); err != nil {
			t.Errorf("receiver.CreateDefaultConfig() returned error: %v", err)
		}
	}

	for k, factory := range factories.Processors {
		if k != factory.Type() {
			t.Errorf("factory is mapped incorrectly: key: %v, actual type: %v", k, factory.Type())
		}
		if err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig()); err != nil {
			t.Errorf("processor.CreateDefaultConfig() returned error: %v", err)
		}
	}

	for k, factory := range factories.Exporters {
		if k != factory.Type() {
			t.Errorf("factory is mapped incorrectly: key: %v, actual type: %v", k, factory.Type())
		}
		if err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig()); err != nil {
			t.Errorf("exporter.CreateDefaultConfig() returned error: %v", err)
		}
	}

	for k, factory := range factories.Connectors {
		if k != factory.Type() {
			t.Errorf("factory is mapped incorrectly: key: %v, actual type: %v", k, factory.Type())
		}
		if err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig()); err != nil {
			t.Errorf("connector.CreateDefaultConfig() returned error: %v", err)
		}
	}

	for k, factory := range factories.Extensions {
		if k != factory.Type() {
			t.Errorf("factory is mapped incorrectly: key: %v, actual type: %v", k, factory.Type())
		}
		if err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig()); err != nil {
			t.Errorf("extension.CreateDefaultConfig() returned error: %v", err)
		}
	}
}
