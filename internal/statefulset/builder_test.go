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

package statefulset_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/statefulset"
)

const stsTestClusterName = "pg"

func newCluster(instances int32) *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsTestClusterName,
			Namespace: "default",
		},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       instances,
			Storage: pgv1alpha1.StorageSpec{
				Size: resource.MustParse("10Gi"),
			},
		},
	}
}

func TestBuild_Replicas(t *testing.T) {
	c := newCluster(3)
	sts := statefulset.Build(c, statefulset.Options{})
	if *sts.Spec.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", *sts.Spec.Replicas)
	}
}

func TestBuild_SingleReplica(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})
	if *sts.Spec.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1", *sts.Spec.Replicas)
	}
}

func TestBuild_PostgresContainer(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})

	containers := sts.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("no containers in StatefulSet")
	}
	pg := containers[0]
	if pg.Name != "postgres" {
		t.Errorf("container name = %q, want postgres", pg.Name)
	}
	if pg.Image == "" {
		t.Error("postgres container image is empty")
	}
}

func TestBuild_DataVolumeClaim(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})

	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		t.Fatal("no VolumeClaimTemplates")
	}
	pvc := sts.Spec.VolumeClaimTemplates[0]
	if pvc.Name != statefulset.DataVolumeName {
		t.Errorf("PVC name = %q, want %q", pvc.Name, statefulset.DataVolumeName)
	}
	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if storage.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Errorf("PVC storage = %s, want 10Gi", storage.String())
	}
}

func TestBuild_WALVolumeConst(t *testing.T) {
	// Verify the WAL volume name constant is exported for external use.
	if statefulset.WALVolumeName == "" {
		t.Error("WALVolumeName constant should not be empty")
	}
}

func TestBuild_ConfigMapVolume(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{
		ConfigMapName: "pg-config",
	})

	found := false
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == statefulset.ConfigVolumeName {
			found = true
			if v.ConfigMap.Name != "pg-config" {
				t.Errorf("config volume uses ConfigMap %q, want pg-config", v.ConfigMap.Name)
			}
		}
	}
	if !found {
		t.Error("config volume not found")
	}
}

func TestBuild_TLSVolume(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{
		TLSSecretName: "pg-tls",
	})

	found := false
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == statefulset.CertVolumeName {
			found = true
			if v.Secret.SecretName != "pg-tls" {
				t.Errorf("TLS volume uses Secret %q, want pg-tls", v.Secret.SecretName)
			}
		}
	}
	if !found {
		t.Error("TLS cert volume not found")
	}
}

func TestBuild_SidecarContainer(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{
		Sidecars: []corev1.Container{
			{Name: "exporter", Image: "prom/postgres-exporter:v0.15.0"},
		},
	})

	if len(sts.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(sts.Spec.Template.Spec.Containers))
	}
	if sts.Spec.Template.Spec.Containers[1].Name != "exporter" {
		t.Error("sidecar container not added correctly")
	}
}

func TestBuild_AntiAffinity_MultiReplica(t *testing.T) {
	c := newCluster(3)
	sts := statefulset.Build(c, statefulset.Options{})

	if sts.Spec.Template.Spec.Affinity == nil {
		t.Error("expected anti-affinity for multi-replica cluster")
	}
	aa := sts.Spec.Template.Spec.Affinity.PodAntiAffinity
	if aa == nil || len(aa.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
		t.Error("expected preferred pod anti-affinity rules")
	}
}

func TestBuild_NoAntiAffinity_SingleReplica(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})

	if sts.Spec.Template.Spec.Affinity != nil {
		t.Error("single-replica cluster should not have anti-affinity")
	}
}

func TestBuild_SecurityContext(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})

	sc := sts.Spec.Template.Spec.SecurityContext
	if sc == nil {
		t.Fatal("pod security context is nil")
	}
	if sc.RunAsUser == nil || *sc.RunAsUser != 999 {
		t.Error("expected RunAsUser=999")
	}
}

func TestBuild_LivenessReadinessProbes(t *testing.T) {
	c := newCluster(1)
	sts := statefulset.Build(c, statefulset.Options{})

	pg := sts.Spec.Template.Spec.Containers[0]
	if pg.LivenessProbe == nil {
		t.Error("liveness probe is nil")
	}
	if pg.ReadinessProbe == nil {
		t.Error("readiness probe is nil")
	}
}

func TestOrdinalFromName(t *testing.T) {
	cases := []struct {
		pod, sts string
		want     int
	}{
		{"mycluster-0", "mycluster", 0},
		{"mycluster-1", "mycluster", 1},
		{"mycluster-12", "mycluster", 12},
		{"other-0", "mycluster", -1},
		{"mycluster-", "mycluster", -1},
		{"mycluster-abc", "mycluster", -1},
	}
	for _, tc := range cases {
		got := statefulset.OrdinalFromName(tc.pod, tc.sts)
		if got != tc.want {
			t.Errorf("OrdinalFromName(%q, %q) = %d, want %d", tc.pod, tc.sts, got, tc.want)
		}
	}
}
