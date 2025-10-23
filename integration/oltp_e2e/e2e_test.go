package e2etest

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/openconfig/clio/collector"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	metricTimestamp = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
)

// newTs advances the metricTimestamp and returns a new timestamp.
func newTs() pcommon.Timestamp {
	metricTimestamp = metricTimestamp.Add(time.Minute)
	return pcommon.NewTimestampFromTime(metricTimestamp)
}

// subscribeRequestForTarget is a convenience function to create a target-specific SubscribeRequest.
func subscribeRequestForTarget(t *testing.T, target string) *gpb.SubscribeRequest {
	return &gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			Subscribe: &gpb.SubscriptionList{
				Prefix: &gpb.Path{
					Target: target,
					Elem: []*gpb.PathElem{
						{
							Name: "containers",
						},
						{
							Name: "container",
							Key:  map[string]string{"name": "test-container"},
						},
					},
					Origin: "clio",
				},
				Mode: gpb.SubscriptionList_STREAM,
				Subscription: []*gpb.Subscription{
					{
						Mode:              gpb.SubscriptionMode_ON_CHANGE,
						Path:              &gpb.Path{},
						SuppressRedundant: false,
					},
				},
			},
		},
	}
}

// startCollectorPipeline starts and returns the collector pipeline (plus the WaitGroup it runs in).
func startCollectorPipeline(t *testing.T, ctx context.Context) (*sync.WaitGroup, *otelcol.Collector) {
	t.Helper()
	t.Log("Starting collector pipeline")

	// Create collector.
	cps := otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{filepath.Join("config", "config.yaml")},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
			},
		},
	}
	set := otelcol.CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              collector.Components,
		ConfigProviderSettings: cps,
	}
	col, err := otelcol.NewCollector(set)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Start collector.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := col.Run(ctx); err != nil {
			t.Errorf("%v", err)
		}
	}()
	return wg, col
}

// stopCollectorPipeline stops the collector pipeline and waits for it to finish.
func stopCollectorPipeline(t *testing.T, wg *sync.WaitGroup, col *otelcol.Collector) {
	t.Helper()
	t.Log("Stopping collector pipeline")
	col.Shutdown()
	wg.Wait()
	t.Log("Stopped collector pipeline")
	if otelcol.StateClosed != col.GetState() {
		t.Errorf("got collector state %v, want %v", col.GetState(), otelcol.StateClosed)
	}
}

