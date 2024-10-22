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
	"sync"
	"testing"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestHandleMetrics(t *testing.T) {
	tests := []struct {
		name         string
		inCnt        int
		InMetricType pmetric.MetricType
		wantCnt      int
	}{
		{
			name:         "gauge-10",
			inCnt:        10,
			InMetricType: pmetric.MetricTypeGauge,
			wantCnt:      20,
		},
		{
			name:         "sum-10",
			inCnt:        10,
			InMetricType: pmetric.MetricTypeSum,
			wantCnt:      20,
		},
		{
			name:         "histogram-10",
			inCnt:        10,
			InMetricType: pmetric.MetricTypeHistogram,
			wantCnt:      20,
		},
		{
			name:         "exponential-histogram-10",
			inCnt:        10,
			InMetricType: pmetric.MetricTypeExponentialHistogram,
			wantCnt:      20,
		},
		{
			name:         "summary-10",
			inCnt:        10,
			InMetricType: pmetric.MetricTypeSummary,
			wantCnt:      20,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GNMI{
				cfg: &Config{
					TargetName: "target",
					Sep:        "/",
				},
				metricCh: make(chan *pmetric.Metrics, 10),
			}

			n := &gpb.Notification{}
			var nMu sync.Mutex
			updateFn := func(notif *gpb.Notification) error {
				nMu.Lock()
				defer nMu.Unlock()
				n.Update = append(n.Update, notif.Update...)
				return nil
			}

			td := GenerateMetrics(tc.inCnt, tc.InMetricType)
			g.handleMetrics(nil, updateFn, "", nil)
			if err := g.storeMetric(context.Background(), td); err != nil {
				t.Errorf("storeMetric returned error: %v", err)
			}
			close(g.metricCh)

			time.Sleep(time.Second)
			nMu.Lock()
			defer nMu.Unlock()
			if len(n.Update) != tc.wantCnt {
				t.Errorf("missing updates: want %d got %d", tc.wantCnt, len(n.Update))
			}
		})
	}
}

var (
	metricStartTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC))
	metricTimestamp      = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

const (
	TestSumIntMetricName               = "sum/int_and_double"
	TestHistogramMetricName            = "histogram/double"
	TestExponentialHistogramMetricName = "exponential-histogram/double"
	TestGaugeMetricName                = "gauge/int_and_double"
	TestSummaryMetricName              = "summary/double"
)

func GenerateMetrics(count int, ty pmetric.MetricType) pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	ms.EnsureCapacity(count)

	for i := 0; i < count; i++ {
		switch ty {
		case pmetric.MetricTypeGauge:
			initGaugeMetric(ms.AppendEmpty())
		case pmetric.MetricTypeSum:
			initSumMetric(ms.AppendEmpty())
		case pmetric.MetricTypeHistogram:
			initHistogramMetric(ms.AppendEmpty())
		case pmetric.MetricTypeExponentialHistogram:
			initExponentialHistogramMetric(ms.AppendEmpty())
		case pmetric.MetricTypeSummary:
			initSummaryMetric(ms.AppendEmpty())
		}
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

func initHistogramMetric(im pmetric.Metric) {
	initMetric(im, TestHistogramMetricName, pmetric.MetricTypeHistogram)

	idps := im.Histogram().DataPoints()
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)

	// Buckets are (-inf, 1), [1, 10), [10, 100), [100, inf)
	idp0.ExplicitBounds().EnsureCapacity(3)
	idp0.ExplicitBounds().Append(1.0, 10.0, 100.0)
	idp0.BucketCounts().EnsureCapacity(4)
	idp0.BucketCounts().Append(0, 10, 100, 0)
	idp0.SetCount(110)
	idp0.SetMax(91)
	idp0.SetMin(1)
	idp0.SetSum(5555) // drawn from 10*[1, 10) + 100*[10, 100)

	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)

	// Buckets are (-inf, 1), [1, 10), [10, 100), [100, inf)
	idp1.ExplicitBounds().EnsureCapacity(3)
	idp1.ExplicitBounds().Append(1.0, 10.0, 100.0)
	idp1.BucketCounts().EnsureCapacity(4)
	idp1.BucketCounts().Append(0, 20, 200, 0)
	idp1.SetCount(220)
	idp1.SetMax(92)
	idp1.SetMin(2)
	idp1.SetSum(7777) // drawn from 10*[1, 10) + 100*[10, 100)
}

func initExponentialHistogramMetric(im pmetric.Metric) {
	initMetric(im, TestExponentialHistogramMetricName, pmetric.MetricTypeExponentialHistogram)

	// The histogram and the exponential histogram are identical except for the bucket scale, see:
	// https://opentelemetry.io/docs/specs/otel/metrics/data-model/#exponential-scale
	idps := im.ExponentialHistogram().DataPoints()
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)

	// Buckets are (1, 1.09051], (1.09051, 1.18921], (1.18921, 1.29684], ...
	idp0.SetScale(3)
	idp0.Negative().BucketCounts().EnsureCapacity(4)
	idp0.Negative().BucketCounts().Append(1, 2, 3, 0)
	idp0.SetCount(6)
	idp0.SetMax(3)
	idp0.SetMin(1)
	idp0.SetSum(7.1)

	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)

	idp1.SetScale(3)
	idp1.Negative().BucketCounts().EnsureCapacity(4)
	idp1.Negative().BucketCounts().Append(3, 2, 2, 0)
	idp1.SetCount(7)
	idp1.SetMax(3)
	idp1.SetMin(2)
	idp1.SetSum(7.95)
}

func initSumMetric(im pmetric.Metric) {
	initMetric(im, TestSumIntMetricName, pmetric.MetricTypeSum)

	idps := im.Sum().DataPoints()

	// Integer data point.
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetIntValue(123)

	// Double data point.
	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetDoubleValue(456.7)
}

func initGaugeMetric(im pmetric.Metric) {
	initMetric(im, TestGaugeMetricName, pmetric.MetricTypeGauge)

	idps := im.Gauge().DataPoints()

	// Integer data point.
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetIntValue(123)

	// Double data point.
	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetDoubleValue(456.0)
}

func initSummaryMetric(im pmetric.Metric) {
	initMetric(im, TestSummaryMetricName, pmetric.MetricTypeSummary)

	idps := im.Summary().DataPoints()
	idp0 := idps.AppendEmpty()
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	qv0 := idp0.QuantileValues().AppendEmpty()
	qv0.SetQuantile(0.95)
	qv0.SetValue(9.5)
	idp1 := idps.AppendEmpty()
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	qv1 := idp1.QuantileValues().AppendEmpty()
	qv1.SetQuantile(0.9)
	qv1.SetValue(9.0)
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
