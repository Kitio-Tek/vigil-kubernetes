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

package portmap_test

import (
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/portmap"
)

func TestAll_ContainsExpectedNames(t *testing.T) {
	want := []string{
		portmap.PostgresPortName,
		portmap.PgBouncerPortName,
		portmap.MetricsPortName,
		portmap.ManagerMetricsPortName,
		portmap.ManagerHealthPortName,
	}
	all := portmap.All()
	if len(all) != len(want) {
		t.Fatalf("len = %d, want %d", len(all), len(want))
	}
	seen := map[string]bool{}
	for _, p := range all {
		seen[p.Name] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("missing %q in All", name)
		}
	}
}

func TestPostgresOnly(t *testing.T) {
	got := portmap.PostgresOnly()
	if len(got) != 1 || got[0].Name != portmap.PostgresPortName {
		t.Errorf("PostgresOnly = %+v", got)
	}
}

func TestFind_Existing(t *testing.T) {
	p, ok := portmap.Find(portmap.PostgresPortName)
	if !ok || p.Port != portmap.PostgresPort {
		t.Errorf("Find = %+v ok=%v", p, ok)
	}
}

func TestFind_Unknown(t *testing.T) {
	if _, ok := portmap.Find("nope"); ok {
		t.Error("expected ok=false for unknown name")
	}
}

func TestIsWellKnown(t *testing.T) {
	if !portmap.IsWellKnown(portmap.PostgresPort) {
		t.Error("postgres port should be well known")
	}
	if portmap.IsWellKnown(1234) {
		t.Error("1234 should not be well known")
	}
}

func TestSortedNames_StableOrder(t *testing.T) {
	a := portmap.SortedNames()
	b := portmap.SortedNames()
	if len(a) != len(b) {
		t.Fatalf("differing lengths")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("order differs at %d: %q vs %q", i, a[i], b[i])
		}
	}
}

func TestMustFind_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	portmap.MustFind("not-real")
}

func TestMustFind_Returns(t *testing.T) {
	p := portmap.MustFind(portmap.PgBouncerPortName)
	if p.Port != portmap.PgBouncerPort {
		t.Errorf("port = %d, want %d", p.Port, portmap.PgBouncerPort)
	}
}
