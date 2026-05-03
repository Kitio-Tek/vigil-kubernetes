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

// Package conditions provides helpers for working with metav1.Condition
// arrays. These helpers complement those in the upstream
// apimachinery/pkg/api/meta package by adding convenience constructors and
// transition-aware setters that the operator's reconcilers use to mutate
// status conditions.
package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Set inserts or updates the given condition in conds. The LastTransitionTime
// is preserved if the existing condition's Status matches the new Status, and
// updated to now otherwise. ObservedGeneration is always replaced.
func Set(conds []metav1.Condition, c metav1.Condition) []metav1.Condition {
	if c.LastTransitionTime.IsZero() {
		c.LastTransitionTime = metav1.Now()
	}
	for i, existing := range conds {
		if existing.Type != c.Type {
			continue
		}
		if existing.Status == c.Status {
			c.LastTransitionTime = existing.LastTransitionTime
		}
		conds[i] = c
		return conds
	}
	return append(conds, c)
}

// Remove deletes the condition with the given type. The returned slice may
// share its backing array with the input slice.
func Remove(conds []metav1.Condition, condType string) []metav1.Condition {
	out := conds[:0]
	for _, c := range conds {
		if c.Type != condType {
			out = append(out, c)
		}
	}
	return out
}

// Find returns the condition with the given type, or nil if absent.
func Find(conds []metav1.Condition, condType string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == condType {
			return &conds[i]
		}
	}
	return nil
}

// IsTrue reports whether the named condition exists and is True.
func IsTrue(conds []metav1.Condition, condType string) bool {
	c := Find(conds, condType)
	return c != nil && c.Status == metav1.ConditionTrue
}

// IsFalse reports whether the named condition exists and is False.
func IsFalse(conds []metav1.Condition, condType string) bool {
	c := Find(conds, condType)
	return c != nil && c.Status == metav1.ConditionFalse
}

// IsUnknown reports whether the named condition exists and is Unknown,
// or is absent (which is treated as Unknown for callers that want a
// three-state view).
func IsUnknown(conds []metav1.Condition, condType string) bool {
	c := Find(conds, condType)
	return c == nil || c.Status == metav1.ConditionUnknown
}

// True returns a "Status=True" Condition with the given type, reason, and
// message. ObservedGeneration is left zero so the caller can populate it
// alongside the metadata.generation it observed.
func True(condType, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    condType,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

// False returns a "Status=False" Condition.
func False(condType, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    condType,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

// Unknown returns a "Status=Unknown" Condition.
func Unknown(condType, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    condType,
		Status:  metav1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	}
}

// WithObservedGeneration returns a copy of c with ObservedGeneration set to gen.
func WithObservedGeneration(c metav1.Condition, gen int64) metav1.Condition {
	c.ObservedGeneration = gen
	return c
}

// AnyFalse reports whether at least one of the named condition types is in
// the False state. It is useful as a single-line "is the resource degraded"
// check at the end of a reconcile loop.
func AnyFalse(conds []metav1.Condition, types ...string) bool {
	for _, t := range types {
		if IsFalse(conds, t) {
			return true
		}
	}
	return false
}

// AllTrue reports whether every named condition is True.
func AllTrue(conds []metav1.Condition, types ...string) bool {
	for _, t := range types {
		if !IsTrue(conds, t) {
			return false
		}
	}
	return true
}

// FilterByPrefix returns the conditions whose Type starts with prefix. Useful
// for collapsing a sub-resource's individual conditions before bubbling them
// into a parent resource's status.
func FilterByPrefix(conds []metav1.Condition, prefix string) []metav1.Condition {
	if prefix == "" {
		return append([]metav1.Condition(nil), conds...)
	}
	out := []metav1.Condition{}
	for _, c := range conds {
		if len(c.Type) >= len(prefix) && c.Type[:len(prefix)] == prefix {
			out = append(out, c)
		}
	}
	return out
}
