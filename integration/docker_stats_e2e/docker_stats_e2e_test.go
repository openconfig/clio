package e2etest

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/openconfig/clio/collector"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// configProviderSettings is a convenience function to create ConfigProviderSettings that use the
// local config.yaml.
func configProviderSettings() otelcol.ConfigProviderSettings {
	return otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{filepath.Join("./config", "config.yaml")},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
			},
		},
	}
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
							Name: target,
						},
					},
				},
				Mode: gpb.SubscriptionList_STREAM,
				Subscription: []*gpb.Subscription{
					&gpb.Subscription{
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
	set := otelcol.CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              collector.Components,
		ConfigProviderSettings: configProviderSettings(),
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
			t.Fatalf("%v", err)
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

// validateNotifications validates that the collector's exported notification stream contain a
// selection of the configured paths.
func validateNotifications(t *testing.T, gotNoti []*gpb.Notification) {
	t.Helper()

	elems2path := func(elems []*gpb.PathElem) string {
		subs := []string{}
		for _, e := range elems {
			subs = append(subs, e.GetName())
		}
		return strings.Join(subs, ".")
	}

	// Map containing some of the enabled paths for each "logical metric group."
	wantPathSet := map[string]bool{
		"container.uptime":                      true,
		"container.restarts":                    true,
		"container.cpu.usage.kernelmode":        true,
		"container.cpu.usage.total":             true,
		"container.cpu.usage.usermode":          true,
		"container.memory.file":                 true,
		"container.memory.percent":              true,
		"container.memory.usage.total":          true,
		"container.cpu.utilization":             true,
		"container.cpu.logical.count":           true,
		"container.cpu.shares":                  true,
		"container.memory.usage.limit":          true,
		"container.pids.limit":                  true,
		"container.network.io.usage.rx_bytes":   true,
		"container.network.io.usage.rx_dropped": true,
		"container.network.io.usage.tx_bytes":   true,
	}

	for _, n := range gotNoti {
		for _, u := range n.GetUpdate() {
			path := elems2path(u.GetPath().GetElem())
			if _, ok := wantPathSet[path]; ok {
				delete(wantPathSet, path)
			}
		}
	}

	if len(wantPathSet) > 0 {
		t.Errorf("Some paths were not found in the notifications:\n")
		for p := range wantPathSet {
			t.Errorf("  %s", p)
		}
	}
}

func TestE2E(t *testing.T) {

	gOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	ctx := context.Background()

	// Early declaration of validation function & wait group to ensure that defer order works.
	var gotNoti []*gpb.Notification
	sinkWg := &sync.WaitGroup{}
	defer sinkWg.Wait()

	// Start collector pipeline & schedule stop.
	cwg, col := startCollectorPipeline(t, ctx)
	defer stopCollectorPipeline(t, cwg, col)

	// Wait for collector to be started.
	for i := 15; col.GetState() != otelcol.StateRunning; i-- {
		if i == 0 {
			t.Fatalf("Collector never started")
		}
		time.Sleep(200 * time.Millisecond)
	}

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

	// Give collector some time to process the notifications.
	time.Sleep(10 * time.Second)

	validateNotifications(t, gotNoti)
}
