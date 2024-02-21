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
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	if cfg == nil {
		t.Error("failed to create default config")
	}
	if err := componenttest.CheckConfigStruct(cfg); err != nil {
		t.Error(err)
	}
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	creationSet := processortest.NewNopCreateSettings()

	mp, err := factory.CreateMetricsProcessor(context.Background(), creationSet, cfg, consumertest.NewNop())
	if mp == nil {
		t.Error("nil metrics processor")
	}

	if err != nil {
		t.Errorf("unable to create processor: %v", err)
	}

	if err := mp.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}
