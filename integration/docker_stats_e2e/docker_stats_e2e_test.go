package e2etest

import (
	"context"
	"github.com/openconfig/clio/collector"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testTarget = "poodle"
	testOrigin = "shiba"
)

func spawnNginxContainer(t *testing.T) (ctr testcontainers.Container, cleanup func()) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "docker.io/library/nginx:1.17",
		ExposedPorts: []string{"80/tcp"},
		Labels:       map[string]string{"app": "nginx", "version": "1.17"},
		WaitingFor:   wait.ForExposedPort(),
	}

	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to create nginx container for test: %v", err)
	}

	f := func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			t.Errorf("failed to terminate container: %s", err)
		}
	}

	return container, f
}

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
					Origin: testOrigin,
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
func startCollectorPipeline(ctx context.Context, t *testing.T) (*sync.WaitGroup, *otelcol.Collector) {
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

// validateNotifications validates that the collector's exported notification stream contain a
// selection of the configured paths.
func validateNotifications(t *testing.T, gotNoti []*gpb.Notification) {
	t.Helper()

	elems2path := func(elems []*gpb.PathElem) string {
		var subs []string
		for _, e := range elems {
			subs = append(subs, e.GetName())
		}
		return strings.Join(subs, ".")
	}

	// Map containing some of the enabled paths for each "logical metric group."
	wantPathSet := map[string]bool{
		"uptime":                      true,
		"restarts":                    true,
		"cpu.usage.kernelmode":        true,
		"cpu.usage.total":             true,
		"cpu.usage.usermode":          true,
		"memory.file":                 true,
		"memory.percent":              true,
		"memory.usage.total":          true,
		"cpu.utilization":             true,
		"cpu.logical.count":           true,
		"cpu.shares":                  true,
		"memory.usage.limit":          true,
		"pids.limit":                  true,
		"network.io.usage.rx_bytes":   true,
		"network.io.usage.rx_dropped": true,
		"network.io.usage.tx_bytes":   true,
	}

	for _, n := range gotNoti {
		if len(n.GetDelete()) > 0 {
			t.Errorf("Unexpected delete notification received: %v", n)
		}
		
		for _, u := range n.GetUpdate() {
			path := elems2path(u.GetPath().GetElem())
			delete(wantPathSet, path)
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
	_, cleanup := spawnNginxContainer(t)
	defer cleanup()

	gOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	ctx := context.Background()

	// Early declaration of validation function & wait group to ensure that defer order works.
	var gotNoti []*gpb.Notification
	sinkWg := &sync.WaitGroup{}
	defer sinkWg.Wait()

	// Start collector pipeline & schedule stop.
	cwg, col := startCollectorPipeline(ctx, t)
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

	sreq := subscribeRequestForTarget(t, testTarget)
	err = stream.Send(sreq)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Give collector some time to process the notifications.
	time.Sleep(5 * time.Second)

	validateNotifications(t, gotNoti)
}

func getContainerName(path *gpb.Path) string {
	for _, elem := range path.Elem {
		if elem.Name == "container" {
			return elem.Key["name"]
		}
	}
	return ""
}

func TestE2ETTL(t *testing.T) {
	container, cleanup := spawnNginxContainer(t)

	gOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	ctx := context.Background()

	var gotNoti []*gpb.Notification
	var notiMu sync.Mutex
	sinkWg := &sync.WaitGroup{}
	defer sinkWg.Wait()

	cwg, col := startCollectorPipeline(ctx, t)
	defer stopCollectorPipeline(t, cwg, col)

	for i := 15; col.GetState() != otelcol.StateRunning; i-- {
		if i == 0 {
			t.Fatalf("Collector never started")
		}
		time.Sleep(200 * time.Millisecond)
	}

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

	sinkWg.Add(1)
	go func() {
		defer sinkWg.Done()
		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}
			if resp.GetUpdate() != nil {
				notiMu.Lock()
				gotNoti = append(gotNoti, resp.GetUpdate())
				notiMu.Unlock()
			}
		}
	}()

	sreq := subscribeRequestForTarget(t, testTarget)
	err = stream.Send(sreq)
	if err != nil {
		t.Fatalf("%v", err)
	}

	containerName, err := container.Name(ctx)
	if err != nil {
		t.Fatalf("Failed to get container name: %v", err)
	}
	// containerName from testcontainers usually has a leading slash, e.g. /reaper_...
	trimmedName := strings.TrimPrefix(containerName, "/")

	// Wait for at least one update for the container to ensure it's registered.
	updateFound := false
	for i := 0; i < 50; i++ {
		notiMu.Lock()
		for _, n := range gotNoti {
			if len(n.Update) > 0 {
				if strings.TrimPrefix(getContainerName(n.Prefix), "/") == trimmedName {
					updateFound = true
					break
				}
			}
		}
		notiMu.Unlock()
		if updateFound {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !updateFound {
		t.Fatalf("Timed out waiting for initial update for container %s", trimmedName)
	}

	// 2. Kill the container to trigger the death condition
	cleanup()

	// 3. Wait for the TTL (config is set to 4s) to expire + sweeper ticker to pick it up.
	deleteFound := false

	for i := 0; i < 75; i++ {
		notiMu.Lock()
		for _, n := range gotNoti {
			if len(n.Delete) > 0 {
				if n.Prefix.GetTarget() != testTarget {
					continue
				}
				if n.Prefix.GetOrigin() != testOrigin {
					continue
				}

				// Verify the container name is in the delete path
				gotName := getContainerName(n.Delete[0])
				if strings.TrimPrefix(gotName, "/") == trimmedName {
					deleteFound = true
					break
				}
			}
		}
		if deleteFound {
			break
		}
		notiMu.Unlock()
		time.Sleep(200 * time.Millisecond)
	}

	if !deleteFound {
		t.Errorf("Expected a container delete notification from the integration pipeline after killing container %v, but none was sent", containerName)
	}
}
