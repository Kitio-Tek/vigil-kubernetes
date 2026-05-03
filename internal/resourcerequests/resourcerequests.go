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

// Package resourcerequests centralises the CPU and memory defaults applied
// to every PostgreSQL container the operator manages. Callers either pass
// in user-supplied requests or accept the defaults.
package resourcerequests

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Default CPU and memory shapes for the postgres container. The values are
// chosen to fit comfortably on a 1-vCPU / 1-GiB worker so a fresh kind
// cluster can host a small cluster without manual tuning.
var (
	DefaultCPURequest    = resource.MustParse("250m")
	DefaultMemoryRequest = resource.MustParse("256Mi")
	DefaultCPULimit      = resource.MustParse("1")
	DefaultMemoryLimit   = resource.MustParse("1Gi")
)

// Default returns a corev1.ResourceRequirements populated with the
// package defaults.
func Default() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    DefaultCPURequest,
			corev1.ResourceMemory: DefaultMemoryRequest,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    DefaultCPULimit,
			corev1.ResourceMemory: DefaultMemoryLimit,
		},
	}
}

// Merge returns a copy of base with overlay values overriding it.
// A zero quantity in overlay is treated as "no override".
func Merge(base, overlay corev1.ResourceRequirements) corev1.ResourceRequirements {
	out := base.DeepCopy()
	if out.Requests == nil {
		out.Requests = corev1.ResourceList{}
	}
	if out.Limits == nil {
		out.Limits = corev1.ResourceList{}
	}
	for k, v := range overlay.Requests {
		if !v.IsZero() {
			out.Requests[k] = v
		}
	}
	for k, v := range overlay.Limits {
		if !v.IsZero() {
			out.Limits[k] = v
		}
	}
	return *out
}

// AsMaxConnections estimates a conservative max_connections setting from
// a memory limit, using ~10 MiB per connection plus 100 MiB of headroom
// for shared buffers. The minimum returned value is 25.
func AsMaxConnections(memoryLimit resource.Quantity) int {
	const minimum = 25
	const perConnMiB = 10
	const headroomMiB = 100
	mi := memoryLimit.Value() / (1024 * 1024)
	avail := mi - headroomMiB
	if avail <= 0 {
		return minimum
	}
	conn := int(avail / perConnMiB)
	if conn < minimum {
		return minimum
	}
	return conn
}
