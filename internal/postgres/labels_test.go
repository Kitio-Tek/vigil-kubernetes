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

package postgres_test

import (
	"testing"

	"github.com/Kitio-Tek/vigil/internal/postgres"
)

func TestCommonLabels(t *testing.T) {
	c := newTestCluster("mycluster")
	labels := postgres.CommonLabels(c)

	required := []string{
		postgres.LabelCluster,
		postgres.LabelManagedBy,
		postgres.LabelName,
		postgres.LabelInstance,
	}
	for _, k := range required {
		if _, ok := labels[k]; !ok {
			t.Errorf("common labels missing key %q", k)
		}
	}
	if labels[postgres.LabelCluster] != "mycluster" {
		t.Errorf("LabelCluster = %q, want mycluster", labels[postgres.LabelCluster])
	}
	if labels[postgres.LabelManagedBy] != postgres.OperatorName {
		t.Errorf("LabelManagedBy = %q, want %q", labels[postgres.LabelManagedBy], postgres.OperatorName)
	}
}

func TestSelectorLabels(t *testing.T) {
	c := newTestCluster("mycluster")
	sel := postgres.SelectorLabels(c)

	if sel[postgres.LabelCluster] != "mycluster" {
		t.Errorf("selector labels missing cluster name")
	}
	// Selector labels should be a subset of common labels.
	common := postgres.CommonLabels(c)
	for k, v := range sel {
		if common[k] != v {
			t.Errorf("selector label %q=%q not present in common labels", k, v)
		}
	}
}

func TestPodLabels(t *testing.T) {
	c := newTestCluster("mycluster")

	primary := postgres.PodLabels(c, postgres.RolePrimary)
	if primary[postgres.LabelRole] != postgres.RolePrimary {
		t.Errorf("pod labels role = %q, want %q", primary[postgres.LabelRole], postgres.RolePrimary)
	}

	replica := postgres.PodLabels(c, postgres.RoleReplica)
	if replica[postgres.LabelRole] != postgres.RoleReplica {
		t.Errorf("pod labels role = %q, want %q", replica[postgres.LabelRole], postgres.RoleReplica)
	}
}

func TestMergeLabels(t *testing.T) {
	a := map[string]string{"x": "1", "y": "2"}
	b := map[string]string{"y": "overridden", "z": "3"}
	merged := postgres.MergeLabels(a, b)

	if merged["x"] != "1" {
		t.Errorf("base key x missing")
	}
	if merged["y"] != "overridden" {
		t.Errorf("overlay key y not applied: got %q", merged["y"])
	}
	if merged["z"] != "3" {
		t.Errorf("overlay key z missing")
	}
	// Original maps must not be mutated.
	if a["y"] != "2" {
		t.Error("MergeLabels mutated the base map")
	}
}
