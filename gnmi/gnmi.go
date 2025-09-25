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
	"fmt"
	"net"
	"strings"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	ompb "go.opentelemetry.io/proto/otlp/metrics/v1"
	anypb "google.golang.org/protobuf/types/known/anypb"

	"github.com/openconfig/magna/lwotgtelem"
	"github.com/openconfig/magna/lwotgtelem/gnmit"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"k8s.io/klog/v2"
)

// GNMI is a gNMI exporter.
type GNMI struct {
	cfg *Config

	lis      net.Listener
	srv      *grpc.Server
	metricCh chan *pmetric.Metrics
	telemSrv *lwotgtelem.Server

	logger *zap.Logger
}

// NewGNMIExporter creates a new gNMI exporter.
func NewGNMIExporter(logger *zap.Logger, cfg *Config) (*GNMI, error) {
	telemSrv, err := lwotgtelem.New(context.Background(), cfg.TargetName)
	if err != nil {
		return nil, err
	}

	gnmiLis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		klog.Exitf("cannot listen on %s, err: %v", cfg.Addr, err)
	}

	var opts []grpc.ServerOption
	opt, err := gRPCSecurityOption(cfg)
	if err != nil {
		return nil, err
	}
	opts = append(opts, opt...)

	return &GNMI{
		cfg:      cfg,
		metricCh: make(chan *pmetric.Metrics, cfg.BufferSize),
		lis:      gnmiLis,
		srv:      grpc.NewServer(opts...),
		telemSrv: telemSrv,
		logger:   logger,
	}, nil
}

// Start starts the gNMI exporter.
func (g *GNMI) Start(_ context.Context, _ component.Host) error {
	g.logger.Info("starting gNMI exporter", zap.String("addr", g.cfg.Addr),
		zap.String("target", g.cfg.TargetName), zap.String("origin", g.cfg.Origin),
		zap.String("sep", g.cfg.Sep), zap.Int("buffer_size", g.cfg.BufferSize),
		zap.String("transport_security", g.cfg.TpSec),
	)
	reflection.Register(g.srv)
	gpb.RegisterGNMIServer(g.srv, g.telemSrv.GNMIServer)
	go g.srv.Serve(g.lis)
	g.telemSrv.AddTask(gnmit.Task{
		Run: g.handleMetrics,
	})
	g.logger.Info("started")
	return nil
}

// Stop stops the gNMI exporter.
func (g *GNMI) Stop(_ context.Context) error {
	close(g.metricCh)
	g.srv.GracefulStop()
	return nil
}

func (g *GNMI) storeMetric(ctx context.Context, md pmetric.Metrics) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		g.metricCh <- &md
	}
	return nil
}

func typedValueFromHistogramDataPoint(p *pmetric.HistogramDataPoint) (*gpb.TypedValue, error) {
	hp := &ompb.HistogramDataPoint{
		Count:          p.Count(),
		BucketCounts:   p.BucketCounts().AsRaw(),
		ExplicitBounds: p.ExplicitBounds().AsRaw(),
	}

	if p.HasSum() {
		hp.Sum = proto.Float64(p.Sum())
	}
	if p.HasMin() {
		hp.Min = proto.Float64(p.Min())
	}
	if p.HasMax() {
		hp.Max = proto.Float64(p.Max())
	}
	any, err := anypb.New(hp)
	if err != nil {
		return nil, err
	}

	return &gpb.TypedValue{
		Value: &gpb.TypedValue_AnyVal{
			AnyVal: any,
		},
	}, nil
}

