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

	"github.com/openconfig/magna/lwotgtelem"
	"github.com/openconfig/magna/lwotgtelem/gnmit"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

type GNMI struct {
	cfg *Config

	lis      net.Listener
	srv      *grpc.Server
	metricCh chan *pmetric.Metrics
	telemSrv *lwotgtelem.Server

	logger *zap.Logger
}

func NewGNMIExporter(logger *zap.Logger, cfg *Config) (*GNMI, error) {
	telemSrv, err := lwotgtelem.New(context.Background(), cfg.TargetName)
	if err != nil {
		return nil, err
	}

	gnmiLis, err := net.Listen("tcp", fmt.Sprintf("%s", cfg.Addr))
	if err != nil {
		klog.Exitf("cannot listen on %s, err: %v", cfg.Addr, err)
	}

	var opts []grpc.ServerOption

	if cfg.CertFile != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("cannot create gNMI credentials, %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	return &GNMI{
		cfg:      cfg,
		metricCh: make(chan *pmetric.Metrics, cfg.BufferSize),
		lis:      gnmiLis,
		srv:      grpc.NewServer(opts...),
		telemSrv: telemSrv,
		logger:   logger,
	}, nil
}

func (g *GNMI) Start(_ context.Context, _ component.Host) error {
	g.logger.Info("starting")
	reflection.Register(g.srv)
	gpb.RegisterGNMIServer(g.srv, g.telemSrv.GNMIServer)
	go g.srv.Serve(g.lis)
	g.telemSrv.AddTask(gnmit.Task{
		Run: g.handleMetrics,
	})
	g.logger.Info("started")
	return nil
}

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

// handleMetrics iterates over all received metrics and converts them into a
// gNMI update. This set of updates are then packed into a gNMI notfication
// and sent to the telemetry server.
// Note: this currently supports only SUM metrics.
func (g *GNMI) handleMetrics(_ gnmit.Queue, updateFn gnmit.UpdateFn, target string, cleanup func()) error {
	go func() {
		for ms := range g.metricCh {
			var updates []*gpb.Update
			rms := ms.ResourceMetrics()
			for i := 0; i < rms.Len(); i++ {
				rm := rms.At(i)
				ilms := rm.ScopeMetrics()
				for j := 0; j < ilms.Len(); j++ {
					ilm := ilms.At(j)
					ms := ilm.Metrics()
					for k := 0; k < ms.Len(); k++ {
						m := ms.At(k)
						switch m.Type() {
						case pmetric.MetricTypeSum:
							sumMetrics := m.Sum().DataPoints()
							for l := 0; l < sumMetrics.Len(); l++ {
								sumMetric := sumMetrics.At(l)
								updates = append(updates, &gpb.Update{
									Path: &gpb.Path{
										Elem:   g.toPathElems(m.Name()),
										Target: g.cfg.TargetName,
									},
									Val: &gpb.TypedValue{
										Value: &gpb.TypedValue_IntVal{
											IntVal: sumMetric.IntValue(),
										},
									},
								})
							}
						}
					}
				}
			}

			if err := updateFn(&gpb.Notification{
				Timestamp: time.Now().Unix(),
				Prefix: &gpb.Path{
					Target: g.cfg.TargetName,
					Elem: []*gpb.PathElem{
						{
							Name: g.cfg.TargetName,
						},
					},
				},
				Update: updates,
			}); err != nil {
				klog.Errorf("failed to send updates %v", err)
			}
		}

	}()
	return nil
}

func (g *GNMI) toPathElems(name string) []*gpb.PathElem {
	var elems []*gpb.PathElem
	for _, p := range strings.Split(name, g.cfg.Sep) {
		elems = append(elems, &gpb.PathElem{
			Name: p,
			// TODO (alshabib): support keyed paths.
		})
	}
	return elems
}
