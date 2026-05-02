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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
	"github.com/Kitio-Tek/vigil/internal/postgres"
)

func newTestBackup(name, clusterName string, method pgv1alpha1.BackupMethod) *pgv1alpha1.PostgresBackup {
	return &pgv1alpha1.PostgresBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: pgv1alpha1.PostgresBackupSpec{
			ClusterName: clusterName,
			Method:      method,
			Online:      true,
		},
	}
}

func newTestClusterForBackup(name string) *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       1,
			Storage: pgv1alpha1.StorageSpec{
				Size: resource.MustParse("10Gi"),
			},
		},
	}
}

func TestBuildBackupJobBaseBackup(t *testing.T) {
	backup := newTestBackup("my-backup", "my-cluster", pgv1alpha1.BackupMethodBaseBackup)
	cluster := newTestClusterForBackup("my-cluster")

	job := postgres.BuildBackupJob(backup, cluster)

	if job.Name != postgres.BackupJobName(backup) {
		t.Errorf("job name = %q, want %q", job.Name, postgres.BackupJobName(backup))
	}
	if job.Namespace != backup.Namespace {
		t.Errorf("job namespace = %q, want %q", job.Namespace, backup.Namespace)
	}
	if len(job.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("backup job must have at least one container")
	}

	cmd := strings.Join(job.Spec.Template.Spec.Containers[0].Command, " ")
	if !strings.Contains(cmd, "pg_basebackup") {
		t.Errorf("basebackup job command should call pg_basebackup, got: %q", cmd)
	}
}

func TestBuildBackupJobPgDump(t *testing.T) {
	backup := newTestBackup("my-dump", "my-cluster", pgv1alpha1.BackupMethodPgDump)
	cluster := newTestClusterForBackup("my-cluster")

	job := postgres.BuildBackupJob(backup, cluster)

	cmd := strings.Join(job.Spec.Template.Spec.Containers[0].Command, " ")
	if !strings.Contains(cmd, "pg_dump") {
		t.Errorf("pgdump job command should call pg_dump, got: %q", cmd)
	}
}

func TestBuildBackupJobLabels(t *testing.T) {
	backup := newTestBackup("my-backup", "my-cluster", pgv1alpha1.BackupMethodBaseBackup)
	cluster := newTestClusterForBackup("my-cluster")

	job := postgres.BuildBackupJob(backup, cluster)

	if job.Labels[postgres.LabelCluster] != cluster.Name {
		t.Errorf("backup job cluster label = %q, want %q",
			job.Labels[postgres.LabelCluster], cluster.Name)
	}
	if job.Labels[postgres.LabelManagedBy] != postgres.OperatorName {
		t.Errorf("backup job managed-by label = %q, want %q",
			job.Labels[postgres.LabelManagedBy], postgres.OperatorName)
	}
}

func TestBuildBackupJobRestartPolicy(t *testing.T) {
	backup := newTestBackup("my-backup", "my-cluster", pgv1alpha1.BackupMethodBaseBackup)
	cluster := newTestClusterForBackup("my-cluster")

	job := postgres.BuildBackupJob(backup, cluster)

	policy := job.Spec.Template.Spec.RestartPolicy
	if policy != "OnFailure" {
		t.Errorf("backup job restart policy = %q, want OnFailure", policy)
	}
}

func TestBuildBackupJobPGPASSWORDEnv(t *testing.T) {
	backup := newTestBackup("my-backup", "my-cluster", pgv1alpha1.BackupMethodBaseBackup)
	cluster := newTestClusterForBackup("my-cluster")

	job := postgres.BuildBackupJob(backup, cluster)
	container := job.Spec.Template.Spec.Containers[0]

	var found bool
	for _, env := range container.Env {
		if env.Name == "PGPASSWORD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("backup container must set PGPASSWORD from the credentials secret")
	}
}