func typedValueFromExponentialHistogramDataPoint(p *pmetric.ExponentialHistogramDataPoint) (*gpb.TypedValue, error) {
	hp := &ompb.ExponentialHistogramDataPoint{
		Count: p.Count(),
		Scale: p.Scale(),
		Positive: &ompb.ExponentialHistogramDataPoint_Buckets{
			BucketCounts: p.Positive().BucketCounts().AsRaw(),
			Offset:       p.Positive().Offset(),
		},
		Negative: &ompb.ExponentialHistogramDataPoint_Buckets{
			BucketCounts: p.Negative().BucketCounts().AsRaw(),
			Offset:       p.Negative().Offset(),
		},
		ZeroThreshold: p.ZeroThreshold(),
		ZeroCount:     p.ZeroCount(),
	}

	if p.HasSum() {
		hp.Sum = proto.Float64(p.Sum())
	}
	if p.HasMin() {
		hp.Min = proto.Float64(p.Min())
	}
	if p.HasMax() {
		hp.Max = proto.Float64(p.Max())
	}
	any, err := anypb.New(hp)
	if err != nil {
		return nil, err
	}

	return &gpb.TypedValue{
		Value: &gpb.TypedValue_AnyVal{
			AnyVal: any,
		},
	}, nil
}

func typedValueFromSummaryDataPoint(p *pmetric.SummaryDataPoint) (*gpb.TypedValue, error) {
	qval := make([]*ompb.SummaryDataPoint_ValueAtQuantile, 0, p.QuantileValues().Len())
	for i := 0; i < p.QuantileValues().Len(); i++ {
		qval = append(qval, &ompb.SummaryDataPoint_ValueAtQuantile{
			Quantile: p.QuantileValues().At(i).Quantile(),
			Value:    p.QuantileValues().At(i).Value(),
		})
	}

	sp := &ompb.SummaryDataPoint{
		Count:          p.Count(),
		QuantileValues: qval,
	}

	any, err := anypb.New(sp)
	if err != nil {
		return nil, err
	}

	return &gpb.TypedValue{
		Value: &gpb.TypedValue_AnyVal{
			AnyVal: any,
		},
	}, nil
}

type simpleNumberDataPoint interface {
	ValueType() pmetric.NumberDataPointValueType
	IntValue() int64
	DoubleValue() float64
}

func typedValueFromNumericMetric(p simpleNumberDataPoint) *gpb.TypedValue {
	switch p.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return &gpb.TypedValue{
			Value: &gpb.TypedValue_IntVal{
				IntVal: p.IntValue(),
			},
		}
	case pmetric.NumberDataPointValueTypeDouble:
		return &gpb.TypedValue{
			Value: &gpb.TypedValue_DoubleVal{
				DoubleVal: p.DoubleValue(),
			},
		}
	default:
		return nil
	}
}

// attrMap is convenience definition for a map of attribute names to values for use in gNMI
// notifications.
type attrMap map[string]string

// convO2GMap converts an OpenTelemetry map to a gNMI map.
func convO2GMap(attrs pcommon.Map) attrMap {
	if attrs.Len() == 0 {
		return nil
	}

	gMap := make(attrMap)
	attrs.Range(func(k string, v pcommon.Value) bool {
		gMap[k] = v.AsString()
		return true
	})
	return gMap
}

