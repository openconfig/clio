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
	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for this processor.
type Config struct {
	// Addr is  the listen address of the gNMI server.
	Addr string `mapstructure:"addr"`

	// CaFile is the CA certificate to use for mTLS.
	CaFile string `mapstructure:"ca_file"`

	// CertFile is the certificate to use for TLS.
	CertFile string `mapstructure:"cert_file"`

	// KeyFile is the key to use for TLS.
	KeyFile string `mapstructure:"key_file"`

	// TargetName is the target name of this gNMI server.
	TargetName string `mapstructure:"target_name"`

	// BufferSize is the buffer depth to use for internal buffering.
	BufferSize int `mapstructure:"buffer_size"`

	// Sep is the separator used in the metric name.
	Sep string `mapstructure:"sep"`
}

var _ component.Config = (*Config)(nil)
