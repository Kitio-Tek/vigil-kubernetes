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

// Package ha implements high-availability logic for the Vigil operator. It
// provides helpers for determining cluster topology, validating failover
// eligibility, and producing the annotations used to trigger switchovers.
package ha

import (
	"fmt"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
)

const (
	// AnnotationFailoverTarget is the annotation set by an operator or user to
	// request a manual failover to a specific pod.
	AnnotationFailoverTarget = "pg.vigil.io/failover-target"

	// AnnotationSwitchoverTarget is the annotation set to request a planned
	// switchover to a specific pod (graceful, with data sync).
	AnnotationSwitchoverTarget = "pg.vigil.io/switchover-target"

	// AnnotationCurrentPrimary records the pod name of the current primary.
	AnnotationCurrentPrimary = "pg.vigil.io/current-primary"

	// AnnotationPromoteWithoutDataSync allows promoting a replica without
	// waiting for full data synchronisation. Use only in disaster scenarios.
	AnnotationPromoteWithoutDataSync = "pg.vigil.io/promote-without-data-sync"

	// LabelRole is the label used to identify the role of a pod.
	LabelRole = "pg.vigil.io/role"

	// RolePrimary identifies the primary pod.
	RolePrimary = "primary"

	// RoleReplica identifies a replica pod.
	RoleReplica = "replica"

	// MinInstancesForHA is the minimum number of instances to achieve HA.
	MinInstancesForHA = 3
)

// TopologyInfo describes the current primary/replica topology of a cluster.
type TopologyInfo struct {
	// PrimaryPod is the name of the current primary pod.
	PrimaryPod string
	// ReplicaPods holds the names of all known replica pods.
	ReplicaPods []string
	// Instances is the total configured instance count.
	Instances int32
}

// IsHA returns true if the topology has the minimum number of instances
// required to tolerate a single-node failure.
func (t TopologyInfo) IsHA() bool {
	return t.Instances >= MinInstancesForHA
}

// HasReplicas returns true if there is at least one replica.
func (t TopologyInfo) HasReplicas() bool {
	return len(t.ReplicaPods) > 0
}

// CanFailover returns true when the topology can accommodate an automatic or
// manual failover (there is a primary and at least one candidate replica).
func (t TopologyInfo) CanFailover() bool {
	return t.PrimaryPod != "" && len(t.ReplicaPods) > 0
}

// FailoverCandidate returns the preferred replica for promotion. The current
// implementation returns the first replica in the list; a future implementation
// may use replication lag metrics to select the most up-to-date replica.
func (t TopologyInfo) FailoverCandidate() (string, error) {
	if !t.CanFailover() {
		return "", fmt.Errorf("no failover candidate available: primary=%q replicas=%v",
			t.PrimaryPod, t.ReplicaPods)
	}
	return t.ReplicaPods[0], nil
}

// FailoverRequested returns true when the cluster has a failover or switchover
// annotation requesting a topology change.
func FailoverRequested(cluster *pgv1alpha1.PostgresCluster) bool {
	annotations := cluster.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, hasFailover := annotations[AnnotationFailoverTarget]
	_, hasSwitchover := annotations[AnnotationSwitchoverTarget]
	return hasFailover || hasSwitchover
}

// FailoverTarget returns the pod name requested as the new primary in a manual
// failover annotation. Returns an empty string if no annotation is set.
func FailoverTarget(cluster *pgv1alpha1.PostgresCluster) string {
	if cluster.GetAnnotations() == nil {
		return ""
	}
	return cluster.GetAnnotations()[AnnotationFailoverTarget]
}

// SwitchoverTarget returns the pod name requested for a planned switchover.
func SwitchoverTarget(cluster *pgv1alpha1.PostgresCluster) string {
	if cluster.GetAnnotations() == nil {
		return ""
	}
	return cluster.GetAnnotations()[AnnotationSwitchoverTarget]
}

// ClearFailoverAnnotations removes failover and switchover annotations from the
// cluster's annotation map and returns it. The caller is responsible for
// applying the updated annotation map.
func ClearFailoverAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}
	out := make(map[string]string, len(annotations))
	for k, v := range annotations {
		out[k] = v
	}
	delete(out, AnnotationFailoverTarget)
	delete(out, AnnotationSwitchoverTarget)
	return out
}

// PodOrdinal extracts the ordinal integer from a StatefulSet pod name of the
// form "<cluster>-<n>". Returns -1 if the name cannot be parsed.
func PodOrdinal(podName string) int {
	n := len(podName)
	if n == 0 {
		return -1
	}
	i := n - 1
	for i >= 0 && podName[i] >= '0' && podName[i] <= '9' {
		i--
	}
	if i == n-1 || podName[i] != '-' {
		return -1
	}
	ordinal := 0
	mul := 1
	for j := n - 1; j > i; j-- {
		ordinal += int(podName[j]-'0') * mul
		mul *= 10
	}
	return ordinal
}
