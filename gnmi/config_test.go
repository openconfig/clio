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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	if err := confmap.New().Unmarshal(cfg); err != nil {
		t.Errorf("UnmarshalConfig returned error: %v", err)
	}

	if diff := cmp.Diff(cfg, factory.CreateDefaultConfig()); diff != "" {
		t.Errorf("UnmarshalConfig() returned diff (-got, +want):\n%s", diff)
	}
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	got := factory.CreateDefaultConfig()
	if err := cm.Unmarshal(got); err != nil {
		t.Errorf("UnmarshalConfig returned error: %v", err)
	}

	want := &Config{
		Addr:       "localhost:10",
		Sep:        "/",
		TargetName: "target",
		BufferSize: 10,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("UnmarshalConfig() returned diff (-got, +want):\n%s", diff)
	}
}
