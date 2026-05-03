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
	"k8s.io/apimachinery/pkg/api/resource"
)

// TuningProfile controls the default parameter set applied to a cluster when
// no explicit PostgreSQL parameters are provided.
type TuningProfile string

const (
	// TuningProfileDefault produces conservative settings suitable for most
	// workloads without specific performance requirements.
	TuningProfileDefault TuningProfile = "default"

	// TuningProfileOLTP tunes for many short-lived read/write transactions.
	TuningProfileOLTP TuningProfile = "oltp"

	// TuningProfileAnalytics tunes for large sequential scans and aggregations.
	TuningProfileAnalytics TuningProfile = "analytics"
)

// AutoTune computes a set of postgresql.conf parameters derived from the
// resource requests of the PostgreSQL container. The result is merged with
// the user-supplied parameters, with user values taking precedence.
func AutoTune(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	params := map[string]string{}

	memBytes := containerMemoryBytes(cluster)
	if memBytes <= 0 {
		return params
	}

	// shared_buffers: 25% of total memory, capped at 8 GB.
	sharedBuffers := memBytes / 4
	const maxSharedBuffers int64 = 8 * 1024 * 1024 * 1024
	if sharedBuffers > maxSharedBuffers {
		sharedBuffers = maxSharedBuffers
	}
	params["shared_buffers"] = fmtMemory(sharedBuffers)

	// effective_cache_size: 75% of total memory.
	params["effective_cache_size"] = fmtMemory(memBytes * 3 / 4)

	// maintenance_work_mem: 5% of memory, capped at 2 GB.
	maintenanceMem := memBytes / 20
	const maxMaintenanceMem int64 = 2 * 1024 * 1024 * 1024
	if maintenanceMem > maxMaintenanceMem {
		maintenanceMem = maxMaintenanceMem
	}
	params["maintenance_work_mem"] = fmtMemory(maintenanceMem)

	// work_mem: (memory - shared_buffers) / (max_connections * 2).
	// Default to a safe 4 MB.
	params["work_mem"] = "4MB"

	// Checkpoint tuning for durability/performance balance.
	params["checkpoint_completion_target"] = "0.9"
	params["wal_buffers"] = "-1"
	params["default_statistics_target"] = "100"
	params["random_page_cost"] = "1.1"
	params["effective_io_concurrency"] = "200"

	return params
}

// MergeParams merges user-supplied parameters over the auto-tuned base,
// ensuring user values always win.
func MergeWithAutoTune(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	base := AutoTune(cluster)
	for k, v := range cluster.Spec.PostgresParameters {
		base[k] = v
	}
	return base
}

// RequiredConnectionParams returns postgresql.conf parameters that must always
// be set by the operator regardless of user configuration. These cannot be
// overridden by the user.
func RequiredConnectionParams(cluster *pgv1alpha1.PostgresCluster) map[string]string {
	return map[string]string{
		"listen_addresses":               "*",
		"port":                           "5432",
		"max_connections":                fmt.Sprintf("%d", maxConnections(cluster)),
		"superuser_reserved_connections": "3",
	}
}

// maxConnections returns the recommended max_connections for the cluster.
func maxConnections(cluster *pgv1alpha1.PostgresCluster) int32 {
	// When a connection pooler is not configured, keep max_connections generous.
	// With PgBouncer, this can be reduced significantly.
	return 100 * cluster.Spec.Instances
}

// containerMemoryBytes returns the memory limit of the PostgreSQL container in
// bytes. Returns 0 if no memory limit is set.
func containerMemoryBytes(cluster *pgv1alpha1.PostgresCluster) int64 {
	mem := cluster.Spec.Resources.Limits.Memory()
	if mem == nil || mem.IsZero() {
		mem = cluster.Spec.Resources.Requests.Memory()
	}
	if mem == nil {
		return 0
	}
	return mem.Value()
}

// fmtMemory formats a byte count as a PostgreSQL memory string (kB/MB/GB).
func fmtMemory(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%dGB", bytes/gb)
	case bytes >= mb:
		return fmt.Sprintf("%dMB", bytes/mb)
	default:
		return fmt.Sprintf("%dkB", bytes/kb)
	}
}

// ParseMemoryBytes parses a Kubernetes resource quantity string into bytes.
// Returns -1 on error.
func ParseMemoryBytes(quantity string) int64 {
	q, err := resource.ParseQuantity(quantity)
	if err != nil {
		return -1
	}
	return q.Value()
}
