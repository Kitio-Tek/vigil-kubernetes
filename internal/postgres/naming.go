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

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
)

// ClusterStatefulSetName returns the name of the StatefulSet for a cluster.
func ClusterStatefulSetName(cluster *pgv1alpha1.PostgresCluster) string {
	return cluster.Name
}

// PrimaryServiceName returns the name of the Service that routes writes to the
// primary instance.
func PrimaryServiceName(cluster *pgv1alpha1.PostgresCluster) string {
	return fmt.Sprintf("%s-primary", cluster.Name)
}

// ReplicaServiceName returns the name of the Service that routes reads to replica
// instances.
func ReplicaServiceName(cluster *pgv1alpha1.PostgresCluster) string {
	return fmt.Sprintf("%s-replicas", cluster.Name)
}

// HeadlessServiceName returns the name of the headless Service used for DNS-based
// pod discovery within the StatefulSet.
func HeadlessServiceName(cluster *pgv1alpha1.PostgresCluster) string {
	return fmt.Sprintf("%s-pods", cluster.Name)
}

// ConfigMapName returns the name of the ConfigMap that holds postgresql.conf and
// pg_hba.conf.
func ConfigMapName(cluster *pgv1alpha1.PostgresCluster) string {
	return fmt.Sprintf("%s-config", cluster.Name)
}

// SecretName returns the name of the Secret that holds the superuser password and
// connection credentials.
func SecretName(cluster *pgv1alpha1.PostgresCluster) string {
	return fmt.Sprintf("%s-credentials", cluster.Name)
}

// ServiceAccountName returns the name of the ServiceAccount used by cluster pods
// when no explicit ServiceAccountName is set in the spec.
func ServiceAccountName(cluster *pgv1alpha1.PostgresCluster) string {
	if cluster.Spec.ServiceAccountName != "" {
		return cluster.Spec.ServiceAccountName
	}
	return fmt.Sprintf("%s-sa", cluster.Name)
}

// BackupJobName returns the name of the Job created for a PostgresBackup.
func BackupJobName(backup *pgv1alpha1.PostgresBackup) string {
	return fmt.Sprintf("%s-backup", backup.Name)
}

// PodName returns the stable DNS name of instance n inside the StatefulSet.
func PodName(cluster *pgv1alpha1.PostgresCluster, n int) string {
	return fmt.Sprintf("%s-%d", cluster.Name, n)
}

// PodFQDN returns the fully qualified DNS name of instance n inside the cluster.
func PodFQDN(cluster *pgv1alpha1.PostgresCluster, n int) string {
	return fmt.Sprintf("%s.%s.%s.svc.cluster.local",
		PodName(cluster, n),
		HeadlessServiceName(cluster),
		cluster.Namespace,
	)
}
