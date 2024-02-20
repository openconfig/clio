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
	"strings"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type Pather struct {
	oldSep, newSep string
}

func NewPather(cfg *Config) *Pather {
	return &Pather{
		oldSep: cfg.OldSep,
		newSep: cfg.NewSep,
	}
}

func (p *Pather) processMetrics(_ context.Context, ms pmetric.Metrics) (pmetric.Metrics, error) {
	rms := ms.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				m.SetName(strings.ReplaceAll(m.Name(), p.oldSep, p.newSep))
			}
		}
	}
	return ms, nil
}
