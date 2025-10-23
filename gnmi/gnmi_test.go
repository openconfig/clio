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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/testutil"
	ompb "go.opentelemetry.io/proto/otlp/metrics/v1"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

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

func TestHandleMetrics(t *testing.T) {
	expectedPfx := func(origin, target, container string) *gpb.Path {
		return &gpb.Path{
			Origin: origin,
			Target: target,
			Elem: []*gpb.PathElem{
				{
					Name: "containers",
				},
				{
					Name: "container",
					Key: map[string]string{
						"name": container,
					},
				},
			},
		}
	}
	tests := []struct {
		name         string
		inCnt        int
		InMetricType pmetric.MetricType
		inTarget     string
		inOrigin     string
		inResAttrs   map[string]string
		wantPrefix   *gpb.Path
		wantCnt      int
	}{
		{
			name:         "gauge-10",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeGauge,
			wantPrefix:   expectedPfx("", "", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "gauge-10-with-target",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeGauge,
			inTarget:     "moo-deng",
			wantPrefix:   expectedPfx("", "moo-deng", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "gauge-10-with-origin",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeGauge,
			inOrigin:     "capybara",
			wantPrefix:   expectedPfx("capybara", "", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "gauge-10-with-target-and-origin",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeGauge,
			inTarget:     "seals-on-ice-floe",
			inOrigin:     "orca-gang",
			wantPrefix:   expectedPfx("orca-gang", "seals-on-ice-floe", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "sum-10",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeSum,
			wantPrefix:   expectedPfx("", "", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "histogram-10",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeHistogram,
			wantPrefix:   expectedPfx("", "", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "exponential-histogram-10",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeExponentialHistogram,
			wantPrefix:   expectedPfx("", "", "test-container"),
			wantCnt:      21,
		},
		{
			name:         "summary-10",
			inResAttrs:   map[string]string{"container.name": "test-container"},
			inCnt:        10,
			InMetricType: pmetric.MetricTypeSummary,
			wantPrefix:   expectedPfx("", "", "test-container"),
			wantCnt:      21,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GNMI{
				logger: zap.NewExample(),
				cfg: &Config{
					TargetName: tc.inTarget,
					Sep:        "/",
					Origin:     tc.inOrigin,
				},
				metricCh: make(chan *pmetric.Metrics, 10),
			}

			var notifs []*gpb.Notification
			var nMu sync.Mutex
			updateFn := func(n *gpb.Notification) error {
				nMu.Lock()
				defer nMu.Unlock()
				notifs = append(notifs, n)
				return nil
			}

			td := GenerateMetrics(tc.inCnt, tc.InMetricType, tc.inResAttrs)
			g.handleMetrics(nil, updateFn, "", nil)
			if err := g.storeMetric(context.Background(), td); err != nil {
				t.Errorf("storeMetric returned error: %v", err)
			}
			close(g.metricCh)

			time.Sleep(time.Second)
			nMu.Lock()
			defer nMu.Unlock()
			if len(notifs) != tc.wantCnt {
				t.Errorf("missing updates: want %d got %d", tc.wantCnt, len(notifs))
			}
			for _, n := range notifs {
				if diff := cmp.Diff(n.Prefix, tc.wantPrefix, protocmp.Transform()); diff != "" {
					t.Errorf("prefix mismatch: diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func GenerateMetrics(count int, ty pmetric.MetricType, resAttrs map[string]string) pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope(resAttrs)
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
	r.Attributes().PutStr("container.name", "test-container")
	r.Attributes().PutEmptyMap("container.labels").FromRaw(map[string]any{"i-am": "groot"})
}

func generateMetricsOneEmptyInstrumentationScope(resAttrs map[string]string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	initResource(md.ResourceMetrics().AppendEmpty().Resource())
	for k, v := range resAttrs {
		md.ResourceMetrics().At(0).Resource().Attributes().PutStr(k, v)
	}
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

func TestNotificationsFromMetric(t *testing.T) {
	testPrefix := &gpb.Path{
		Target: "test-target",
		Origin: "test-origin",
		Elem: []*gpb.PathElem{
			{
				Name: "containers",
			},
			{
				Name: "container",
				Key:  map[string]string{"name": "test-container"},
			},
		},
	}

	// anyWrapOrFatal is a convenience function to wrap a proto in an anypb.Any message.
	anyWrapOrFatal := func(mgs proto.Message) *anypb.Any {
		any, err := anypb.New(mgs)
		if err != nil {
			t.Fatalf("failed to wrap proto in anypb: %v", err)
		}
		return any
	}

	// The resource attributes are used to determine the container name/
	resAttrs := map[string]string{"container.name": "test-container"}

	// Special metric for which first update has attributes, and second update does not.
	attredGaugeMetric := GenerateMetrics(10, pmetric.MetricTypeGauge, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	attredGaugeMetric.Gauge().DataPoints().At(0).Attributes().PutStr("the-key", "the-value")

	tests := []struct {
		name     string
		inMetric pmetric.Metric
		want     []*gpb.Notification
	}{
		{
			name:     "gauge-simple",
			inMetric: GenerateMetrics(10, pmetric.MetricTypeGauge, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0),
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "gauge"},
									{Name: "int_and_double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_IntVal{
									IntVal: 123,
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "gauge"},
									{Name: "int_and_double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_DoubleVal{
									DoubleVal: 456.0,
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "gauge-with-attributes-in-first-update",
			inMetric: attredGaugeMetric,
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "gauge"},
									{
										Name: "int_and_double",
										Key:  map[string]string{"the-key": "the-value"},
									},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_IntVal{
									IntVal: 123,
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "gauge"},
									{Name: "int_and_double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_DoubleVal{
									DoubleVal: 456.0,
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "sum-simple",
			inMetric: GenerateMetrics(10, pmetric.MetricTypeSum, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0),
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "sum"},
									{Name: "int_and_double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_IntVal{
									IntVal: 123,
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "sum"},
									{Name: "int_and_double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_DoubleVal{
									DoubleVal: 456.7,
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "histogram-simple",
			inMetric: GenerateMetrics(10, pmetric.MetricTypeHistogram, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0),
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "histogram"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.HistogramDataPoint{
										Count:          110,
										Min:            proto.Float64(1),
										Max:            proto.Float64(91),
										Sum:            proto.Float64(5555),
										BucketCounts:   []uint64{0, 10, 100, 0},
										ExplicitBounds: []float64{1.0, 10.0, 100.0},
									}),
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "histogram"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.HistogramDataPoint{
										Count:          220,
										Min:            proto.Float64(2),
										Max:            proto.Float64(92),
										Sum:            proto.Float64(7777),
										BucketCounts:   []uint64{0, 20, 200, 0},
										ExplicitBounds: []float64{1.0, 10.0, 100.0},
									}),
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "exponential-histogram-simple",
			inMetric: GenerateMetrics(10, pmetric.MetricTypeExponentialHistogram, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0),
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "exponential-histogram"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.ExponentialHistogramDataPoint{
										Count: 6,
										Min:   proto.Float64(1),
										Max:   proto.Float64(3),
										Sum:   proto.Float64(7.1),
										Negative: &ompb.ExponentialHistogramDataPoint_Buckets{
											BucketCounts: []uint64{1, 2, 3, 0},
											Offset:       0,
										},
										Positive: &ompb.ExponentialHistogramDataPoint_Buckets{},
										Scale:    3,
									}),
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "exponential-histogram"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.ExponentialHistogramDataPoint{
										Count: 7,
										Min:   proto.Float64(2),
										Max:   proto.Float64(3),
										Sum:   proto.Float64(7.95),
										Negative: &ompb.ExponentialHistogramDataPoint_Buckets{
											BucketCounts: []uint64{3, 2, 2, 0},
											Offset:       0,
										},
										Positive: &ompb.ExponentialHistogramDataPoint_Buckets{},
										Scale:    3,
									}),
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "summary-simple",
			inMetric: GenerateMetrics(10, pmetric.MetricTypeSummary, resAttrs).ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0),
			want: []*gpb.Notification{
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "summary"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.SummaryDataPoint{
										QuantileValues: []*ompb.SummaryDataPoint_ValueAtQuantile{
											{
												Quantile: 0.95,
												Value:    9.5,
											},
										},
									}),
								},
							},
						},
					},
				},
				{
					Prefix: testPrefix,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "summary"},
									{Name: "double"},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_AnyVal{
									AnyVal: anyWrapOrFatal(&ompb.SummaryDataPoint{
										QuantileValues: []*ompb.SummaryDataPoint_ValueAtQuantile{
											{
												Quantile: 0.9,
												Value:    9.0,
											},
										},
									}),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GNMI{
				logger: zap.NewExample(),
				cfg: &Config{
					TargetName: "test-target",
					Sep:        "/",
					Origin:     "test-origin",
				},
			}
			got := g.notificationsFromMetric(tc.inMetric, "test-container")
			if diff := cmp.Diff(tc.want, got, protocmp.Transform(), cmpopts.EquateEmpty(), protocmp.IgnoreFields(&gpb.Notification{}, "timestamp")); diff != "" {
				t.Errorf("notificationsFromMetric(%v, 'test-container') returned an unexpected diff (-want +got): %v", tc.inMetric, diff)
			}
		})
	}
}

func TestNotificationsFromLabels(t *testing.T) {
	tests := []struct {
		name     string
		inLabels map[string]any
		want     []*gpb.Notification
	}{
		{
			name: "simple",
			inLabels: map[string]any{
				"container.version": "1.0.0",
				"i.am.a":            "fancy.label",
			},
			want: []*gpb.Notification{
				{
					Prefix: &gpb.Path{
						Target: "test-target",
						Elem: []*gpb.PathElem{
							{
								Name: "containers",
							},
							{
								Name: "container",
								Key:  map[string]string{"name": "simple"},
							},
						},
						Origin: "test-origin",
					},
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "labels"},
									{
										Name: "label",
										Key:  map[string]string{"name": "container.version"},
									},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_StringVal{
									StringVal: "1.0.0",
								},
							},
						},
					},
				},
				{
					Prefix: &gpb.Path{
						Target: "test-target",
						Elem: []*gpb.PathElem{
							{
								Name: "containers",
							},
							{
								Name: "container",
								Key:  map[string]string{"name": "simple"},
							},
						},
						Origin: "test-origin",
					},
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{
								Target: "test-target",
								Elem: []*gpb.PathElem{
									{Name: "labels"},
									{
										Name: "label",
										Key:  map[string]string{"name": "i.am.a"},
									},
								},
							},
							Val: &gpb.TypedValue{
								Value: &gpb.TypedValue_StringVal{
									StringVal: "fancy.label",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GNMI{
				logger: zap.NewExample(),
				cfg: &Config{
					TargetName: "test-target",
					Sep:        "/",
					Origin:     "test-origin",
				},
			}

			lMap := pcommon.NewMap()
			if err := lMap.FromRaw(tc.inLabels); err != nil {
				t.Fatalf("failed to create label map: %v", err)
			}

			// Ensure order stability.
			sortProtos := cmpopts.SortSlices(func(m1, m2 *gpb.Notification) bool {
				return m1.String() < m2.String()
			})

			got := g.notificationsFromLabels(lMap, tc.name)

			// We don't rely on sortProtos to do the match, since (proto.Message).String() is not a stable
			// format, and this results in flakes. ygot's testutil provides a gNMI-specific checker (but
			// does not produce a diff, so we use cmp's to do that for us).
			if !testutil.NotificationSetEqual(got, tc.want, testutil.IgnoreTimestamp{}) {
				diff := cmp.Diff(tc.want, got, protocmp.Transform(), cmpopts.EquateEmpty(), sortProtos, protocmp.IgnoreFields(&gpb.Notification{}, "timestamp"))
				t.Errorf("notificationsFromLabels(%v) returned an unexpected diff (-want +got): %v", tc.inLabels, diff)
			}
		})
	}

}
