/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certificate_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kitio-Tek/athos-kubernetes/internal/certificate"
)

func TestGenerateCA(t *testing.T) {
	kp, err := certificate.GenerateCA("my-cluster", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("GenerateCA() error: %v", err)
	}
	if len(kp.CertPEM) == 0 {
		t.Error("expected non-empty CA certificate PEM")
	}
	if len(kp.KeyPEM) == 0 {
		t.Error("expected non-empty CA key PEM")
	}
	if !strings.Contains(string(kp.CertPEM), "CERTIFICATE") {
		t.Error("CA cert PEM should contain CERTIFICATE block")
	}
	if !strings.Contains(string(kp.KeyPEM), "EC PRIVATE KEY") {
		t.Error("CA key PEM should contain EC PRIVATE KEY block")
	}
}

func TestGenerateServerCert(t *testing.T) {
	ca, err := certificate.GenerateCA("my-cluster", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("GenerateCA() error: %v", err)
	}

	sans := certificate.ServerSANs("my-cluster", "default", "my-cluster-pods")
	server, err := certificate.GenerateServerCert("my-cluster", sans, ca, certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("GenerateServerCert() error: %v", err)
	}
	if len(server.CertPEM) == 0 {
		t.Error("expected non-empty server certificate PEM")
	}
}

func TestNeedsRenewal_NotExpiringSoon(t *testing.T) {
	kp, err := certificate.GenerateCA("my-cluster", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("GenerateCA() error: %v", err)
	}

	needs, err := certificate.NeedsRenewal(kp.CertPEM, certificate.DefaultRenewBefore)
	if err != nil {
		t.Fatalf("NeedsRenewal() error: %v", err)
	}
	if needs {
		t.Error("expected fresh certificate to not need renewal")
	}
}

func TestNeedsRenewal_ExpiringSoon(t *testing.T) {
	// Generate a cert that expires in 10 days
	kp, err := certificate.GenerateCA("my-cluster", 10*24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateCA() error: %v", err)
	}

	// Renew 30 days before: should need renewal
	needs, err := certificate.NeedsRenewal(kp.CertPEM, certificate.DefaultRenewBefore)
	if err != nil {
		t.Fatalf("NeedsRenewal() error: %v", err)
	}
	if !needs {
		t.Error("expected expiring certificate to need renewal")
	}
}

func TestNeedsRenewal_InvalidPEM(t *testing.T) {
	_, err := certificate.NeedsRenewal([]byte("not a pem block"), certificate.DefaultRenewBefore)
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestExpiresAt(t *testing.T) {
	kp, err := certificate.GenerateCA("my-cluster", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("GenerateCA() error: %v", err)
	}

	expiry, err := certificate.ExpiresAt(kp.CertPEM)
	if err != nil {
		t.Fatalf("ExpiresAt() error: %v", err)
	}
	if expiry.IsZero() {
		t.Error("expected non-zero expiry time")
	}
	if !expiry.After(time.Now()) {
		t.Error("expected certificate to not yet be expired")
	}
}

func TestServerSANs(t *testing.T) {
	sans := certificate.ServerSANs("my-cluster", "default", "my-cluster-pods")
	if len(sans) == 0 {
		t.Error("expected non-empty SANs")
	}
	hasLocalhost := false
	hasFQDN := false
	for _, s := range sans {
		if s == "localhost" {
			hasLocalhost = true
		}
		if strings.Contains(s, "svc.cluster.local") {
			hasFQDN = true
		}
	}
	if !hasLocalhost {
		t.Error("SANs should include localhost")
	}
	if !hasFQDN {
		t.Error("SANs should include cluster.local FQDN")
	}
}

func TestGenerateCA_Uniqueness(t *testing.T) {
	kp1, err := certificate.GenerateCA("cluster-a", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("first GenerateCA() error: %v", err)
	}
	kp2, err := certificate.GenerateCA("cluster-b", certificate.DefaultCertDuration)
	if err != nil {
		t.Fatalf("second GenerateCA() error: %v", err)
	}
	if string(kp1.CertPEM) == string(kp2.CertPEM) {
		t.Error("two distinct CAs should have different certificates")
	}
}
