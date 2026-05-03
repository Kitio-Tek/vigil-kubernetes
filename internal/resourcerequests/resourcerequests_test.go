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

package resourcerequests_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/Kitio-Tek/athos-kubernetes/internal/resourcerequests"
)

func TestDefault_PopulatesAllFields(t *testing.T) {
	r := resourcerequests.Default()
	if _, ok := r.Requests[corev1.ResourceCPU]; !ok {
		t.Error("missing CPU request")
	}
	if _, ok := r.Limits[corev1.ResourceMemory]; !ok {
		t.Error("missing memory limit")
	}
}

func TestMerge_OverlayOverrides(t *testing.T) {
	base := resourcerequests.Default()
	overlay := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
	}
	merged := resourcerequests.Merge(base, overlay)
	if got := merged.Requests[corev1.ResourceCPU]; got.Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("cpu request = %s", got.String())
	}
}

func TestMerge_ZeroOverlayIgnored(t *testing.T) {
	base := resourcerequests.Default()
	overlay := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.Quantity{}},
	}
	merged := resourcerequests.Merge(base, overlay)
	if got := merged.Requests[corev1.ResourceCPU]; got.Cmp(resourcerequests.DefaultCPURequest) != 0 {
		t.Errorf("zero overlay should not override base, got %s", got.String())
	}
}

func TestMerge_NilBaseRequests(t *testing.T) {
	overlay := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("2Gi")},
	}
	merged := resourcerequests.Merge(corev1.ResourceRequirements{}, overlay)
	got := merged.Limits[corev1.ResourceMemory]
	want := resource.MustParse("2Gi")
	if got.Cmp(want) != 0 {
		t.Errorf("expected 2Gi memory limit, got %s", got.String())
	}
}

func TestAsMaxConnections_Small(t *testing.T) {
	got := resourcerequests.AsMaxConnections(resource.MustParse("128Mi"))
	if got < 25 {
		t.Errorf("expected at least 25, got %d", got)
	}
}

func TestAsMaxConnections_Large(t *testing.T) {
	got := resourcerequests.AsMaxConnections(resource.MustParse("4Gi"))
	if got < 100 {
		t.Errorf("expected >100 connections for 4Gi, got %d", got)
	}
}

func TestAsMaxConnections_NoMemory(t *testing.T) {
	got := resourcerequests.AsMaxConnections(resource.Quantity{})
	if got != 25 {
		t.Errorf("expected minimum 25 for empty quantity, got %d", got)
	}
}
