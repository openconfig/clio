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
	"testing"
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
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantCnt: 1,
		},
		{
			name: "mtls",
			cfg: &Config{
				CaFile:   caFile,
				CertFile: certFile,
				KeyFile:  keyFile,
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
				CertFile: "testdata/capybara-stole-this-cert.pem",
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent client certificate",
		},
		{
			name: "tls-nonexist-key",
			cfg: &Config{
				CertFile: certFile,
				KeyFile:  "testdata/capybara-stole-this-key.pem",
			},
			errMsg: "for nonexistent client private key",
		},
		{
			name: "mtls-nonexist-ca-cert",
			cfg: &Config{
				CaFile:   "testdata/capybara-stole-this-ca-cert.crt",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent CA certificate",
		},
		{
			name: "mtls-nonexist-cli-cert",
			cfg: &Config{
				CaFile:   caFile,
				CertFile: "testdata/capybara-stole-this-cert.pem",
				KeyFile:  keyFile,
			},
			errMsg: "for nonexistent client certificate",
		},
		{
			name: "mtls-nonexist-key",
			cfg: &Config{
				CaFile:   caFile,
				CertFile: certFile,
				KeyFile:  "testdata/capybara-stole-this-key.pem",
			},
			errMsg: "for nonexistent client private key",
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