// typedValuesFromMetric returns a list of typed values based on a metric's values.
func (g *GNMI) typedValuesAndTimesFromMetric(m pmetric.Metric) ([]*gpb.TypedValue, []pcommon.Timestamp, []attrMap) {
	var tvals []*gpb.TypedValue
	var tstamps []pcommon.Timestamp
	var attrs []attrMap

	switch m.Type() {
	case pmetric.MetricTypeEmpty:
		g.logger.Error("empty metric type", zap.String("name", m.Name()))
		return nil, nil, nil
	case pmetric.MetricTypeGauge:
		gaugeMetrics := m.Gauge().DataPoints()
		for l := 0; l < gaugeMetrics.Len(); l++ {
			gaugeMetric := gaugeMetrics.At(l)
			val := typedValueFromNumericMetric(gaugeMetric)
			if val == nil {
				g.logger.Error("found numeric metric value with type other than int or double", zap.String("name", m.Name()))
				continue
			}
			tvals = append(tvals, val)
			tstamps = append(tstamps, gaugeMetric.Timestamp())
			attrs = append(attrs, convO2GMap(gaugeMetric.Attributes()))
		}
	case pmetric.MetricTypeSum:
		sumMetrics := m.Sum().DataPoints()
		for l := 0; l < sumMetrics.Len(); l++ {
			sumMetric := sumMetrics.At(l)
			val := typedValueFromNumericMetric(sumMetric)
			if val == nil {
				g.logger.Error("found numeric metric value with type other than int or double", zap.String("name", m.Name()))
				continue
			}
			tvals = append(tvals, val)
			tstamps = append(tstamps, sumMetric.Timestamp())
			attrs = append(attrs, convO2GMap(sumMetric.Attributes()))
		}
	case pmetric.MetricTypeHistogram:
		histMetrics := m.Histogram().DataPoints()
		for l := 0; l < histMetrics.Len(); l++ {
			histMetric := histMetrics.At(l)
			val, err := typedValueFromHistogramDataPoint(&histMetric)
			if err != nil {
				g.logger.Error("failed to marshal histogram metric", zap.String("name", m.Name()), zap.Error(err))
				continue
			}
			tvals = append(tvals, val)
			tstamps = append(tstamps, histMetric.Timestamp())
			attrs = append(attrs, convO2GMap(histMetric.Attributes()))
		}
	case pmetric.MetricTypeExponentialHistogram:
		histMetrics := m.ExponentialHistogram().DataPoints()
		for l := 0; l < histMetrics.Len(); l++ {
			histMetric := histMetrics.At(l)
			val, err := typedValueFromExponentialHistogramDataPoint(&histMetric)
			if err != nil {
				g.logger.Error("failed to marshal exponential histogram metric", zap.String("name", m.Name()), zap.Error(err))
				continue
			}
			tvals = append(tvals, val)
			tstamps = append(tstamps, histMetric.Timestamp())
			attrs = append(attrs, convO2GMap(histMetric.Attributes()))
		}
	case pmetric.MetricTypeSummary:
		sumMetrics := m.Summary().DataPoints()
		for l := 0; l < sumMetrics.Len(); l++ {
			sumMetric := sumMetrics.At(l)
			val, err := typedValueFromSummaryDataPoint(&sumMetric)
			if err != nil {
				g.logger.Error("failed to marshal summary metric", zap.String("name", m.Name()), zap.Error(err))
				continue
			}
			tvals = append(tvals, val)
			tstamps = append(tstamps, sumMetric.Timestamp())
			attrs = append(attrs, convO2GMap(sumMetric.Attributes()))
		}
	default:
		g.logger.Error("unsupported metric type", zap.String("name", m.Name()), zap.String("type", m.Type().String()))
	}
	return tvals, tstamps, attrs
}

// notificationsFromMetric returns a list of gNMI notifications based on a metric.
func (g *GNMI) notificationsFromMetric(p pmetric.Metric, container string) []*gpb.Notification {
	var notis []*gpb.Notification
	values, timestamps, attrs := g.typedValuesAndTimesFromMetric(p)
	if len(values) == 0 {
		return nil
	}

	for i, val := range values {
		// Some leaves are dependant on the attributes of the metric.
		elems := g.toPathElems(p.Name())
		if attrs[i] != nil {
			elems[len(elems)-1].Key = attrs[i]
		}

		notis = append(notis, &gpb.Notification{
			Timestamp: timestamps[i].AsTime().UnixNano(),
			Prefix: &gpb.Path{
				Origin: g.cfg.Origin,
				Target: g.cfg.TargetName,
				Elem: []*gpb.PathElem{
					{
						Name: "containers",
					},
					{
						Name: "container",
						Key:  map[string]string{"name": container},
					},
				},
			},
			Update: []*gpb.Update{
				{
					Path: &gpb.Path{
						Target: g.cfg.TargetName,
						Elem:   elems,
					},
					Val: val,
				},
			},
		})
	}
	return notis
}

