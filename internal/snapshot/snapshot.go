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

// Package snapshot provides helpers for creating and restoring CSI volume
// snapshots of PostgreSQL data PVCs. It uses the v1 VolumeSnapshot API which
// has been GA since Kubernetes 1.20.
package snapshot

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SnapshotSpec captures the per-cluster fields that influence the shape of
// generated snapshots. Other fields like StorageClassName are read off the
// source PVC at runtime.
type SnapshotSpec struct {
	ClusterName       string
	Namespace         string
	SnapshotClassName string
	SourcePVC         string
	Suffix            string
}

// Manifest is a minimal in-memory representation of a VolumeSnapshot. It is
// intentionally not a typed snapshot.storage.k8s.io API object so the package
// can be used without pulling in that dependency in unit tests; the
// controller layer converts it to the proper type before applying.
type Manifest struct {
	APIVersion        string
	Kind              string
	Name              string
	Namespace         string
	Labels            map[string]string
	SnapshotClassName string
	SourcePVC         string
	CreationTimestamp metav1.Time
}

// Build returns a snapshot manifest for the given spec at the given time.
// The name embeds the source PVC name and a sortable timestamp suffix so
// listing snapshots in lexical order yields chronological order.
func Build(spec SnapshotSpec, now time.Time) (Manifest, error) {
	if spec.ClusterName == "" {
		return Manifest{}, fmt.Errorf("snapshot: cluster name is required")
	}
	if spec.SourcePVC == "" {
		return Manifest{}, fmt.Errorf("snapshot: source PVC is required")
	}
	suffix := spec.Suffix
	if suffix == "" {
		suffix = now.UTC().Format("20060102t150405")
	}
	return Manifest{
		APIVersion:        "snapshot.storage.k8s.io/v1",
		Kind:              "VolumeSnapshot",
		Name:              joinName(spec.SourcePVC, suffix),
		Namespace:         spec.Namespace,
		Labels:            buildLabels(spec.ClusterName, spec.SourcePVC),
		SnapshotClassName: spec.SnapshotClassName,
		SourcePVC:         spec.SourcePVC,
		CreationTimestamp: metav1.NewTime(now.UTC()),
	}, nil
}

func joinName(pvc, suffix string) string {
	pvc = strings.ToLower(strings.TrimSpace(pvc))
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	if suffix == "" {
		return pvc
	}
	return pvc + "-snap-" + suffix
}

func buildLabels(cluster, pvc string) map[string]string {
	return map[string]string{
		"pg.vigil.io/cluster":         cluster,
		"pg.vigil.io/source-pvc":      pvc,
		"app.kubernetes.io/component": "snapshot",
		"app.kubernetes.io/managed-by": "vigil",
	}
}

// RetentionPolicy expresses the maximum number of snapshots to keep and the
// maximum age beyond which snapshots are eligible for deletion.
type RetentionPolicy struct {
	MaxCount int
	MaxAge   time.Duration
}

// Apply returns the subset of snapshots that should be deleted to satisfy the
// retention policy. Snapshots are evaluated in chronological order: the
// newest are kept and the oldest removed first.
func (p RetentionPolicy) Apply(snaps []Manifest, now time.Time) []Manifest {
	sorted := make([]Manifest, len(snaps))
	copy(sorted, snaps)
	// Sort newest first.
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].CreationTimestamp.Time.After(sorted[i].CreationTimestamp.Time) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var toDelete []Manifest
	for i, s := range sorted {
		if p.MaxCount > 0 && i >= p.MaxCount {
			toDelete = append(toDelete, s)
			continue
		}
		if p.MaxAge > 0 && now.Sub(s.CreationTimestamp.Time) > p.MaxAge {
			toDelete = append(toDelete, s)
		}
	}
	return toDelete
}

// IsCompleted reports whether the snapshot is ready to be used for restore.
// In production this is decided by inspecting the .status.readyToUse field
// of a real VolumeSnapshot; the helper exists for completeness.
func IsCompleted(readyToUse *bool) bool {
	return readyToUse != nil && *readyToUse
}
