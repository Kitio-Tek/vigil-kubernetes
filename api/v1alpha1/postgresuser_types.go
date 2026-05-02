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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseGrant defines privileges to grant on a specific database.
type DatabaseGrant struct {
	// Name of the database.
	Name string `json:"name"`

	// Privileges to grant (e.g. ["SELECT", "INSERT", "ALL PRIVILEGES"]).
	// +optional
	Privileges []string `json:"privileges,omitempty"`
}

// PostgresUserSpec defines the desired state of PostgresUser.
type PostgresUserSpec struct {
	// ClusterName references the PostgresCluster this user belongs to.
	// +kubebuilder:validation:MinLength=1
	ClusterName string `json:"clusterName"`

	// PasswordSecret references a Secret containing a "password" key.
	// When omitted the user is created without a password.
	// +optional
	PasswordSecret *corev1.LocalObjectReference `json:"passwordSecret,omitempty"`

	// Databases lists the databases this user may access and the privileges
	// to grant on each.
	// +optional
	Databases []DatabaseGrant `json:"databases,omitempty"`

	// Roles lists PostgreSQL role names to grant to this user.
	// +optional
	Roles []string `json:"roles,omitempty"`

	// Superuser grants superuser privileges when true.
	// +kubebuilder:default=false
	// +optional
	Superuser bool `json:"superuser,omitempty"`

	// ConnectionLimit sets the maximum number of simultaneous connections.
	// -1 means no limit.
	// +kubebuilder:default=-1
	// +optional
	ConnectionLimit int32 `json:"connectionLimit,omitempty"`
}

// PostgresUserStatus defines the observed state of PostgresUser.
type PostgresUserStatus struct {
	// Applied indicates whether the user definition has been applied to the
	// database successfully.
	// +optional
	Applied bool `json:"applied,omitempty"`

	// Conditions represent the latest available observations of the user state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the .metadata.generation that was last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.clusterName"
// +kubebuilder:printcolumn:name="Applied",type="boolean",JSONPath=".status.applied"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=pgu;pguser,categories=vigil

// PostgresUser is the Schema for the postgresusers API.
type PostgresUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresUserSpec   `json:"spec,omitempty"`
	Status PostgresUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresUserList contains a list of PostgresUser.
type PostgresUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresUser{}, &PostgresUserList{})
}