// notificationsFromLabels returns a list of gNMI notifications based on a map of labels. Given that
// the labels are fetched from the same dockerstatereceiver.scrapev2 call as the other container
// metrics, we can draft most notification fields --- e.g., timestamps and prefixes --- from a
// reference notification.
func (g *GNMI) notificationsFromLabels(m pcommon.Map, cname string) []*gpb.Notification {
	var notis []*gpb.Notification

	for k, v := range m.All() {
		notis = append(notis, &gpb.Notification{
			Timestamp: time.Now().UnixNano(),
			Prefix: &gpb.Path{
				Origin: g.cfg.Origin,
				Target: g.cfg.TargetName,
				Elem: []*gpb.PathElem{
					{
						Name: "containers",
					},
					{
						Name: "container",
						Key:  map[string]string{"name": cname},
					},
				},
			},
			Update: []*gpb.Update{
				{
					Path: &gpb.Path{
						Target: g.cfg.TargetName,
						Elem: []*gpb.PathElem{
							{
								Name: "labels",
							},
							{
								Name: "label",
								Key:  map[string]string{"name": k},
							},
						},
					},
					Val: &gpb.TypedValue{
						Value: &gpb.TypedValue_StringVal{
							StringVal: v.AsString(),
						},
					},
				},
			},
		})
	}
	return notis
}

// handleMetrics iterates over all received metrics and converts them into a
// gNMI update. This set of updates are then packed into a gNMI notification
// and sent to the telemetry server.
func (g *GNMI) handleMetrics(_ gnmit.Queue, updateFn gnmit.UpdateFn, target string, cleanup func()) error {
	go func() {
		for ms := range g.metricCh {
			var notis []*gpb.Notification

			// Iterate over all resources (e.g., app).
			rms := ms.ResourceMetrics()
			for i := 0; i < rms.Len(); i++ {
				rm := rms.At(i)
				cname := ""

				// Extract container name from resource, if not found, log error and continue.
				cNameVal, ok := rm.Resource().Attributes().Get("container.name")
				if ok && cNameVal.Type() == pcommon.ValueTypeStr {
					cname = cNameVal.Str()
				} else {
					g.logger.Error("resource is not associated with a container name formatted as a string", zap.String("resource", fmt.Sprintf("%+v", rm.Resource().Attributes().AsRaw())))
					continue
				}

				// Iterate over all instrument scopes within the resource (e.g., module within an app).
				ilms := rm.ScopeMetrics()
				for j := 0; j < ilms.Len(); j++ {
					ilm := ilms.At(j)

					// Iterate over all metrics for the instrument scope.
					ms := ilm.Metrics()
					for k := 0; k < ms.Len(); k++ {
						m := ms.At(k)
						notis = append(notis, g.notificationsFromMetric(m, cname)...)
					}
				}

				// Obtain notifications for labels.
				lmap, ok := rm.Resource().Attributes().Get("container.labels")
				if ok && lmap.Type() == pcommon.ValueTypeMap && len(notis) > 0 {
					notis = append(notis, g.notificationsFromLabels(lmap.Map(), cname)...)
				}
			}

			// Send all notifications.
			for _, notification := range notis {
				if err := updateFn(notification); err != nil {
					klog.Errorf("failed to send notification: %v", err)
				}
			}
		}
	}()
	return nil
}

func (g *GNMI) toPathElems(name string) []*gpb.PathElem {
	var elems []*gpb.PathElem
	for i, p := range strings.Split(name, g.cfg.Sep) {
		if i == 0 && p == "container" {
			continue
		}
		elems = append(elems, &gpb.PathElem{
			Name: p,
			// TODO (alshabib): support keyed paths.
		})
	}
	return elems
}
