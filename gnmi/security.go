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
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/security/advancedtls"
)

const (
	// defaultCredsRefreshDuration is the default refresh duration for the credentials.
	defaultCredsRefreshDuration = time.Hour
)

func credsRefreshDuration(cfg *Config) (time.Duration, error) {
	if cfg.CredsRefresh == "" {
		return defaultCredsRefreshDuration, nil
	}
	return time.ParseDuration(cfg.CredsRefresh)
}

// gRPCSecurityOption returns a gRPC server option based on the transport security
// configuration.
func gRPCSecurityOption(cfg *Config) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption
	var err error

	switch cfg.TpSec {
	case "", "insecure": // No security option requested.
	case "tls":
		opts, err = optionTLS(cfg)
	case "mtls":
		opts, err = optionMutualTLS(cfg)
	default:
		return nil, fmt.Errorf("unsupported transport security: %q; must be one of: insecure, tls, mtls", cfg.TpSec)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create gNMI credentials: %v", err)
	}

	return opts, nil
}

func optionTLS(cfg *Config) ([]grpc.ServerOption, error) {
	// Check that all needed files actually exist.
	for _, f := range []string{cfg.CertFile, cfg.KeyFile} {
		if _, err := os.Stat(f); f == "" || errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file %q does not exist", f)
		}
	}

	creds, err := credentials.NewServerTLSFromFile(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	return []grpc.ServerOption{grpc.Creds(creds)}, nil
}

func optionMutualTLS(cfg *Config) ([]grpc.ServerOption, error) {

	// Check that all needed files actually exist.
	for _, f := range []string{cfg.CertFile, cfg.KeyFile, cfg.CAFile} {
		if _, err := os.Stat(f); f == "" || errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file %q does not exist", f)
		}
	}

	// Determine the refresh duration.
	crd, err := credsRefreshDuration(cfg)
	if err != nil {
		return nil, err
	}

	// Get a provider for the identity credentials.
	identity := pemfile.Options{
		CertFile:        cfg.CertFile,
		KeyFile:         cfg.KeyFile,
		RefreshDuration: crd,
	}

	identityProvider, err := pemfile.NewProvider(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %v", err)
	}

	// Get a provider for the root credentials.
	root := pemfile.Options{
		RootFile:        cfg.CAFile,
		RefreshDuration: crd,
	}
	rootProvider, err := pemfile.NewProvider(root)
	if err != nil {
		return nil, fmt.Errorf("failed to create root provider: %v", err)
	}

	// Setup the mTLS option.
	options := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: identityProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootProvider: rootProvider,
		},
		RequireClientCert: true,
	}

	// Setup the server credentials.
	creds, err := advancedtls.NewServerCreds(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create client creds: %v", err)
	}

	return []grpc.ServerOption{grpc.Creds(creds)}, nil

}
