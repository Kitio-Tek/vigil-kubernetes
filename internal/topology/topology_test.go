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

package topology_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/topology"
)

var selector = map[string]string{"app": "vigil", "cluster": "pg"}

func TestPodAntiAffinity_None(t *testing.T) {
	if topology.PodAntiAffinity(selector, topology.SpreadNone) != nil {
		t.Error("SpreadNone should return nil affinity")
	}
}

func TestPodAntiAffinity_Host(t *testing.T) {
	a := topology.PodAntiAffinity(selector, topology.SpreadHost)
	if a == nil {
		t.Fatal("expected anti-affinity for SpreadHost")
	}
	if len(a.RequiredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Errorf("expected 1 required term")
	}
	if a.RequiredDuringSchedulingIgnoredDuringExecution[0].TopologyKey != topology.KeyHostname {
		t.Errorf("expected hostname topology key")
	}
}

func TestPodAntiAffinity_Zone(t *testing.T) {
	a := topology.PodAntiAffinity(selector, topology.SpreadZone)
	if a == nil || a.RequiredDuringSchedulingIgnoredDuringExecution[0].TopologyKey != topology.KeyZone {
		t.Error("expected required zone topology key")
	}
}

func TestPodAntiAffinity_PreferHost(t *testing.T) {
	a := topology.PodAntiAffinity(selector, topology.SpreadPreferHost)
	if a == nil {
		t.Fatal("expected affinity for prefer host")
	}
	if len(a.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Errorf("expected 1 preferred term")
	}
}

func TestPodAntiAffinity_PreferZone(t *testing.T) {
	a := topology.PodAntiAffinity(selector, topology.SpreadPreferZone)
	if len(a.PreferredDuringSchedulingIgnoredDuringExecution) != 1 ||
		a.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.TopologyKey != topology.KeyZone {
		t.Errorf("expected preferred zone term")
	}
}

func TestSpreads(t *testing.T) {
	s := topology.Spreads(selector)
	if len(s) != 2 {
		t.Fatalf("expected 2 spreads, got %d", len(s))
	}
	if s[0].WhenUnsatisfiable != corev1.DoNotSchedule {
		t.Errorf("first spread should be DoNotSchedule")
	}
	if s[1].WhenUnsatisfiable != corev1.ScheduleAnyway {
		t.Errorf("second spread should be ScheduleAnyway")
	}
}

func TestIsRequired(t *testing.T) {
	cases := map[topology.Spread]bool{
		topology.SpreadNone:        false,
		topology.SpreadHost:        true,
		topology.SpreadZone:        true,
		topology.SpreadPreferHost:  false,
		topology.SpreadPreferZone:  false,
	}
	for s, want := range cases {
		if got := topology.IsRequired(s); got != want {
			t.Errorf("IsRequired(%s) = %v, want %v", s, got, want)
		}
	}
}

func TestSpreadFromString(t *testing.T) {
	cases := map[string]topology.Spread{
		"":            topology.SpreadPreferHost,
		"none":        topology.SpreadNone,
		"None":        topology.SpreadNone,
		"Host":        topology.SpreadHost,
		"zone":        topology.SpreadZone,
		"PreferHost":  topology.SpreadPreferHost,
		"PreferZone":  topology.SpreadPreferZone,
		"unrecognized": topology.SpreadPreferHost,
	}
	for s, want := range cases {
		if got := topology.SpreadFromString(s); got != want {
			t.Errorf("SpreadFromString(%q) = %s, want %s", s, got, want)
		}
	}
}
