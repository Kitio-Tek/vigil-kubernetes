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

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

func clusterWithMemory(limitMem string) *pgv1alpha1.PostgresCluster {
	c := &pgv1alpha1.PostgresCluster{}
	c.Spec.Instances = 3
	if limitMem != "" {
		c.Spec.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(limitMem),
			},
		}
	}
	return c
}

func TestAutoTune_NoMemory(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{}
	params := postgres.AutoTune(cluster)
	if len(params) != 0 {
		t.Errorf("expected empty params with no memory limit, got %v", params)
	}
}

func TestAutoTune_WithMemory(t *testing.T) {
	cluster := clusterWithMemory("4Gi")
	params := postgres.AutoTune(cluster)

	if _, ok := params["shared_buffers"]; !ok {
		t.Error("expected shared_buffers to be set")
	}
	if _, ok := params["effective_cache_size"]; !ok {
		t.Error("expected effective_cache_size to be set")
	}
	if _, ok := params["maintenance_work_mem"]; !ok {
		t.Error("expected maintenance_work_mem to be set")
	}
}

func TestAutoTune_SharedBuffers_Cap(t *testing.T) {
	// Very high memory should cap shared_buffers at 8 GB
	cluster := clusterWithMemory("64Gi")
	params := postgres.AutoTune(cluster)

	if params["shared_buffers"] != "8GB" {
		t.Errorf("shared_buffers should be capped at 8GB for large instances, got %q", params["shared_buffers"])
	}
}

func TestAutoTune_SmallMemory(t *testing.T) {
	cluster := clusterWithMemory("512Mi")
	params := postgres.AutoTune(cluster)

	if params["shared_buffers"] == "" {
		t.Error("shared_buffers should be set for 512Mi")
	}
}

func TestMergeWithAutoTune_UserOverride(t *testing.T) {
	cluster := clusterWithMemory("4Gi")
	cluster.Spec.PostgresParameters = map[string]string{
		"shared_buffers": "256MB",
		"work_mem":       "16MB",
	}
	params := postgres.MergeWithAutoTune(cluster)

	if params["shared_buffers"] != "256MB" {
		t.Errorf("user shared_buffers should override auto-tune, got %q", params["shared_buffers"])
	}
	if params["work_mem"] != "16MB" {
		t.Errorf("user work_mem should override auto-tune, got %q", params["work_mem"])
	}
}

func TestRequiredConnectionParams(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{}
	cluster.Spec.Instances = 1
	params := postgres.RequiredConnectionParams(cluster)

	if params["listen_addresses"] != "*" {
		t.Error("listen_addresses should always be *")
	}
	if params["port"] != "5432" {
		t.Error("port should always be 5432")
	}
	if params["max_connections"] == "" {
		t.Error("max_connections should be set")
	}
}

func TestParseMemoryBytes(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1Gi", 1024 * 1024 * 1024},
		{"512Mi", 512 * 1024 * 1024},
		{"1000000000", 1000000000},
	}
	for _, tt := range tests {
		got := postgres.ParseMemoryBytes(tt.input)
		if got != tt.want {
			t.Errorf("ParseMemoryBytes(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseMemoryBytes_Invalid(t *testing.T) {
	if postgres.ParseMemoryBytes("invalid") != -1 {
		t.Error("expected -1 for invalid quantity")
	}
}
