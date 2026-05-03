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

// Package rolerollout chooses which pod to update next during a rolling
// upgrade of a PostgresCluster. The rule is simple: replicas first (in
// reverse ordinal order), then the primary, and only after the primary
// finishes its restart-and-recover cycle.
package rolerollout

import (
	"sort"
)

// Pod is a minimal projection of a Kubernetes pod that the rollout planner
// needs. The controller code maps real corev1.Pods into this shape before
// calling Plan.
type Pod struct {
	Name           string
	Ordinal        int
	IsPrimary      bool
	HealthyOK      bool
	OnDesiredImage bool
}

// Plan returns the ordered slice of pods that still need to be updated.
// Pods already on the desired image are skipped. Replicas come first
// (reverse-ordinal so the most recently created replica is updated first);
// the primary is always updated last.
func Plan(pods []Pod) []Pod {
	pending := make([]Pod, 0, len(pods))
	var primary *Pod

	for i := range pods {
		p := pods[i]
		if p.OnDesiredImage {
			continue
		}
		if p.IsPrimary {
			tmp := p
			primary = &tmp
			continue
		}
		pending = append(pending, p)
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Ordinal > pending[j].Ordinal
	})

	if primary != nil {
		pending = append(pending, *primary)
	}
	return pending
}

// CanProceed reports whether the rollout may safely advance to the next
// pod. The rule is "no rollout step while a previous one is still being
// applied": every replica that has already been touched (i.e. is on the
// desired image) must be healthy before the next pod can be evicted.
func CanProceed(pods []Pod) bool {
	for _, p := range pods {
		if p.OnDesiredImage && !p.HealthyOK {
			return false
		}
	}
	return true
}

// IsComplete reports whether every pod is on the desired image and healthy.
func IsComplete(pods []Pod) bool {
	if len(pods) == 0 {
		return false
	}
	for _, p := range pods {
		if !p.OnDesiredImage || !p.HealthyOK {
			return false
		}
	}
	return true
}

// PendingCount returns the number of pods still requiring an update.
func PendingCount(pods []Pod) int {
	count := 0
	for _, p := range pods {
		if !p.OnDesiredImage {
			count++
		}
	}
	return count
}

// Next returns the next pod to update or nil if none.
func Next(pods []Pod) *Pod {
	plan := Plan(pods)
	if len(plan) == 0 {
		return nil
	}
	if !CanProceed(pods) {
		return nil
	}
	first := plan[0]
	return &first
}