func TestE2E(t *testing.T) {
	// Number of metric waves to export. Based on the output of GenerateMetrics(...),
	// each wave should produce 20 notifications.
	exportWaves := 10

	gOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	ctx := context.Background()

	// Early declaration of wait group to ensure that defer order works.
	sinkWg := &sync.WaitGroup{}
	defer sinkWg.Wait()

	// Start collector pipeline & schedule stop.
	cwg, col := startCollectorPipeline(t, ctx)
	defer stopCollectorPipeline(t, cwg, col)

	// Get a gnmi client to subscribe to incoming notifications.
	gnmiConn, err := grpc.NewClient("localhost:6030", gOpts...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer gnmiConn.Close()
	gnmiClient := gpb.NewGNMIClient(gnmiConn)
	stream, err := gnmiClient.Subscribe(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Setup sink routine for incoming notifications.
	var gotNoti []*gpb.Notification
	sinkWg.Add(1)
	go func() {
		defer sinkWg.Done()
		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}

			if resp.GetUpdate() != nil {
				gotNoti = append(gotNoti, resp.GetUpdate())
			}
		}
	}()

	sreq := subscribeRequestForTarget(t, "poodle")
	err = stream.Send(sreq)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Get a gRPC client for exporting metrics.
	expConn, err := grpc.NewClient("localhost:4317", gOpts...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer expConn.Close()
	expClient := pmetricotlp.NewGRPCClient(expConn)

	// Generate & Export metrics.
	for i := 0; i < exportWaves; i++ {
		metrics := generateMetrics(t)
		req := pmetricotlp.NewExportRequestFromMetrics(metrics)
		_, err = expClient.Export(ctx, req)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}

	// Give collector some time to process the notifications.
	time.Sleep(1 * time.Second)

	// Count the number of received (but potentially coalesced) notifications.
	var notiCnt int
	for _, noti := range gotNoti {
		notiCnt++
		for _, update := range noti.GetUpdate() {
			notiCnt += int(update.GetDuplicates())
		}
	}

	if notiCnt != 20*exportWaves {
		t.Errorf("got %d (potentially coalesced) notification updates, want %d", notiCnt, 20*exportWaves)
	}
}

// generateMetrics generates the following Metrics structure:
//
// Resource (resource-0)
// -- (0) ScopeMetrics (resource-0-scope-0)
//
//	-- (0) Metrics
//	  -- ... <MetricsWithValue(0)> ...
//	-- (1) Metrics
//	  -- ... <MetricsWithValue(1)> ...
//
// -- (1) ScopeMetrics (resource-0-scope-1)
//
//	-- (0) Metrics
//	  -- ... <MetricsWithValue(10)> ...
//
// Resource (resource-1)
// -- (1) ScopeMetrics (resource-1-scope-0)
//
//	-- (0) Metrics
//	  -- ... <MetricsWithValue(100)> ...
func generateMetrics(t *testing.T) pmetric.Metrics {
	t.Helper()
	md := pmetric.NewMetrics()

	// Initialize resource.
	md.ResourceMetrics().AppendEmpty().Resource().Attributes().PutStr("resource-attr", "resource-0")
	md.ResourceMetrics().AppendEmpty().Resource().Attributes().PutStr("resource-attr", "resource-1")

	// Initialize scopes.
	md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty().Scope().SetName("resource-0-scope-0")
	md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty().Scope().SetName("resource-0-scope-1")
	md.ResourceMetrics().At(1).ScopeMetrics().AppendEmpty().Scope().SetName("resource-1-scope-0")

	// Ensure container names are set.
	md.ResourceMetrics().At(0).Resource().Attributes().PutStr("container.name", "test-container")
	md.ResourceMetrics().At(1).Resource().Attributes().PutStr("container.name", "test-container")

	// Initialize metrics.
	r0s0 := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	insertMetricsWithValue(t, r0s0, 0)
	insertMetricsWithValue(t, r0s0, 1)
	r0s1 := md.ResourceMetrics().At(0).ScopeMetrics().At(1).Metrics()
	insertMetricsWithValue(t, r0s1, 10)
	r1s0 := md.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics()
	insertMetricsWithValue(t, r1s0, 100)

	return md
}

// insertMetricsWithValue inserts a Gauge, Sum, Histogram, ExponentialHistogram and Summary metric
// containing the given value across various metric fields.
func insertMetricsWithValue(t *testing.T, slice pmetric.MetricSlice, value int64) {
	t.Helper()
	ts := newTs()

	// Insert Gauge metric.
	gm := slice.AppendEmpty()
	initMetric(gm, "mymetric/GaugeMetric", pmetric.MetricTypeGauge)
	gm.Gauge().DataPoints().AppendEmpty().SetIntValue(value)
	gm.Gauge().DataPoints().At(0).SetTimestamp(ts)

	// Insert Sum metric.
	sm := slice.AppendEmpty()
	initMetric(sm, "mymetric/SumMetric", pmetric.MetricTypeSum)
	sm.Sum().DataPoints().AppendEmpty().SetIntValue(value)
	sm.Sum().DataPoints().At(0).SetTimestamp(ts)

	// Insert Histogram metric.
	hm := slice.AppendEmpty()
	initMetric(hm, "mymetric/HistogramMetric", pmetric.MetricTypeHistogram)
	hdp := hm.Histogram().DataPoints().AppendEmpty()
	hdp.SetTimestamp(ts)
	hdp.SetStartTimestamp(ts)
	hdp.ExplicitBounds().EnsureCapacity(1)
	hdp.ExplicitBounds().Append(float64(value))
	hdp.BucketCounts().EnsureCapacity(2)
	hdp.BucketCounts().Append(0, uint64(value))
	hdp.SetCount(uint64(value))
	hdp.SetMax(float64(value))
	hdp.SetMin(float64(value))
	hdp.SetSum(float64(value))

	// Insert ExponentialHistogram metric.
	ehm := slice.AppendEmpty()
	initMetric(ehm, "mymetric/ExponentialHistogramMetric", pmetric.MetricTypeExponentialHistogram)
	ehdp := ehm.ExponentialHistogram().DataPoints().AppendEmpty()
	ehdp.SetTimestamp(ts)
	ehdp.SetStartTimestamp(ts)
	ehdp.SetScale(int32(value))
	ehdp.Negative().BucketCounts().EnsureCapacity(1)
	ehdp.Negative().BucketCounts().Append(uint64(value))
	ehdp.SetCount(uint64(value))
	ehdp.SetMax(float64(value))
	ehdp.SetMin(float64(value))
	ehdp.SetSum(float64(value))

	// Insert Summary metric.
	smm := slice.AppendEmpty()
	initMetric(smm, "mymetric/SummaryMetric", pmetric.MetricTypeSummary)
	smmdp := smm.Summary().DataPoints().AppendEmpty()
	smmdp.SetTimestamp(ts)
	smmdp.SetStartTimestamp(ts)
	qv0 := smmdp.QuantileValues().AppendEmpty()
	qv0.SetQuantile(0.95)
	qv0.SetValue(float64(value))
}

// initMetric initializes the given metric with the given name and type.
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
