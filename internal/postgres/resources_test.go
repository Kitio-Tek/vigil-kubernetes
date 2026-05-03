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

package postgres_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

func newFullCluster() *pgv1alpha1.PostgresCluster {
	replicas := int32(3)
	_ = replicas
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       3,
			Storage: pgv1alpha1.StorageSpec{
				Size: resource.MustParse("10Gi"),
			},
			Monitoring: &pgv1alpha1.MonitoringSpec{
				Enabled: true,
				Port:    9187,
			},
		},
	}
}

func TestBuildStatefulSet(t *testing.T) {
	cluster := newFullCluster()
	sts := postgres.BuildStatefulSet(cluster)

	if sts.Name != cluster.Name {
		t.Errorf("StatefulSet name = %q, want %q", sts.Name, cluster.Name)
	}
	if sts.Namespace != cluster.Namespace {
		t.Errorf("StatefulSet namespace = %q, want %q", sts.Namespace, cluster.Namespace)
	}
	if *sts.Spec.Replicas != cluster.Spec.Instances {
		t.Errorf("StatefulSet replicas = %d, want %d", *sts.Spec.Replicas, cluster.Spec.Instances)
	}
	if sts.Spec.ServiceName != postgres.HeadlessServiceName(cluster) {
		t.Errorf("StatefulSet serviceName = %q, want %q",
			sts.Spec.ServiceName, postgres.HeadlessServiceName(cluster))
	}
}

func TestBuildStatefulSetContainers(t *testing.T) {
	cluster := newFullCluster()
	sts := postgres.BuildStatefulSet(cluster)

	containers := sts.Spec.Template.Spec.Containers
	if len(containers) < 1 {
		t.Fatal("expected at least one container")
	}

	// Find postgres container.
	var pg *corev1.Container
	for i := range containers {
		if containers[i].Name == "postgres" {
			pg = &containers[i]
			break
		}
	}
	if pg == nil {
		t.Fatal("postgres container not found")
	}
	if pg.Image == "" {
		t.Error("postgres container image must not be empty")
	}
	if pg.ReadinessProbe == nil {
		t.Error("postgres container must have a readiness probe")
	}
	if pg.LivenessProbe == nil {
		t.Error("postgres container must have a liveness probe")
	}
}

func TestBuildStatefulSetMonitoringEnabled(t *testing.T) {
	cluster := newFullCluster()
	sts := postgres.BuildStatefulSet(cluster)

	var found bool
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "metrics" {
			found = true
			break
		}
	}
	if !found {
		t.Error("metrics sidecar container not found when monitoring is enabled")
	}
}

func TestBuildStatefulSetMonitoringDisabled(t *testing.T) {
	cluster := newFullCluster()
	cluster.Spec.Monitoring = nil
	sts := postgres.BuildStatefulSet(cluster)

	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "metrics" {
			t.Error("metrics sidecar should not be present when monitoring is disabled")
		}
	}
}

func TestBuildStatefulSetVolumeClaimTemplate(t *testing.T) {
	cluster := newFullCluster()
	sts := postgres.BuildStatefulSet(cluster)

	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		t.Fatal("expected at least one VolumeClaimTemplate")
	}
	pvc := sts.Spec.VolumeClaimTemplates[0]
	req := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if req.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Errorf("PVC storage request = %s, want 10Gi", req.String())
	}
}

func TestBuildPrimaryService(t *testing.T) {
	cluster := newFullCluster()
	svc := postgres.BuildPrimaryService(cluster)

	if svc.Name != postgres.PrimaryServiceName(cluster) {
		t.Errorf("primary service name = %q, want %q",
			svc.Name, postgres.PrimaryServiceName(cluster))
	}
	if svc.Spec.ClusterIP == "None" {
		t.Error("primary service must not be headless")
	}
	if svc.Spec.Selector[postgres.LabelRole] != postgres.RolePrimary {
		t.Errorf("primary service selector role = %q, want %q",
			svc.Spec.Selector[postgres.LabelRole], postgres.RolePrimary)
	}
}

func TestBuildReplicaService(t *testing.T) {
	cluster := newFullCluster()
	svc := postgres.BuildReplicaService(cluster)

	if svc.Spec.Selector[postgres.LabelRole] != postgres.RoleReplica {
		t.Errorf("replica service selector role = %q, want %q",
			svc.Spec.Selector[postgres.LabelRole], postgres.RoleReplica)
	}
}

func TestBuildHeadlessService(t *testing.T) {
	cluster := newFullCluster()
	svc := postgres.BuildHeadlessService(cluster)

	if svc.Spec.ClusterIP != "None" {
		t.Errorf("headless service ClusterIP = %q, want None", svc.Spec.ClusterIP)
	}
	if !svc.Spec.PublishNotReadyAddresses {
		t.Error("headless service must publish not-ready addresses")
	}
}

func TestBuildConfigMap(t *testing.T) {
	cluster := newFullCluster()
	cm := postgres.BuildConfigMap(cluster)

	if _, ok := cm.Data["postgresql.conf"]; !ok {
		t.Error("ConfigMap missing postgresql.conf key")
	}
	if _, ok := cm.Data["pg_hba.conf"]; !ok {
		t.Error("ConfigMap missing pg_hba.conf key")
	}
}

func TestBuildServiceAccount(t *testing.T) {
	cluster := newFullCluster()
	sa := postgres.BuildServiceAccount(cluster)

	if sa.Name != postgres.ServiceAccountName(cluster) {
		t.Errorf("ServiceAccount name = %q, want %q",
			sa.Name, postgres.ServiceAccountName(cluster))
	}
	if sa.Namespace != cluster.Namespace {
		t.Errorf("ServiceAccount namespace = %q, want %q", sa.Namespace, cluster.Namespace)
	}
}
