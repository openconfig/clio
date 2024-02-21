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
	"testing"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestHandleMetrics(t *testing.T) {
	g := &GNMI{
		cfg: &Config{
			TargetName: "target",
			Sep:        "/",
		},
		metricCh: make(chan *pmetric.Metrics, 10),
	}
	var n *gpb.Notification
	updateFn := func(notif *gpb.Notification) error {
		n = notif
		return nil
	}
	td := GenerateMetrics(20)
	g.handleMetrics(nil, updateFn, "", nil)
	if err := g.storeMetric(context.Background(), td); err != nil {
		t.Errorf("storeMetric returned error: %v", err)
	}
	close(g.metricCh)

	time.Sleep(time.Second)
	if len(n.Update) != 40 {
		t.Errorf("missing updates: want 40 got %d", len(n.Update))
	}
}

var (
	metricStartTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC))
	metricTimestamp      = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

const (
	TestSumIntMetricName = "sum/int"
)

func GenerateMetrics(count int) pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	ms.EnsureCapacity(count)

	for i := 0; i < count; i++ {
		initSumIntMetric(ms.AppendEmpty())
	}
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

func initSumIntMetric(im pmetric.Metric) {
	initMetric(im, TestSumIntMetricName, pmetric.MetricTypeSum)

	idps := im.Sum().DataPoints()
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetIntValue(123)
	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetIntValue(456)
}

func initMetric(m pmetric.Metric, name string, ty pmetric.MetricType) {
	m.SetName(name)
	m.SetDescription("")
	m.SetUnit("1")
	switch ty {
	case pmetric.MetricTypeGauge:
		m.SetEmptyGauge()
	case pmetric.MetricTypeSum:
		sum := m.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	case pmetric.MetricTypeHistogram:
		histo := m.SetEmptyHistogram()
		histo.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	case pmetric.MetricTypeExponentialHistogram:
		histo := m.SetEmptyExponentialHistogram()
		histo.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	case pmetric.MetricTypeSummary:
		m.SetEmptySummary()
	}
}
