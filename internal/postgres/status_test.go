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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
	"github.com/Kitio-Tek/vigil/internal/postgres"
)

func TestSetClusterConditionNew(t *testing.T) {
	cluster := newTestCluster("test")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionTrue, "AllReady", "all instances ready")

	if len(cluster.Status.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(cluster.Status.Conditions))
	}
	c := cluster.Status.Conditions[0]
	if c.Type != pgv1alpha1.ConditionReady {
		t.Errorf("condition type = %q, want %q", c.Type, pgv1alpha1.ConditionReady)
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("condition status = %q, want True", c.Status)
	}
	if c.Reason != "AllReady" {
		t.Errorf("condition reason = %q, want AllReady", c.Reason)
	}
}

func TestSetClusterConditionUpdate(t *testing.T) {
	cluster := newTestCluster("test")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionFalse, "NotReady", "no instances ready")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionTrue, "AllReady", "all instances ready")

	if len(cluster.Status.Conditions) != 1 {
		t.Fatalf("expected 1 condition after update, got %d", len(cluster.Status.Conditions))
	}
	c := cluster.Status.Conditions[0]
	if c.Status != metav1.ConditionTrue {
		t.Errorf("updated condition status = %q, want True", c.Status)
	}
}

func TestSetClusterConditionMultiple(t *testing.T) {
	cluster := newTestCluster("test")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionTrue, "AllReady", "all ready")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionDegraded,
		metav1.ConditionFalse, "OK", "not degraded")

	if len(cluster.Status.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(cluster.Status.Conditions))
	}
}

func TestIsClusterReadyTrue(t *testing.T) {
	cluster := newTestCluster("test")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionTrue, "AllReady", "all ready")

	if !postgres.IsClusterReady(cluster) {
		t.Error("IsClusterReady should return true when Ready condition is True")
	}
}

func TestIsClusterReadyFalse(t *testing.T) {
	cluster := newTestCluster("test")
	postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
		metav1.ConditionFalse, "NotReady", "not ready")

	if postgres.IsClusterReady(cluster) {
		t.Error("IsClusterReady should return false when Ready condition is False")
	}
}

func TestIsClusterReadyNoConditions(t *testing.T) {
	cluster := newTestCluster("test")
	if postgres.IsClusterReady(cluster) {
		t.Error("IsClusterReady should return false when there are no conditions")
	}
}

func TestIsClusterRunning(t *testing.T) {
	cluster := newTestCluster("test")
	cluster.Status.Phase = pgv1alpha1.PhaseRunning

	if !postgres.IsClusterRunning(cluster) {
		t.Error("IsClusterRunning should return true for Running phase")
	}

	cluster.Status.Phase = pgv1alpha1.PhaseDegraded
	if postgres.IsClusterRunning(cluster) {
		t.Error("IsClusterRunning should return false for Degraded phase")
	}
}
