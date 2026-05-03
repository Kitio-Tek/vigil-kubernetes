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

package conditions_test

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/conditions"
)

func TestSet_AppendsNew(t *testing.T) {
	out := conditions.Set(nil, conditions.True("Ready", "Up", "all good"))
	if len(out) != 1 || out[0].Type != "Ready" {
		t.Errorf("expected Ready condition appended, got %+v", out)
	}
}

func TestSet_PreservesTransitionTimeWhenStatusUnchanged(t *testing.T) {
	now := metav1.NewTime(time.Now().Add(-time.Hour))
	conds := []metav1.Condition{{
		Type: "Ready", Status: metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "Up", Message: "ok",
	}}
	out := conditions.Set(conds, conditions.True("Ready", "Up", "still ok"))
	if !out[0].LastTransitionTime.Equal(&now) {
		t.Errorf("LastTransitionTime should be preserved when status unchanged")
	}
	if out[0].Message != "still ok" {
		t.Errorf("Message should be updated, got %q", out[0].Message)
	}
}

func TestSet_BumpsTransitionTimeWhenStatusChanges(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	conds := []metav1.Condition{{
		Type: "Ready", Status: metav1.ConditionTrue, LastTransitionTime: old,
	}}
	out := conditions.Set(conds, conditions.False("Ready", "Down", "broken"))
	if out[0].LastTransitionTime.Equal(&old) {
		t.Error("LastTransitionTime should change when status flips")
	}
	if out[0].Status != metav1.ConditionFalse {
		t.Errorf("Status = %s, want False", out[0].Status)
	}
}

func TestRemove(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "A"},
		{Type: "B"},
		{Type: "C"},
	}
	out := conditions.Remove(conds, "B")
	if len(out) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(out))
	}
	for _, c := range out {
		if c.Type == "B" {
			t.Error("B should have been removed")
		}
	}
}

func TestFind_Found(t *testing.T) {
	conds := []metav1.Condition{{Type: "Ready"}}
	if c := conditions.Find(conds, "Ready"); c == nil {
		t.Error("expected to find Ready")
	}
}

func TestFind_NotFound(t *testing.T) {
	if c := conditions.Find(nil, "Ready"); c != nil {
		t.Error("expected nil for missing condition")
	}
}

func TestIsTrue_FalseUnknown(t *testing.T) {
	conds := []metav1.Condition{
		conditions.True("A", "r", "m"),
		conditions.False("B", "r", "m"),
		conditions.Unknown("C", "r", "m"),
	}
	if !conditions.IsTrue(conds, "A") {
		t.Error("A should be true")
	}
	if !conditions.IsFalse(conds, "B") {
		t.Error("B should be false")
	}
	if !conditions.IsUnknown(conds, "C") {
		t.Error("C should be unknown")
	}
	// Missing condition is Unknown.
	if !conditions.IsUnknown(conds, "Missing") {
		t.Error("missing condition should be Unknown")
	}
}

func TestWithObservedGeneration(t *testing.T) {
	c := conditions.WithObservedGeneration(conditions.True("Ready", "r", "m"), 42)
	if c.ObservedGeneration != 42 {
		t.Errorf("ObservedGeneration = %d", c.ObservedGeneration)
	}
}

func TestAnyFalse(t *testing.T) {
	conds := []metav1.Condition{
		conditions.True("A", "r", "m"),
		conditions.False("B", "r", "m"),
	}
	if !conditions.AnyFalse(conds, "A", "B") {
		t.Error("expected AnyFalse to find B")
	}
	if conditions.AnyFalse(conds, "A") {
		t.Error("AnyFalse on only A should return false")
	}
}

func TestAllTrue(t *testing.T) {
	conds := []metav1.Condition{
		conditions.True("A", "r", "m"),
		conditions.True("B", "r", "m"),
	}
	if !conditions.AllTrue(conds, "A", "B") {
		t.Error("AllTrue should be true")
	}
	if conditions.AllTrue(conds, "A", "B", "Missing") {
		t.Error("AllTrue should be false when one is missing")
	}
}

func TestFilterByPrefix(t *testing.T) {
	conds := []metav1.Condition{
		conditions.True("Pooler.Ready", "r", "m"),
		conditions.True("Pooler.Healthy", "r", "m"),
		conditions.True("Cluster.Ready", "r", "m"),
	}
	got := conditions.FilterByPrefix(conds, "Pooler.")
	if len(got) != 2 {
		t.Errorf("expected 2 Pooler conditions, got %d", len(got))
	}
}

func TestFilterByPrefix_Empty(t *testing.T) {
	conds := []metav1.Condition{conditions.True("A", "r", "m")}
	got := conditions.FilterByPrefix(conds, "")
	if len(got) != 1 {
		t.Errorf("empty prefix should match all, got %d", len(got))
	}
}
