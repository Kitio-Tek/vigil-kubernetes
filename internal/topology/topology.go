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

// Package topology helps reason about pod placement across nodes, zones and
// regions. It produces affinity, anti-affinity and topology-spread constraint
// shapes used by the StatefulSet builder.
package topology

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Standard topology key constants matching the well-known node labels.
const (
	KeyHostname = "kubernetes.io/hostname"
	KeyZone     = "topology.kubernetes.io/zone"
	KeyRegion   = "topology.kubernetes.io/region"
)

// Spread represents a desired anti-affinity policy across pods of the same
// PostgresCluster.
type Spread string

const (
	// SpreadNone disables anti-affinity.
	SpreadNone Spread = "None"
	// SpreadHost forces pods onto distinct nodes.
	SpreadHost Spread = "Host"
	// SpreadZone forces pods onto distinct availability zones.
	SpreadZone Spread = "Zone"
	// SpreadPreferHost prefers but does not require distinct nodes.
	SpreadPreferHost Spread = "PreferHost"
	// SpreadPreferZone prefers but does not require distinct zones.
	SpreadPreferZone Spread = "PreferZone"
)

// PodAntiAffinity returns a PodAntiAffinity matching the given spread, or nil
// if spread is SpreadNone.
func PodAntiAffinity(selector map[string]string, spread Spread) *corev1.PodAntiAffinity {
	switch spread {
	case SpreadHost:
		return &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				term(selector, KeyHostname),
			},
		}
	case SpreadZone:
		return &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				term(selector, KeyZone),
			},
		}
	case SpreadPreferHost:
		return &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{Weight: 100, PodAffinityTerm: term(selector, KeyHostname)},
			},
		}
	case SpreadPreferZone:
		return &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{Weight: 100, PodAffinityTerm: term(selector, KeyZone)},
			},
		}
	}
	return nil
}

func term(selector map[string]string, topologyKey string) corev1.PodAffinityTerm {
	return corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{MatchLabels: selector},
		TopologyKey:   topologyKey,
	}
}

// Spreads returns a slice of TopologySpreadConstraint objects for the given
// selector. The first constraint enforces a single skew across hosts; the
// second is a preferred constraint across zones.
func Spreads(selector map[string]string) []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       KeyHostname,
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: selector},
		},
		{
			MaxSkew:           1,
			TopologyKey:       KeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: selector},
		},
	}
}

// IsRequired reports whether the spread mode requires (rather than prefers)
// the constraint, which influences how the manager treats unschedulable pods.
func IsRequired(spread Spread) bool {
	switch spread {
	case SpreadHost, SpreadZone:
		return true
	}
	return false
}

// SpreadFromString parses a string into a Spread. Empty input maps to
// SpreadPreferHost which is the safe default for clusters across mixed nodes.
func SpreadFromString(s string) Spread {
	switch s {
	case "":
		return SpreadPreferHost
	case "None", "none":
		return SpreadNone
	case "Host", "host":
		return SpreadHost
	case "Zone", "zone":
		return SpreadZone
	case "PreferHost", "preferHost", "preferhost":
		return SpreadPreferHost
	case "PreferZone", "preferZone", "preferzone":
		return SpreadPreferZone
	default:
		return SpreadPreferHost
	}
}
