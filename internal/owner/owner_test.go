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

package owner_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Kitio-Tek/athos-kubernetes/internal/owner"
)

func ptrBool(b bool) *bool { return &b }

func sample() []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion: "pg.athos.io/v1alpha1",
			Kind:       "PostgresCluster",
			Name:       "my-cluster",
			Controller: ptrBool(true),
		},
		{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "side-car",
		},
	}
}

func TestFind(t *testing.T) {
	refs := sample()
	if got := owner.Find(refs, "v1", "ConfigMap"); got == nil {
		t.Error("expected to find ConfigMap")
	}
	if got := owner.Find(refs, "v1", "Pod"); got != nil {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestFindByKind(t *testing.T) {
	if got := owner.FindByKind(sample(), "PostgresCluster"); got == nil {
		t.Error("expected to find PostgresCluster")
	}
	if got := owner.FindByKind(sample(), "Pod"); got != nil {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestIsControlledBy_Match(t *testing.T) {
	if !owner.IsControlledBy(sample(), "pg.athos.io/v1alpha1", "PostgresCluster", "my-cluster") {
		t.Error("expected IsControlledBy to be true")
	}
}

func TestIsControlledBy_NoController(t *testing.T) {
	refs := []metav1.OwnerReference{{
		APIVersion: "pg.athos.io/v1alpha1",
		Kind:       "PostgresCluster",
		Name:       "my-cluster",
	}}
	if owner.IsControlledBy(refs, "pg.athos.io/v1alpha1", "PostgresCluster", "my-cluster") {
		t.Error("expected false when Controller is nil")
	}
}

func TestIsControlledBy_DifferentName(t *testing.T) {
	if owner.IsControlledBy(sample(), "pg.athos.io/v1alpha1", "PostgresCluster", "other") {
		t.Error("expected false for different name")
	}
}

func TestIsManagedByAthos(t *testing.T) {
	if !owner.IsManagedByAthos(sample()) {
		t.Error("expected IsManagedByAthos true")
	}
	foreign := []metav1.OwnerReference{{Kind: "ReplicaSet"}}
	if owner.IsManagedByAthos(foreign) {
		t.Error("expected false for foreign owner")
	}
}

func TestNames(t *testing.T) {
	got := owner.Names(sample())
	if len(got) != 2 || got[0] != "my-cluster" || got[1] != "side-car" {
		t.Errorf("Names = %+v", got)
	}
}

func TestController(t *testing.T) {
	c := owner.Controller(sample())
	if c == nil || c.Name != "my-cluster" {
		t.Errorf("Controller = %+v", c)
	}
}

func TestController_None(t *testing.T) {
	refs := []metav1.OwnerReference{{Kind: "ConfigMap", Name: "x"}}
	if got := owner.Controller(refs); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
