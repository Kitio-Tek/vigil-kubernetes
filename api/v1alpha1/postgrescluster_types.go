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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresClusterSpec defines the desired state of PostgresCluster.
type PostgresClusterSpec struct {
	// PostgreSQL major version (14, 15, 16, or 17).
	// +kubebuilder:validation:Minimum=14
	// +kubebuilder:validation:Maximum=17
	// +kubebuilder:default=16
	PostgresVersion int32 `json:"postgresVersion"`

	// Number of PostgreSQL instances (primary + replicas).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	Instances int32 `json:"instances"`

	// Storage configuration for the data volume.
	Storage StorageSpec `json:"storage"`

	// Resources for PostgreSQL containers.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// PostgreSQL configuration parameters (postgresql.conf overrides).
	// +optional
	PostgresParameters map[string]string `json:"postgresParameters,omitempty"`

	// HBA rules appended to pg_hba.conf.
	// +optional
	PostgresHBA []string `json:"postgresHBA,omitempty"`

	// Backup configuration.
	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`

	// TLS configuration for PostgreSQL connections.
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`

	// Monitoring enables the Prometheus metrics sidecar.
	// +optional
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`

	// TopologySpreadConstraints control how pods are spread across failure domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// Affinity rules for pod scheduling.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations for pod scheduling.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// ImagePullSecrets for pulling the PostgreSQL image.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ServiceAccountName for the PostgreSQL pods. If empty, a dedicated
	// ServiceAccount is created for the cluster.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// PriorityClassName for the PostgreSQL pods.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Paused suspends all reconciliation when true.
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// StorageSpec defines the PVC configuration for the PostgreSQL data volume.
type StorageSpec struct {
	// Size of the PVC (e.g. "10Gi").
	// +kubebuilder:validation:Pattern=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	Size resource.Quantity `json:"size"`

	// StorageClass for the PVC. Uses the cluster default when omitted.
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// AccessModes for the PVC. Defaults to ReadWriteOnce when omitted.
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
}

// BackupSpec defines the backup schedule and destination for a cluster.
type BackupSpec struct {
	// Enabled toggles automatic scheduled backups.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// RetentionPolicy controls how long backups are kept (e.g. "7d", "30d").
	// +kubebuilder:default="7d"
	// +optional
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// Schedule is a cron expression for the backup schedule (e.g. "0 2 * * *").
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Destination configures where backups are stored.
	// +optional
	Destination *BackupDestinationSpec `json:"destination,omitempty"`
}

// BackupDestinationSpec selects the backend for backup storage.
type BackupDestinationSpec struct {
	// S3 stores backups in an S3-compatible object store.
	// +optional
	S3 *S3BackupSpec `json:"s3,omitempty"`

	// GCS stores backups in Google Cloud Storage.
	// +optional
	GCS *GCSBackupSpec `json:"gcs,omitempty"`
}

// S3BackupSpec holds credentials and addressing for S3-compatible storage.
type S3BackupSpec struct {
	// Bucket name.
	Bucket string `json:"bucket"`

	// Region of the S3 bucket.
	// +optional
	Region string `json:"region,omitempty"`

	// Endpoint for S3-compatible stores (e.g. MinIO).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Path prefix within the bucket.
	// +optional
	Path string `json:"path,omitempty"`

	// CredentialsSecret references a Secret containing AWS_ACCESS_KEY_ID
	// and AWS_SECRET_ACCESS_KEY keys.
	// +optional
	CredentialsSecret *corev1.LocalObjectReference `json:"credentialsSecret,omitempty"`
}

// GCSBackupSpec holds credentials and addressing for Google Cloud Storage.
type GCSBackupSpec struct {
	// Bucket name.
	Bucket string `json:"bucket"`

	// Path prefix within the bucket.
	// +optional
	Path string `json:"path,omitempty"`

	// CredentialsSecret references a Secret containing a GCS JSON credentials file.
	// +optional
	CredentialsSecret *corev1.LocalObjectReference `json:"credentialsSecret,omitempty"`
}

// TLSSpec configures TLS for PostgreSQL client connections.
type TLSSpec struct {
	// Enabled toggles TLS enforcement.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// CertificateSecret references a Secret with tls.crt and tls.key entries.
	// +optional
	CertificateSecret *corev1.LocalObjectReference `json:"certificateSecret,omitempty"`

	// CASecret references a Secret with a ca.crt entry used for client verification.
	// +optional
	CASecret *corev1.LocalObjectReference `json:"caSecret,omitempty"`
}

// MonitoringSpec configures the Prometheus metrics exporter sidecar.
type MonitoringSpec struct {
	// Enabled toggles the postgres_exporter sidecar.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Port on which the metrics endpoint is exposed.
	// +kubebuilder:default=9187
	// +optional
	Port int32 `json:"port,omitempty"`
}

// PostgresClusterStatus defines the observed state of PostgresCluster.
type PostgresClusterStatus struct {
	// Conditions represent the latest available observations of the cluster state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is the current lifecycle phase of the cluster.
	// +optional
	Phase PostgresClusterPhase `json:"phase,omitempty"`

	// ReadyInstances is the count of PostgreSQL pods that are ready.
	// +optional
	ReadyInstances int32 `json:"readyInstances,omitempty"`

	// CurrentPrimary is the pod name of the active primary instance.
	// +optional
	CurrentPrimary string `json:"currentPrimary,omitempty"`

	// WriteServiceName is the name of the Service routing writes to the primary.
	// +optional
	WriteServiceName string `json:"writeServiceName,omitempty"`

	// ReadServiceName is the name of the Service routing reads to replicas.
	// +optional
	ReadServiceName string `json:"readServiceName,omitempty"`

	// PostgresVersion is the running PostgreSQL version string reported by the cluster.
	// +optional
	PostgresVersion string `json:"postgresVersion,omitempty"`

	// ObservedGeneration is the .metadata.generation that was last fully reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LatestBackup is the timestamp of the last successful backup.
	// +optional
	LatestBackup *metav1.Time `json:"latestBackup,omitempty"`
}

// PostgresClusterPhase describes the lifecycle phase of a PostgresCluster.
type PostgresClusterPhase string

const (
	// PhaseInitializing indicates the cluster resources are being created.
	PhaseInitializing PostgresClusterPhase = "Initializing"
	// PhaseCreating indicates the StatefulSet pods are starting up.
	PhaseCreating PostgresClusterPhase = "Creating"
	// PhaseRunning indicates all instances are ready and the cluster is healthy.
	PhaseRunning PostgresClusterPhase = "Running"
	// PhaseDegraded indicates the cluster is operational but fewer than desired
	// instances are ready.
	PhaseDegraded PostgresClusterPhase = "Degraded"
	// PhaseFailed indicates the cluster has encountered a terminal error.
	PhaseFailed PostgresClusterPhase = "Failed"
	// PhasePaused indicates reconciliation is suspended.
	PhasePaused PostgresClusterPhase = "Paused"
	// PhaseUpgrading indicates a rolling upgrade is in progress.
	PhaseUpgrading PostgresClusterPhase = "Upgrading"
)

// Condition type constants used in PostgresClusterStatus.Conditions.
const (
	// ConditionReady reports that the cluster is fully operational.
	ConditionReady = "Ready"
	// ConditionAvailable reports that at least one instance is serving traffic.
	ConditionAvailable = "Available"
	// ConditionProgressing reports that a state transition is in progress.
	ConditionProgressing = "Progressing"
	// ConditionDegraded reports that the cluster is operating below desired capacity.
	ConditionDegraded = "Degraded"
	// ConditionPrimaryReady reports that the primary instance is accepting connections.
	ConditionPrimaryReady = "PrimaryReady"
)

// Finalizer applied to every PostgresCluster to allow orderly cleanup.
const PostgresClusterFinalizer = "pg.vigil.io/finalizer"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Primary",type="string",JSONPath=".status.currentPrimary"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyInstances"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=pgc;pgcluster,categories=vigil

// PostgresCluster is the Schema for the postgresclusters API.
type PostgresCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresClusterSpec   `json:"spec,omitempty"`
	Status PostgresClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresClusterList contains a list of PostgresCluster.
type PostgresClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresCluster{}, &PostgresClusterList{})
}
