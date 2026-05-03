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

package pdb_test

import (
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/pdb"
)

func TestBuild_DefaultsToMaxUnavailable(t *testing.T) {
	out := pdb.Build(pdb.Spec{Name: "x", Namespace: "ns", Selector: map[string]string{"a": "b"}})
	if out.Spec.MaxUnavailable == nil || out.Spec.MaxUnavailable.IntValue() != 1 {
		t.Errorf("expected default MaxUnavailable=1, got %+v", out.Spec.MaxUnavailable)
	}
	if out.Spec.MinAvailable != nil {
		t.Errorf("MinAvailable should be nil")
	}
}

func TestBuild_MinAvailable(t *testing.T) {
	out := pdb.Build(pdb.Spec{
		Name: "x", Namespace: "ns",
		Selector:     map[string]string{"a": "b"},
		MinAvailable: 2,
	})
	if out.Spec.MinAvailable == nil || out.Spec.MinAvailable.IntValue() != 2 {
		t.Errorf("MinAvailable = %+v", out.Spec.MinAvailable)
	}
	if out.Spec.MaxUnavailable != nil {
		t.Error("MaxUnavailable should be nil when MinAvailable is set")
	}
}

func TestBuild_MaxUnavailable(t *testing.T) {
	out := pdb.Build(pdb.Spec{
		Name: "x", Namespace: "ns",
		Selector:       map[string]string{"a": "b"},
		MaxUnavailable: 2,
	})
	if out.Spec.MaxUnavailable == nil || out.Spec.MaxUnavailable.IntValue() != 2 {
		t.Errorf("MaxUnavailable = %+v", out.Spec.MaxUnavailable)
	}
}

func TestBuild_SelectorPropagated(t *testing.T) {
	out := pdb.Build(pdb.Spec{
		Name: "x", Namespace: "ns",
		Selector: map[string]string{"role": "primary"},
	})
	if out.Spec.Selector == nil || out.Spec.Selector.MatchLabels["role"] != "primary" {
		t.Errorf("selector lost: %+v", out.Spec.Selector)
	}
}

func TestRecommendedFor_HASizing(t *testing.T) {
	s := pdb.RecommendedFor("x", "ns", 3, nil, nil)
	if s.MinAvailable != 2 {
		t.Errorf("MinAvailable for 3 instances = %d", s.MinAvailable)
	}
	if s.MaxUnavailable != 0 {
		t.Errorf("MaxUnavailable should be 0 when MinAvailable is used: %d", s.MaxUnavailable)
	}
}

func TestRecommendedFor_SmallCluster(t *testing.T) {
	s := pdb.RecommendedFor("x", "ns", 2, nil, nil)
	if s.MaxUnavailable != 1 {
		t.Errorf("MaxUnavailable for 2 instances = %d", s.MaxUnavailable)
	}
	if s.MinAvailable != 0 {
		t.Errorf("MinAvailable should be 0 for small clusters")
	}
}

func TestIsAdvisory(t *testing.T) {
	if !pdb.IsAdvisory(pdb.Spec{}) {
		t.Error("zero-value spec should be advisory")
	}
	if pdb.IsAdvisory(pdb.Spec{MinAvailable: 1}) {
		t.Error("with MinAvailable should not be advisory")
	}
}
