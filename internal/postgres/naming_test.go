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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

func newTestCluster(name string) *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
		},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       1,
		},
	}
}

func TestClusterStatefulSetName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.ClusterStatefulSetName(c)
	if got != "mycluster" {
		t.Errorf("got %q, want mycluster", got)
	}
}

func TestPrimaryServiceName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.PrimaryServiceName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("primary service name should start with cluster name, got %q", got)
	}
	if !strings.Contains(got, "primary") {
		t.Errorf("primary service name should contain 'primary', got %q", got)
	}
}

func TestReplicaServiceName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.ReplicaServiceName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("replica service name should start with cluster name, got %q", got)
	}
}

func TestHeadlessServiceName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.HeadlessServiceName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("headless service name should start with cluster name, got %q", got)
	}
}

func TestConfigMapName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.ConfigMapName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("configmap name should start with cluster name, got %q", got)
	}
}

func TestSecretName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.SecretName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("secret name should start with cluster name, got %q", got)
	}
}

func TestServiceAccountNameDefault(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.ServiceAccountName(c)
	if !strings.HasPrefix(got, "mycluster") {
		t.Errorf("serviceaccount name should start with cluster name when not overridden, got %q", got)
	}
}

func TestServiceAccountNameOverride(t *testing.T) {
	c := newTestCluster("mycluster")
	c.Spec.ServiceAccountName = "custom-sa"
	got := postgres.ServiceAccountName(c)
	if got != "custom-sa" {
		t.Errorf("got %q, want custom-sa", got)
	}
}

func TestBackupJobName(t *testing.T) {
	backup := &pgv1alpha1.PostgresBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "my-backup"},
	}
	got := postgres.BackupJobName(backup)
	if !strings.HasPrefix(got, "my-backup") {
		t.Errorf("backup job name should start with backup name, got %q", got)
	}
}

func TestPodName(t *testing.T) {
	c := newTestCluster("mycluster")
	got := postgres.PodName(c, 0)
	if got != "mycluster-0" {
		t.Errorf("got %q, want mycluster-0", got)
	}
	got2 := postgres.PodName(c, 2)
	if got2 != "mycluster-2" {
		t.Errorf("got %q, want mycluster-2", got2)
	}
}

func TestNamesAreUnique(t *testing.T) {
	c := newTestCluster("x")
	names := []string{
		postgres.PrimaryServiceName(c),
		postgres.ReplicaServiceName(c),
		postgres.HeadlessServiceName(c),
		postgres.ConfigMapName(c),
		postgres.SecretName(c),
	}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate name %q", n)
		}
		seen[n] = true
	}
}
