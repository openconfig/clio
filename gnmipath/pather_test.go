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

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestProcessMetrics(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		inOldSep string
		inNewSep string
		inName   string
		wantName string
	}{
		{
			name:     "dots-and-slashes",
			inOldSep: ".",
			inNewSep: "/",
			inName:   "some.dotted.name",
			wantName: "some/dotted/name",
		},
		{
			name:     "dashes-and-dots",
			inOldSep: "-",
			inNewSep: ".",
			inName:   "some-dotted-name",
			wantName: "some.dotted.name",
		},
		{
			name:     "no-match",
			inOldSep: "-",
			inNewSep: ".",
			inName:   "some/dotted/name",
			wantName: "some/dotted/name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			p := NewPather(&Config{
				OldSep: test.inOldSep,
				NewSep: test.inNewSep,
			})
			ms := GenerateMetric(test.inName)
			nms, err := p.processMetrics(ctx, ms)
			if err != nil {
				t.Errorf("processMetrics() returned error: %v", err)
			}

			if test.wantName != metricName(t, nms) {
				t.Errorf("processMetrics() returned diff: want %s got %s", test.wantName, metricName(t, nms))
			}
		})
	}

}

func GenerateMetric(name string) pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	ms.EnsureCapacity(1)

	m := ms.AppendEmpty()
	m.SetName(name)

	return md
}

func initResource(r pcommon.Resource) {
	r.Attributes().PutStr("resource-attr", "resource-attr-val-1")
}

func generateMetricsOneEmptyInstrumentationScope() pmetric.Metrics {
	md := pmetric.NewMetrics()
	initResource(md.ResourceMetrics().AppendEmpty().Resource())
	md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
	return md
}

func metricName(t *testing.T, ms pmetric.Metrics) string {
	t.Helper()

	rms := ms.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				return m.Name()
			}
		}
	}

	t.Error("metric has no name")
	return ""
}
