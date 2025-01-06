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
	"errors"
	//"strings"
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
)

var (
	certFile = "testdata/cert.pem"
	keyFile  = "testdata/private.pem"
	caFile   = "testdata/gnmi.example.com.crt"
)

func TestGRPCSecurityOption(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantCnt int
	}{
		{
			name: "no-tls",
			cfg:  &Config{},
		},
		{
			name: "tls",
			cfg: &Config{
				TpSec:    "tls",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantCnt: 1,
		},
		{
			name: "mtls",
			cfg: &Config{
				TpSec:    "mtls",
				CAFile:   caFile,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantCnt: 1,
		},
		{
			name: "alts",
			cfg: &Config{
				TpSec: "alts",
			},
			wantCnt: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := gRPCSecurityOption(tc.cfg)
			if err != nil {
				t.Errorf("gRPCSecurityOption returned error: %v", err)
			}
			if len(got) != tc.wantCnt {
				t.Errorf("gRPCSecurityOption returned %d options, want %d", len(got), tc.wantCnt)
			}
		})
	}
}

func TestGRPCSecurityOptionErrors(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		errMsg string
	}{
		{
			name: "tls-nonexist-cert",
			cfg: &Config{
				TpSec:    "tls",
				CertFile: "testdata/capybara-stole-this-cert.pem",
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent client certificate",
		},
		{
			name: "tls-nonexist-key",
			cfg: &Config{
				TpSec:    "tls",
				CertFile: certFile,
				KeyFile:  "testdata/capybara-stole-this-key.pem",
			},
			errMsg: "for nonexistent client private key",
		},
		{
			name: "mtls-nonexist-ca-cert",
			cfg: &Config{
				TpSec:    "mtls",
				CAFile:   "testdata/capybara-stole-this-ca-cert.crt",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent CA certificate",
		},
		{
			name: "mtls-nonexist-cli-cert",
			cfg: &Config{
				TpSec:    "mtls",
				CAFile:   caFile,
				CertFile: "",
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent client certificate",
		},
		{
			name: "mtls-nonexist-key",
			cfg: &Config{
				TpSec:    "mtls",
				CAFile:   caFile,
				CertFile: certFile,
				KeyFile:  "testdata/capybara-stole-this-key.pem",
			},
			errMsg: "for nonexistent client private key",
		},
		{
			name: "mtls-refresh-duration-error",
			cfg: &Config{
				TpSec:        "mtls",
				CAFile:       caFile,
				CertFile:     certFile,
				KeyFile:      keyFile,
				CredsRefresh: "1095.75fortnights",
			},
			errMsg: "unknown unit \"fortnights\" in duration \"1095.75fortnights\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := gRPCSecurityOption(tc.cfg)
			if err == nil {
				t.Errorf("gRPCSecurityOption did not return error %v", tc.errMsg)
			}

		})
	}
}

// TestALTSConnection tests that we can connect to a gNMI server using ALTS.
func TestALTSConnection(t *testing.T) {
	// Set up ALTS config & logger.
	cfg := &Config{
		TpSec: "alts",
	}
	logger, err := zap.NewProduction()
	if err != nil {
		t.Errorf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Start the exporter.
	exporter, err := NewGNMIExporter(logger, cfg)
	if err != nil {
		t.Errorf("NewGNMIExporter returned error: %v", err)
	}
	if err := exporter.Start(context.Background(), nil); err != nil {
		t.Errorf("Start returned error: %v", err)
	}
	defer exporter.Stop(context.Background())

	// Connect to the exporter.
	addr := exporter.lis.Addr()
	altsTC := alts.NewClientCreds(alts.DefaultClientOptions())
	conn, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(altsTC))
	if err != nil {
		t.Errorf("failed to connect to gNMI server: %v", err)
	}
	defer conn.Close()

	// Subscribe to the exporter and see whether we get an error.
	gnmiClient := gpb.NewGNMIClient(conn)
	if _, err = gnmiClient.Subscribe(context.Background()); err != nil {
		// If we are not running on a GCE platform, ALTS does not work.
		if errors.Is(err, alts.ErrUntrustedPlatform) {
			// if strings.HasSuffix(err.Error(), alts.ErrUntrustedPlatform.Error()) {
			return
		}
		t.Fatalf("%v", err)
	}
}
