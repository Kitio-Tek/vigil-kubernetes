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
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newValidCluster() *PostgresCluster {
	return &PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       1,
			Storage: StorageSpec{
				Size: resource.MustParse("10Gi"),
			},
		},
	}
}

func TestDefaultSetsVersion(t *testing.T) {
	c := &PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "x"},
		Spec:       PostgresClusterSpec{},
	}
	c.Default()
	if c.Spec.PostgresVersion != 16 {
		t.Errorf("default postgresVersion = %d, want 16", c.Spec.PostgresVersion)
	}
}

func TestDefaultSetsInstances(t *testing.T) {
	c := &PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "x"},
		Spec:       PostgresClusterSpec{PostgresVersion: 16},
	}
	c.Default()
	if c.Spec.Instances != 1 {
		t.Errorf("default instances = %d, want 1", c.Spec.Instances)
	}
}

func TestDefaultSetsStorageSize(t *testing.T) {
	c := &PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "x"},
		Spec:       PostgresClusterSpec{PostgresVersion: 16, Instances: 1},
	}
	c.Default()
	if c.Spec.Storage.Size.IsZero() {
		t.Error("default storage size should not be zero")
	}
}

func TestDefaultSetsMonitoring(t *testing.T) {
	c := newValidCluster()
	c.Spec.Monitoring = nil
	c.Default()
	if c.Spec.Monitoring == nil {
		t.Fatal("default monitoring should be set")
	}
	if !c.Spec.Monitoring.Enabled {
		t.Error("monitoring should be enabled by default")
	}
	if c.Spec.Monitoring.Port != 9187 {
		t.Errorf("default monitoring port = %d, want 9187", c.Spec.Monitoring.Port)
	}
}

func TestValidateCreateValidCluster(t *testing.T) {
	c := newValidCluster()
	_, err := c.ValidateCreate()
	if err != nil {
		t.Errorf("valid cluster should pass validation: %v", err)
	}
}

func TestValidateVersionOutOfRange(t *testing.T) {
	c := newValidCluster()
	c.Spec.PostgresVersion = 13
	_, err := c.ValidateCreate()
	if err == nil {
		t.Error("version 13 should fail validation")
	}
}

func TestValidateVersionUpperBound(t *testing.T) {
	c := newValidCluster()
	c.Spec.PostgresVersion = 18
	_, err := c.ValidateCreate()
	if err == nil {
		t.Error("version 18 should fail validation")
	}
}

func TestValidateInstancesEven(t *testing.T) {
	c := newValidCluster()
	c.Spec.Instances = 2
	_, err := c.ValidateCreate()
	if err == nil {
		t.Error("even instance count > 1 should fail validation")
	}
}

func TestValidateInstancesOdd(t *testing.T) {
	c := newValidCluster()
	c.Spec.Instances = 3
	_, err := c.ValidateCreate()
	if err != nil {
		t.Errorf("odd instance count 3 should pass validation: %v", err)
	}
}

func TestValidateInstancesSingleAllowed(t *testing.T) {
	c := newValidCluster()
	c.Spec.Instances = 1
	_, err := c.ValidateCreate()
	if err != nil {
		t.Errorf("single instance should pass validation: %v", err)
	}
}

func TestValidateBackupInvalidCron(t *testing.T) {
	c := newValidCluster()
	c.Spec.Backup = &BackupSpec{
		Enabled:  true,
		Schedule: "not-a-cron",
	}
	_, err := c.ValidateCreate()
	if err == nil {
		t.Error("invalid cron expression should fail validation")
	}
}

func TestValidateBackupValidCron(t *testing.T) {
	c := newValidCluster()
	c.Spec.Backup = &BackupSpec{
		Enabled:  true,
		Schedule: "0 2 * * *",
	}
	_, err := c.ValidateCreate()
	if err != nil {
		t.Errorf("valid cron expression should pass validation: %v", err)
	}
}

func TestValidateDelete(t *testing.T) {
	c := newValidCluster()
	_, err := c.ValidateDelete()
	if err != nil {
		t.Errorf("delete validation should always pass: %v", err)
	}
}
