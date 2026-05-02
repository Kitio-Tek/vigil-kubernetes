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

// Package upgrade contains the version upgrade logic for the Vigil operator.
// It supports in-place minor-version upgrades (image tag bumps) and validates
// major version upgrade paths before triggering a pg_upgrade Job.
package upgrade

import (
	"fmt"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
)

const (
	// MinSupportedVersion is the oldest PostgreSQL major version the operator supports.
	MinSupportedVersion int32 = 14

	// MaxSupportedVersion is the latest PostgreSQL major version the operator supports.
	MaxSupportedVersion int32 = 17

	// AnnotationUpgradeApproved is set by an operator/user to approve a major
	// version upgrade. The operator will not start a major upgrade without it.
	AnnotationUpgradeApproved = "pg.vigil.io/upgrade-approved"

	// LabelUpgradeInProgress is set on the cluster while a major upgrade job is
	// running, to prevent concurrent reconcile from modifying the StatefulSet.
	LabelUpgradeInProgress = "pg.vigil.io/upgrade-in-progress"
)

// Kind classifies an upgrade request.
type Kind int

const (
	// KindNone means no version change is required.
	KindNone Kind = iota
	// KindMinor means only the image tag changes (same major version).
	KindMinor
	// KindMajor means the PostgreSQL major version is changing.
	KindMajor
)

func (k Kind) String() string {
	switch k {
	case KindMinor:
		return "minor"
	case KindMajor:
		return "major"
	default:
		return "none"
	}
}

// Plan describes what kind of version change is needed and whether it is
// permitted under the current cluster state.
type Plan struct {
	Kind        Kind
	FromVersion int32
	ToVersion   int32
}

// Classify returns a Plan describing the upgrade path from currentVersion to
// the version requested in the spec.
func Classify(cluster *pgv1alpha1.PostgresCluster, currentVersion int32) (Plan, error) {
	target := cluster.Spec.PostgresVersion
	if err := ValidateVersion(target); err != nil {
		return Plan{}, err
	}
	if target == currentVersion {
		return Plan{Kind: KindNone, FromVersion: currentVersion, ToVersion: target}, nil
	}
	if target < currentVersion {
		return Plan{}, fmt.Errorf("downgrading PostgreSQL from version %d to %d is not supported",
			currentVersion, target)
	}
	if target-currentVersion == 0 {
		return Plan{Kind: KindNone, FromVersion: currentVersion, ToVersion: target}, nil
	}
	if currentVersion/1 == target/1 {
		return Plan{Kind: KindMinor, FromVersion: currentVersion, ToVersion: target}, nil
	}
	return Plan{Kind: KindMajor, FromVersion: currentVersion, ToVersion: target}, nil
}

// ValidateVersion returns an error if the given PostgreSQL major version is not
// supported by this operator release.
func ValidateVersion(version int32) error {
	if version < MinSupportedVersion || version > MaxSupportedVersion {
		return fmt.Errorf("PostgreSQL major version %d is not supported; supported range is [%d, %d]",
			version, MinSupportedVersion, MaxSupportedVersion)
	}
	return nil
}

// MajorUpgradeApproved returns true when the cluster has the upgrade approval
// annotation set. This prevents accidental major upgrades during normal
// spec changes.
func MajorUpgradeApproved(cluster *pgv1alpha1.PostgresCluster) bool {
	if cluster.GetAnnotations() == nil {
		return false
	}
	return cluster.GetAnnotations()[AnnotationUpgradeApproved] == "true"
}

// UpgradeInProgress returns true when the cluster is currently undergoing a
// major version upgrade.
func UpgradeInProgress(cluster *pgv1alpha1.PostgresCluster) bool {
	if cluster.GetLabels() == nil {
		return false
	}
	return cluster.GetLabels()[LabelUpgradeInProgress] == "true"
}

// ImageTag returns the canonical PostgreSQL image tag for the given major
// version. The image is sourced from the official Docker Hub postgres image.
func ImageTag(majorVersion int32) string {
	return fmt.Sprintf("postgres:%d", majorVersion)
}

// ImageTagAlpine returns the Alpine variant of the PostgreSQL image.
func ImageTagAlpine(majorVersion int32) string {
	return fmt.Sprintf("postgres:%d-alpine", majorVersion)
}

// UpgradeJobName returns the name of the pg_upgrade Job for a given upgrade
// transition.
func UpgradeJobName(clusterName string, fromVersion, toVersion int32) string {
	return fmt.Sprintf("%s-upgrade-%d-to-%d", clusterName, fromVersion, toVersion)
}

// SupportedVersions returns all PostgreSQL major versions supported by this
// operator release.
func SupportedVersions() []int32 {
	var versions []int32
	for v := MinSupportedVersion; v <= MaxSupportedVersion; v++ {
		versions = append(versions, v)
	}
	return versions
}
