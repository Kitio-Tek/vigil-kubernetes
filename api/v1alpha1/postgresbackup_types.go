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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupMethod selects the PostgreSQL backup tool.
type BackupMethod string

const (
	// BackupMethodBaseBackup performs a streaming base backup via pg_basebackup.
	BackupMethodBaseBackup BackupMethod = "basebackup"
	// BackupMethodPgDump performs a logical backup via pg_dump.
	BackupMethodPgDump BackupMethod = "pgdump"
)

// BackupPhase describes the current phase of a PostgresBackup.
type BackupPhase string

const (
	// BackupPhasePending indicates the backup has not started yet.
	BackupPhasePending BackupPhase = "Pending"
	// BackupPhaseRunning indicates the backup job is active.
	BackupPhaseRunning BackupPhase = "Running"
	// BackupPhaseCompleted indicates the backup finished successfully.
	BackupPhaseCompleted BackupPhase = "Completed"
	// BackupPhaseFailed indicates the backup job failed.
	BackupPhaseFailed BackupPhase = "Failed"
)

// PostgresBackupSpec defines the desired state of PostgresBackup.
type PostgresBackupSpec struct {
	// ClusterName references the PostgresCluster to back up.
	// +kubebuilder:validation:MinLength=1
	ClusterName string `json:"clusterName"`

	// Method selects the backup tool (basebackup or pgdump).
	// +kubebuilder:validation:Enum=basebackup;pgdump
	// +kubebuilder:default=basebackup
	// +optional
	Method BackupMethod `json:"method,omitempty"`

	// Online controls whether the backup is taken from a running hot-standby.
	// +kubebuilder:default=true
	// +optional
	Online bool `json:"online,omitempty"`
}

// PostgresBackupStatus defines the observed state of PostgresBackup.
type PostgresBackupStatus struct {
	// Phase is the current phase of the backup operation.
	// +optional
	Phase BackupPhase `json:"phase,omitempty"`

	// StartTime records when the backup job began.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime records when the backup job finished.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// BackupSize is the size of the backup in bytes.
	// +optional
	BackupSize int64 `json:"backupSize,omitempty"`

	// DestinationPath is the fully-qualified path where the backup was stored.
	// +optional
	DestinationPath string `json:"destinationPath,omitempty"`

	// Conditions represent the latest available observations of the backup state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.clusterName"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Started",type="date",JSONPath=".status.startTime"
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime"
// +kubebuilder:resource:shortName=pgb;pgbackup,categories=vigil

// PostgresBackup is the Schema for the postgresbackups API.
type PostgresBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresBackupSpec   `json:"spec,omitempty"`
	Status PostgresBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresBackupList contains a list of PostgresBackup.
type PostgresBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresBackup{}, &PostgresBackupList{})
}
