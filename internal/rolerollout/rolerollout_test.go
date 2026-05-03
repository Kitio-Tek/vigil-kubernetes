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

package rolerollout_test

import (
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/rolerollout"
)

func TestPlan_PrimaryLast(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", Ordinal: 0, IsPrimary: true},
		{Name: "pg-1", Ordinal: 1},
		{Name: "pg-2", Ordinal: 2},
	}
	plan := rolerollout.Plan(pods)
	if len(plan) != 3 {
		t.Fatalf("plan len = %d", len(plan))
	}
	if !plan[len(plan)-1].IsPrimary {
		t.Errorf("primary should be last, got %+v", plan)
	}
}

func TestPlan_ReplicasInReverseOrdinal(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-1", Ordinal: 1},
		{Name: "pg-0", Ordinal: 0, IsPrimary: true},
		{Name: "pg-2", Ordinal: 2},
	}
	plan := rolerollout.Plan(pods)
	if plan[0].Ordinal != 2 || plan[1].Ordinal != 1 {
		t.Errorf("expected replicas in reverse order, got %+v", plan)
	}
}

func TestPlan_SkipsAlreadyOnDesiredImage(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", Ordinal: 0, IsPrimary: true},
		{Name: "pg-1", Ordinal: 1, OnDesiredImage: true},
		{Name: "pg-2", Ordinal: 2},
	}
	plan := rolerollout.Plan(pods)
	if len(plan) != 2 {
		t.Errorf("expected 2 pending pods, got %d", len(plan))
	}
	for _, p := range plan {
		if p.Name == "pg-1" {
			t.Error("already-updated pod should not appear in plan")
		}
	}
}

func TestCanProceed_AllUpdatedHealthy(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", IsPrimary: true},
		{Name: "pg-1", OnDesiredImage: true, HealthyOK: true},
	}
	if !rolerollout.CanProceed(pods) {
		t.Error("expected CanProceed=true when updated pods are healthy")
	}
}

func TestCanProceed_UpdatedPodUnhealthy(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", IsPrimary: true},
		{Name: "pg-1", OnDesiredImage: true, HealthyOK: false},
	}
	if rolerollout.CanProceed(pods) {
		t.Error("expected CanProceed=false when an updated pod is unhealthy")
	}
}

func TestIsComplete_Empty(t *testing.T) {
	if rolerollout.IsComplete(nil) {
		t.Error("empty pod list should not be considered complete")
	}
}

func TestIsComplete_AllUpdated(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", OnDesiredImage: true, HealthyOK: true},
		{Name: "pg-1", OnDesiredImage: true, HealthyOK: true},
	}
	if !rolerollout.IsComplete(pods) {
		t.Error("expected complete")
	}
}

func TestPendingCount(t *testing.T) {
	pods := []rolerollout.Pod{
		{OnDesiredImage: false},
		{OnDesiredImage: true},
		{OnDesiredImage: false},
	}
	if got := rolerollout.PendingCount(pods); got != 2 {
		t.Errorf("PendingCount = %d, want 2", got)
	}
}

func TestNext_PicksFirstReplica(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", Ordinal: 0, IsPrimary: true},
		{Name: "pg-1", Ordinal: 1},
		{Name: "pg-2", Ordinal: 2},
	}
	next := rolerollout.Next(pods)
	if next == nil || next.Ordinal != 2 {
		t.Errorf("expected pg-2 as next, got %+v", next)
	}
}

func TestNext_BlockedByUnhealthyPriorUpdate(t *testing.T) {
	pods := []rolerollout.Pod{
		{Name: "pg-0", IsPrimary: true},
		{Name: "pg-2", Ordinal: 2, OnDesiredImage: true, HealthyOK: false},
		{Name: "pg-1", Ordinal: 1},
	}
	if rolerollout.Next(pods) != nil {
		t.Error("expected nil while previous step is still unhealthy")
	}
}

func TestNext_AllDone(t *testing.T) {
	pods := []rolerollout.Pod{
		{OnDesiredImage: true, HealthyOK: true},
	}
	if got := rolerollout.Next(pods); got != nil {
		t.Errorf("expected nil when nothing pending, got %+v", got)
	}
}
