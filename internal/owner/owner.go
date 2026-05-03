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

// Package owner provides helpers for inspecting the metav1.OwnerReference
// fields the operator places on every sub-resource. The helpers are pure so
// they can be exercised in tests without spinning up a fake client.
package owner

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Find returns the first OwnerReference matching the given (apiVersion, kind)
// pair, or nil if there is no match.
func Find(refs []metav1.OwnerReference, apiVersion, kind string) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].APIVersion == apiVersion && refs[i].Kind == kind {
			return &refs[i]
		}
	}
	return nil
}

// FindByKind returns the first owner whose Kind matches, ignoring APIVersion.
func FindByKind(refs []metav1.OwnerReference, kind string) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Kind == kind {
			return &refs[i]
		}
	}
	return nil
}

// IsControlledBy reports whether refs contains a controller=true reference
// to the given (apiVersion, kind, name) tuple.
func IsControlledBy(refs []metav1.OwnerReference, apiVersion, kind, name string) bool {
	for _, r := range refs {
		if r.APIVersion == apiVersion && r.Kind == kind && r.Name == name {
			if r.Controller != nil && *r.Controller {
				return true
			}
		}
	}
	return false
}

// IsManagedByAthos reports whether at least one OwnerReference points at one
// of the Athos resource Kinds. Useful for guarding accidental modification
// of foreign-owned resources.
func IsManagedByAthos(refs []metav1.OwnerReference) bool {
	managed := map[string]bool{
		"PostgresCluster": true,
		"PostgresBackup":  true,
		"PostgresUser":    true,
		"PostgresPooler":  true,
	}
	for _, r := range refs {
		if managed[r.Kind] {
			return true
		}
	}
	return false
}

// Names returns the names of the owners, in the order they appear in refs.
func Names(refs []metav1.OwnerReference) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Name)
	}
	return out
}

// Controller returns the OwnerReference whose Controller is true, or nil.
func Controller(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}
