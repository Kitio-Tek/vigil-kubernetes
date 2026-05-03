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

package podmanager_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/podmanager"
)

func pod(name string, status corev1.PodPhase, ready bool) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.Now()},
		Status:     corev1.PodStatus{Phase: status},
	}
	if ready {
		p.Status.Conditions = []corev1.PodCondition{
			{Type: corev1.PodReady, Status: corev1.ConditionTrue},
		}
	}
	return p
}

func TestClassify_Nil(t *testing.T) {
	if got := podmanager.Classify(nil); got != podmanager.PhasePending {
		t.Errorf("Classify(nil) = %s", got)
	}
}

func TestClassify_Healthy(t *testing.T) {
	if got := podmanager.Classify(pod("a", corev1.PodRunning, true)); got != podmanager.PhaseHealthy {
		t.Errorf("got %s", got)
	}
}

func TestClassify_Unhealthy(t *testing.T) {
	p := pod("a", corev1.PodRunning, false)
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
	}
	if got := podmanager.Classify(p); got != podmanager.PhaseUnhealthy {
		t.Errorf("got %s", got)
	}
}

func TestClassify_Starting(t *testing.T) {
	p := pod("a", corev1.PodRunning, false)
	if got := podmanager.Classify(p); got != podmanager.PhaseStarting {
		t.Errorf("got %s", got)
	}
}

func TestClassify_Terminating(t *testing.T) {
	p := pod("a", corev1.PodRunning, true)
	now := metav1.Now()
	p.DeletionTimestamp = &now
	if got := podmanager.Classify(p); got != podmanager.PhaseTerminating {
		t.Errorf("got %s", got)
	}
}

func TestClassify_Failed(t *testing.T) {
	if got := podmanager.Classify(pod("a", corev1.PodFailed, false)); got != podmanager.PhaseFailed {
		t.Errorf("got %s", got)
	}
}

func TestClassify_Pending(t *testing.T) {
	if got := podmanager.Classify(pod("a", corev1.PodPending, false)); got != podmanager.PhasePending {
		t.Errorf("got %s", got)
	}
}

func TestSplitByPhase(t *testing.T) {
	pods := []*corev1.Pod{
		pod("h1", corev1.PodRunning, true),
		pod("h2", corev1.PodRunning, true),
		pod("p", corev1.PodPending, false),
		pod("f", corev1.PodFailed, false),
	}
	g := podmanager.SplitByPhase(pods)
	if g.HealthyCount() != 2 {
		t.Errorf("expected 2 healthy, got %d", g.HealthyCount())
	}
	if len(g.Pending) != 1 {
		t.Errorf("expected 1 pending")
	}
	if len(g.Failed) != 1 {
		t.Errorf("expected 1 failed")
	}
	if g.Total() != 4 {
		t.Errorf("Total = %d", g.Total())
	}
}

func TestGroup_IsAllHealthy(t *testing.T) {
	if (podmanager.Group{}).IsAllHealthy() {
		t.Error("empty group should not be all healthy")
	}
	g := podmanager.SplitByPhase([]*corev1.Pod{pod("a", corev1.PodRunning, true)})
	if !g.IsAllHealthy() {
		t.Error("single healthy pod should be all healthy")
	}
}

func TestSortByCreation(t *testing.T) {
	a := pod("a", corev1.PodRunning, true)
	a.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	b := pod("b", corev1.PodRunning, true)
	b.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	pods := []*corev1.Pod{a, b}
	podmanager.SortByCreation(pods)
	if pods[0].Name != "b" {
		t.Errorf("expected b first, got %v", pods[0].Name)
	}
}

func TestOldestHealthy(t *testing.T) {
	a := pod("a", corev1.PodRunning, true)
	a.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	b := pod("b", corev1.PodRunning, true)
	b.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	c := pod("c", corev1.PodRunning, false)
	c.CreationTimestamp = metav1.NewTime(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))

	got := podmanager.OldestHealthy([]*corev1.Pod{a, b, c})
	if got == nil || got.Name != "b" {
		t.Errorf("expected b (oldest healthy), got %+v", got)
	}
}

func TestOldestHealthy_None(t *testing.T) {
	if got := podmanager.OldestHealthy(nil); got != nil {
		t.Errorf("expected nil, got %v", got.Name)
	}
}

func TestAgeOf(t *testing.T) {
	now := time.Now()
	p := pod("a", corev1.PodRunning, true)
	p.CreationTimestamp = metav1.NewTime(now.Add(-time.Hour))
	if d := podmanager.AgeOf(p, now); d < 59*time.Minute || d > 61*time.Minute {
		t.Errorf("age = %v, want ~1h", d)
	}
}

func TestAgeOf_Nil(t *testing.T) {
	if podmanager.AgeOf(nil, time.Now()) != 0 {
		t.Error("AgeOf(nil) should be 0")
	}
}
