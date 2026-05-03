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

// Package finalizers provides string-list helpers for working with the
// .metadata.finalizers field. Controller-runtime's controllerutil
// equivalents work on typed objects; this package keeps the operations
// pure so they can be unit-tested without a fake client.
package finalizers

// Add inserts the given finalizer if it is not already present. It returns
// the new slice and a bool indicating whether a change was made.
func Add(list []string, f string) ([]string, bool) {
	for _, x := range list {
		if x == f {
			return list, false
		}
	}
	out := make([]string, 0, len(list)+1)
	out = append(out, list...)
	return append(out, f), true
}

// Remove deletes every occurrence of f from the list. It returns the new
// slice and a bool indicating whether a change was made.
func Remove(list []string, f string) ([]string, bool) {
	out := make([]string, 0, len(list))
	changed := false
	for _, x := range list {
		if x == f {
			changed = true
			continue
		}
		out = append(out, x)
	}
	return out, changed
}

// Contains reports whether f appears in the list.
func Contains(list []string, f string) bool {
	for _, x := range list {
		if x == f {
			return true
		}
	}
	return false
}

// Standard finalizer constants used by Athos resources.
const (
	// PostgresClusterFinalizer is added to PostgresCluster objects so the
	// reconciler can perform clean-up before Kubernetes deletes the CR.
	PostgresClusterFinalizer = "pg.athos.io/postgrescluster"
	// PostgresBackupFinalizer is added to PostgresBackup objects so the
	// reconciler can persist final status before letting the CR be removed.
	PostgresBackupFinalizer = "pg.athos.io/postgresbackup"
	// PostgresUserFinalizer is added to PostgresUser objects so the
	// reconciler can revoke privileges before the CR disappears.
	PostgresUserFinalizer = "pg.athos.io/postgresuser"
	// PostgresPoolerFinalizer is added to PostgresPooler objects so the
	// reconciler can drain in-flight connections before tear-down.
	PostgresPoolerFinalizer = "pg.athos.io/postgrespooler"
)

// AthosFinalizer reports whether the given finalizer string belongs to one
// of the Athos resource kinds.
func AthosFinalizer(f string) bool {
	switch f {
	case PostgresClusterFinalizer,
		PostgresBackupFinalizer,
		PostgresUserFinalizer,
		PostgresPoolerFinalizer:
		return true
	}
	return false
}
