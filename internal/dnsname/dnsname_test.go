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

package dnsname_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/dnsname"
)

func resetDomain(t *testing.T) {
	t.Helper()
	dnsname.SetClusterDomain("")
}

func TestServiceFQDN_Default(t *testing.T) {
	resetDomain(t)
	got := dnsname.ServiceFQDN("svc", "ns")
	want := "svc.ns.svc.cluster.local"
	if got != want {
		t.Errorf("ServiceFQDN = %q, want %q", got, want)
	}
}

func TestPodFQDN(t *testing.T) {
	resetDomain(t)
	got := dnsname.PodFQDN("pg-0", "head", "ns")
	want := "pg-0.head.ns.svc.cluster.local"
	if got != want {
		t.Errorf("PodFQDN = %q, want %q", got, want)
	}
}

func TestPrimaryFQDN(t *testing.T) {
	resetDomain(t)
	if !strings.Contains(dnsname.PrimaryFQDN("pg", "ns"), "pg-rw.ns.svc") {
		t.Errorf("PrimaryFQDN = %q", dnsname.PrimaryFQDN("pg", "ns"))
	}
}

func TestReplicaFQDN(t *testing.T) {
	resetDomain(t)
	if !strings.HasPrefix(dnsname.ReplicaFQDN("pg", "ns"), "pg-ro.ns") {
		t.Errorf("ReplicaFQDN = %q", dnsname.ReplicaFQDN("pg", "ns"))
	}
}

func TestAnyFQDN(t *testing.T) {
	resetDomain(t)
	if !strings.HasPrefix(dnsname.AnyFQDN("pg", "ns"), "pg-any.ns") {
		t.Errorf("AnyFQDN = %q", dnsname.AnyFQDN("pg", "ns"))
	}
}

func TestHeadlessFQDN(t *testing.T) {
	resetDomain(t)
	if !strings.HasPrefix(dnsname.HeadlessFQDN("pg", "ns"), "pg-headless.ns") {
		t.Errorf("HeadlessFQDN = %q", dnsname.HeadlessFQDN("pg", "ns"))
	}
}

func TestSearchPath(t *testing.T) {
	resetDomain(t)
	got := dnsname.SearchPath("foo")
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0] != "foo.svc.cluster.local" {
		t.Errorf("first entry = %q", got[0])
	}
}

func TestSetClusterDomain(t *testing.T) {
	defer dnsname.SetClusterDomain("")
	dnsname.SetClusterDomain("k8s.example.com")
	if got := dnsname.ServiceFQDN("a", "b"); got != "a.b.svc.k8s.example.com" {
		t.Errorf("custom domain ignored: %q", got)
	}
}

func TestSetClusterDomain_TrimsTrailingDot(t *testing.T) {
	defer dnsname.SetClusterDomain("")
	dnsname.SetClusterDomain("foo.example.")
	if got := dnsname.ClusterDomain(); got != "foo.example" {
		t.Errorf("trailing dot not trimmed: %q", got)
	}
}

func TestSetClusterDomain_EmptyResetsToDefault(t *testing.T) {
	dnsname.SetClusterDomain("custom")
	dnsname.SetClusterDomain("")
	if got := dnsname.ClusterDomain(); got != "cluster.local" {
		t.Errorf("expected default domain after empty Set, got %q", got)
	}
}
