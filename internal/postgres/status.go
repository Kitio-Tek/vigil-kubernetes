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
	"time"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetClusterCondition upserts a condition on the PostgresCluster status. If a
// condition of the given type already exists it is updated in place; otherwise a
// new condition is appended.
func SetClusterCondition(
	cluster *pgv1alpha1.PostgresCluster,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	now := metav1.NewTime(time.Now())

	for i, c := range cluster.Status.Conditions {
		if c.Type == conditionType {
			// Preserve the last-transition time when the status has not changed.
			if c.Status == status {
				now = c.LastTransitionTime
			}
			cluster.Status.Conditions[i] = metav1.Condition{
				Type:               conditionType,
				Status:             status,
				ObservedGeneration: cluster.Generation,
				LastTransitionTime: now,
				Reason:             reason,
				Message:            message,
			}
			return
		}
	}

	cluster.Status.Conditions = append(cluster.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: cluster.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}

// IsClusterReady reports whether the cluster has a Ready condition set to True.
func IsClusterReady(cluster *pgv1alpha1.PostgresCluster) bool {
	for _, c := range cluster.Status.Conditions {
		if c.Type == pgv1alpha1.ConditionReady && c.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsClusterRunning reports whether the cluster phase is Running.
func IsClusterRunning(cluster *pgv1alpha1.PostgresCluster) bool {
	return cluster.Status.Phase == pgv1alpha1.PhaseRunning
}

// SetBackupCondition upserts a condition on a PostgresBackup status.
func SetBackupCondition(
	backup *pgv1alpha1.PostgresBackup,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	now := metav1.NewTime(time.Now())

	for i, c := range backup.Status.Conditions {
		if c.Type == conditionType {
			if c.Status == status {
				now = c.LastTransitionTime
			}
			backup.Status.Conditions[i] = metav1.Condition{
				Type:               conditionType,
				Status:             status,
				ObservedGeneration: backup.Generation,
				LastTransitionTime: now,
				Reason:             reason,
				Message:            message,
			}
			return
		}
	}

	backup.Status.Conditions = append(backup.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: backup.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}

// SetUserCondition upserts a condition on a PostgresUser status.
func SetUserCondition(
	user *pgv1alpha1.PostgresUser,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	now := metav1.NewTime(time.Now())

	for i, c := range user.Status.Conditions {
		if c.Type == conditionType {
			if c.Status == status {
				now = c.LastTransitionTime
			}
			user.Status.Conditions[i] = metav1.Condition{
				Type:               conditionType,
				Status:             status,
				ObservedGeneration: user.Generation,
				LastTransitionTime: now,
				Reason:             reason,
				Message:            message,
			}
			return
		}
	}

	user.Status.Conditions = append(user.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: user.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}
