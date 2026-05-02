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

package ha_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/ha"
)

func TestTopologyInfo_IsHA(t *testing.T) {
	tests := []struct {
		instances int32
		wantHA    bool
	}{
		{1, false},
		{2, false},
		{3, true},
		{5, true},
	}
	for _, tt := range tests {
		topo := ha.TopologyInfo{Instances: tt.instances}
		if topo.IsHA() != tt.wantHA {
			t.Errorf("IsHA() with %d instances: expected %v, got %v", tt.instances, tt.wantHA, topo.IsHA())
		}
	}
}

func TestTopologyInfo_CanFailover(t *testing.T) {
	t.Run("no primary", func(t *testing.T) {
		topo := ha.TopologyInfo{ReplicaPods: []string{"pod-1"}}
		if topo.CanFailover() {
			t.Error("expected CanFailover to return false when no primary")
		}
	})
	t.Run("no replicas", func(t *testing.T) {
		topo := ha.TopologyInfo{PrimaryPod: "pod-0"}
		if topo.CanFailover() {
			t.Error("expected CanFailover to return false when no replicas")
		}
	})
	t.Run("primary and replicas", func(t *testing.T) {
		topo := ha.TopologyInfo{PrimaryPod: "pod-0", ReplicaPods: []string{"pod-1", "pod-2"}}
		if !topo.CanFailover() {
			t.Error("expected CanFailover to return true")
		}
	})
}

func TestTopologyInfo_FailoverCandidate(t *testing.T) {
	topo := ha.TopologyInfo{
		PrimaryPod:  "pod-0",
		ReplicaPods: []string{"pod-1", "pod-2"},
	}
	candidate, err := topo.FailoverCandidate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if candidate != "pod-1" {
		t.Errorf("expected pod-1, got %q", candidate)
	}
}

func TestTopologyInfo_FailoverCandidate_NoReplica(t *testing.T) {
	topo := ha.TopologyInfo{PrimaryPod: "pod-0"}
	_, err := topo.FailoverCandidate()
	if err == nil {
		t.Error("expected error when no replicas available")
	}
}

func TestFailoverRequested(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ha.AnnotationFailoverTarget: "pod-1",
			},
		},
	}
	if !ha.FailoverRequested(cluster) {
		t.Error("expected FailoverRequested to return true")
	}
}

func TestFailoverRequested_NoAnnotation(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{}
	if ha.FailoverRequested(cluster) {
		t.Error("expected FailoverRequested to return false with no annotations")
	}
}

func TestFailoverTarget(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ha.AnnotationFailoverTarget: "pod-2",
			},
		},
	}
	target := ha.FailoverTarget(cluster)
	if target != "pod-2" {
		t.Errorf("expected pod-2, got %q", target)
	}
}

func TestClearFailoverAnnotations(t *testing.T) {
	annotations := map[string]string{
		ha.AnnotationFailoverTarget:   "pod-1",
		ha.AnnotationSwitchoverTarget: "pod-2",
		"some.other.annotation":       "value",
	}
	result := ha.ClearFailoverAnnotations(annotations)
	if _, ok := result[ha.AnnotationFailoverTarget]; ok {
		t.Error("failover annotation should be removed")
	}
	if _, ok := result[ha.AnnotationSwitchoverTarget]; ok {
		t.Error("switchover annotation should be removed")
	}
	if result["some.other.annotation"] != "value" {
		t.Error("unrelated annotation should be preserved")
	}
}

func TestClearFailoverAnnotations_Nil(t *testing.T) {
	result := ha.ClearFailoverAnnotations(nil)
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}

func TestPodOrdinal(t *testing.T) {
	tests := []struct {
		podName string
		want    int
	}{
		{"my-cluster-0", 0},
		{"my-cluster-1", 1},
		{"my-cluster-10", 10},
		{"my-cluster-postgres-0", 0},
		{"nohyphen", -1},
		{"", -1},
		{"pod-", -1},
	}
	for _, tt := range tests {
		got := ha.PodOrdinal(tt.podName)
		if got != tt.want {
			t.Errorf("PodOrdinal(%q) = %d, want %d", tt.podName, got, tt.want)
		}
	}
}
