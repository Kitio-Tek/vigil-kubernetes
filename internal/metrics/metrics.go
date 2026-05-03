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

// Package metrics registers Prometheus metrics for the Athos operator and
// provides helpers for recording reconcile, upgrade, and backup events.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileTotal counts reconcile loop invocations per controller and result.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "athos",
			Subsystem: "controller",
			Name:      "reconcile_total",
			Help:      "Total number of reconcile loop invocations per controller and result.",
		},
		[]string{"controller", "result"},
	)

	// ReconcileDuration observes the duration of each reconcile loop.
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "athos",
			Subsystem: "controller",
			Name:      "reconcile_duration_seconds",
			Help:      "Duration of reconcile loop invocations in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	// ClustersTotal tracks the number of PostgresCluster objects by phase.
	ClustersTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "athos",
			Subsystem: "cluster",
			Name:      "total",
			Help:      "Number of PostgresCluster objects managed by this operator, by phase.",
		},
		[]string{"phase"},
	)

	// InstancesReady tracks ready PostgreSQL instances across all clusters.
	InstancesReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "athos",
			Subsystem: "cluster",
			Name:      "instances_ready",
			Help:      "Number of ready PostgreSQL instances per cluster.",
		},
		[]string{"cluster", "namespace"},
	)

	// BackupsTotal counts backup attempts per cluster and status.
	BackupsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "athos",
			Subsystem: "backup",
			Name:      "total",
			Help:      "Total backup attempts per cluster and terminal status.",
		},
		[]string{"cluster", "namespace", "status"},
	)

	// BackupDuration observes how long backup jobs take.
	BackupDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "athos",
			Subsystem: "backup",
			Name:      "duration_seconds",
			Help:      "Duration of backup jobs in seconds.",
			Buckets:   []float64{30, 60, 120, 300, 600, 1200, 3600},
		},
		[]string{"cluster", "namespace"},
	)

	// UpgradesTotal counts upgrade attempts by outcome.
	UpgradesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "athos",
			Subsystem: "upgrade",
			Name:      "total",
			Help:      "Total PostgreSQL version upgrade attempts and outcomes.",
		},
		[]string{"cluster", "namespace", "result"},
	)

	// FailoversTotal counts failover events by trigger type.
	FailoversTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "athos",
			Subsystem: "ha",
			Name:      "failovers_total",
			Help:      "Total failover events by trigger type (manual, automatic).",
		},
		[]string{"cluster", "namespace", "trigger"},
	)

	// PasswordRotationsTotal counts secret rotation events.
	PasswordRotationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "athos",
			Subsystem: "security",
			Name:      "password_rotations_total",
			Help:      "Total password rotation events.",
		},
		[]string{"cluster", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileDuration,
		ClustersTotal,
		InstancesReady,
		BackupsTotal,
		BackupDuration,
		UpgradesTotal,
		FailoversTotal,
		PasswordRotationsTotal,
	)
}

// RecordReconcile increments the reconcile total counter for the given
// controller and result label ("success", "error", "requeue").
func RecordReconcile(controller, result string) {
	ReconcileTotal.WithLabelValues(controller, result).Inc()
}

// RecordBackup increments the backup total counter with the given status
// label ("completed", "failed").
func RecordBackup(cluster, namespace, status string) {
	BackupsTotal.WithLabelValues(cluster, namespace, status).Inc()
}

// RecordUpgrade increments the upgrade counter with the given result label
// ("completed", "failed").
func RecordUpgrade(cluster, namespace, result string) {
	UpgradesTotal.WithLabelValues(cluster, namespace, result).Inc()
}

// RecordFailover increments the failover counter with the trigger type
// ("manual", "automatic").
func RecordFailover(cluster, namespace, trigger string) {
	FailoversTotal.WithLabelValues(cluster, namespace, trigger).Inc()
}

// SetInstancesReady sets the ready instance gauge for a cluster.
func SetInstancesReady(cluster, namespace string, count int) {
	InstancesReady.WithLabelValues(cluster, namespace).Set(float64(count))
}

// SetClustersPhase sets the cluster count gauge for a given phase.
func SetClustersPhase(phase string, count int) {
	ClustersTotal.WithLabelValues(phase).Set(float64(count))
}
