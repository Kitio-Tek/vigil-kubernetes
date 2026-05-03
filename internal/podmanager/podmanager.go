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

// Package podmanager classifies pods belonging to a PostgresCluster and
// produces the data the controller needs to make rolling-update,
// delete-and-recreate, and switchover decisions.
package podmanager

import (
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Phase summarises a pod's high-level status.
type Phase string

const (
	// PhasePending is a pod that has not been scheduled yet.
	PhasePending Phase = "Pending"
	// PhaseStarting is a pod that has been scheduled but is not yet ready.
	PhaseStarting Phase = "Starting"
	// PhaseHealthy is a pod that is ready and serving traffic.
	PhaseHealthy Phase = "Healthy"
	// PhaseUnhealthy is a pod whose container is running but not ready.
	PhaseUnhealthy Phase = "Unhealthy"
	// PhaseTerminating is a pod with a non-zero DeletionTimestamp.
	PhaseTerminating Phase = "Terminating"
	// PhaseFailed is a pod whose Phase is Failed.
	PhaseFailed Phase = "Failed"
)

// Classify reduces a corev1.Pod to a single Phase.
func Classify(p *corev1.Pod) Phase {
	if p == nil {
		return PhasePending
	}
	if p.DeletionTimestamp != nil {
		return PhaseTerminating
	}
	switch p.Status.Phase {
	case corev1.PodPending:
		return PhasePending
	case corev1.PodFailed:
		return PhaseFailed
	}
	if !isReady(p) {
		if hasRunningContainer(p) {
			return PhaseUnhealthy
		}
		return PhaseStarting
	}
	return PhaseHealthy
}

func isReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func hasRunningContainer(p *corev1.Pod) bool {
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Running != nil {
			return true
		}
	}
	return false
}

// Group sorts the pods of a cluster into health groups for use in rolling
// updates. The fields are exported so tests can compare slices.
type Group struct {
	Healthy      []*corev1.Pod
	Unhealthy    []*corev1.Pod
	Starting     []*corev1.Pod
	Terminating  []*corev1.Pod
	Pending      []*corev1.Pod
	Failed       []*corev1.Pod
}

// Total returns the number of pods across all groups.
func (g Group) Total() int {
	return len(g.Healthy) + len(g.Unhealthy) + len(g.Starting) +
		len(g.Terminating) + len(g.Pending) + len(g.Failed)
}

// IsAllHealthy reports whether every pod in the group is in the Healthy
// bucket. An empty group is treated as not-all-healthy.
func (g Group) IsAllHealthy() bool {
	if g.Total() == 0 {
		return false
	}
	return len(g.Healthy) == g.Total()
}

// HealthyCount returns the number of pods in the Healthy bucket.
func (g Group) HealthyCount() int { return len(g.Healthy) }

// SplitByPhase classifies a slice of pods into groups by Phase.
func SplitByPhase(pods []*corev1.Pod) Group {
	g := Group{}
	for _, p := range pods {
		switch Classify(p) {
		case PhaseHealthy:
			g.Healthy = append(g.Healthy, p)
		case PhaseUnhealthy:
			g.Unhealthy = append(g.Unhealthy, p)
		case PhaseStarting:
			g.Starting = append(g.Starting, p)
		case PhaseTerminating:
			g.Terminating = append(g.Terminating, p)
		case PhaseFailed:
			g.Failed = append(g.Failed, p)
		case PhasePending:
			g.Pending = append(g.Pending, p)
		}
	}
	return g
}

// SortByCreation sorts pods in-place by CreationTimestamp ascending. Pods
// created in the same second are tie-broken by name to keep the order stable.
func SortByCreation(pods []*corev1.Pod) {
	sort.Slice(pods, func(i, j int) bool {
		ti := pods[i].CreationTimestamp.Time
		tj := pods[j].CreationTimestamp.Time
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return pods[i].Name < pods[j].Name
	})
}

// OldestHealthy returns the oldest healthy pod or nil if there are none.
func OldestHealthy(pods []*corev1.Pod) *corev1.Pod {
	healthy := []*corev1.Pod{}
	for _, p := range pods {
		if Classify(p) == PhaseHealthy {
			healthy = append(healthy, p)
		}
	}
	if len(healthy) == 0 {
		return nil
	}
	SortByCreation(healthy)
	return healthy[0]
}

// AgeOf returns the age of the pod relative to now.
func AgeOf(p *corev1.Pod, now time.Time) time.Duration {
	if p == nil {
		return 0
	}
	if p.CreationTimestamp.Time.IsZero() {
		return 0
	}
	return now.Sub(p.CreationTimestamp.Time)
}
