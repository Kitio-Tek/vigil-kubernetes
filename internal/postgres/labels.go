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

package postgres

import (
	"fmt"

	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
)

const (
	// LabelCluster identifies the PostgresCluster a resource belongs to.
	LabelCluster = "pg.vigil.io/cluster"
	// LabelRole distinguishes primary from replica pods.
	LabelRole = "pg.vigil.io/role"
	// LabelManagedBy identifies the operator managing a resource.
	LabelManagedBy = "app.kubernetes.io/managed-by"
	// LabelComponent identifies the functional component of a resource.
	LabelComponent = "app.kubernetes.io/component"
	// LabelName is the application name label.
	LabelName = "app.kubernetes.io/name"
	// LabelInstance is the unique instance identifier label.
	LabelInstance = "app.kubernetes.io/instance"
	// LabelVersion labels the PostgreSQL major version.
	LabelVersion = "pg.vigil.io/postgres-version"

	// RolePrimary is the value used in LabelRole for the primary pod.
	RolePrimary = "primary"
	// RoleReplica is the value used in LabelRole for replica pods.
	RoleReplica = "replica"

	// OperatorName is the canonical name of this operator.
	OperatorName = "vigil"
)

// CommonLabels returns the standard set of labels applied to every resource
// owned by a PostgresCluster. These labels should not be used as pod selectors
// because they include mutable metadata.
func CommonLabels(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	return map[string]string{
		LabelName:      OperatorName,
		LabelInstance:  cluster.Name,
		LabelManagedBy: OperatorName,
		LabelComponent: "database",
		LabelCluster:   cluster.Name,
		LabelVersion:   fmt.Sprintf("%d", cluster.Spec.PostgresVersion),
	}
}

// SelectorLabels returns the minimal label set used as the StatefulSet pod
// selector. These labels are immutable after creation.
func SelectorLabels(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	return map[string]string{
		LabelName:    OperatorName,
		LabelInstance: cluster.Name,
		LabelCluster: cluster.Name,
	}
}

// PodLabels returns labels for a specific pod role within the cluster. The
// role parameter should be RolePrimary or RoleReplica.
func PodLabels(cluster *pgv1alpha1.PostgresCluster, role string) map[string]string {
	labels := CommonLabels(cluster)
	labels[LabelRole] = role
	return labels
}

// PrimaryPodLabels returns the labels applied to the primary pod.
func PrimaryPodLabels(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	return PodLabels(cluster, RolePrimary)
}

// ReplicaPodLabels returns the labels applied to replica pods.
func ReplicaPodLabels(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	return PodLabels(cluster, RoleReplica)
}

// MergeLabels merges two label maps, with values in overlay taking precedence.
func MergeLabels(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}
