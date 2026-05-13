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

// Package portmap centralises the named ports used across PostgreSQL,
// PgBouncer and the operator manager. Centralising the values prevents
// drift between the Service builder, the StatefulSet builder, the helm
// chart values and the README examples.
package portmap

import (
	"fmt"
	"sort"
)

// Named ports for the cluster. Names are reused across container ports,
// service ports and probe targets so they stay in sync.
const (
	PostgresPortName       = "postgres"
	PostgresPort     int32 = 5432

	PgBouncerPortName       = "pgbouncer"
	PgBouncerPort     int32 = 6432

	MetricsPortName       = "metrics"
	MetricsPort     int32 = 9187

	ManagerMetricsPortName       = "manager-metrics"
	ManagerMetricsPort     int32 = 8443

	ManagerHealthPortName       = "health"
	ManagerHealthPort     int32 = 8081
)

// Port pairs a name and a numeric port together for use in ports lists.
type Port struct {
	Name string
	Port int32
}

// All returns the standard cluster ports in stable order.
func All() []Port {
	return []Port{
		{Name: PostgresPortName, Port: PostgresPort},
		{Name: PgBouncerPortName, Port: PgBouncerPort},
		{Name: MetricsPortName, Port: MetricsPort},
		{Name: ManagerMetricsPortName, Port: ManagerMetricsPort},
		{Name: ManagerHealthPortName, Port: ManagerHealthPort},
	}
}

// PostgresOnly returns the postgres-only port list, used for services that
// should not expose pgbouncer or metrics.
func PostgresOnly() []Port {
	return []Port{{Name: PostgresPortName, Port: PostgresPort}}
}

// Find returns the Port with the given name or false.
func Find(name string) (Port, bool) {
	for _, p := range All() {
		if p.Name == name {
			return p, true
		}
	}
	return Port{}, false
}

// IsWellKnown reports whether port matches one of the registered named ports.
func IsWellKnown(port int32) bool {
	for _, p := range All() {
		if p.Port == port {
			return true
		}
	}
	return false
}

// SortedNames returns the registered port names alphabetically. Useful in
// tests to assert stable iteration order regardless of map traversal.
func SortedNames() []string {
	all := All()
	names := make([]string, 0, len(all))
	for _, p := range all {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}

// MustFind returns the Port with the given name or panics. Used in tests.
func MustFind(name string) Port {
	p, ok := Find(name)
	if !ok {
		panic(fmt.Sprintf("portmap: unknown port name %q", name))
	}
	return p
}

// SafeInt32 converts a TCP port expressed as int into int32. Ports are
// always within [0, 65535] so the narrowing is mathematically safe, but
// the explicit clamp satisfies static analysers that flag bare int->int32
// conversions (gosec G115). Out-of-range inputs collapse to zero so the
// caller fails closed on misconfiguration rather than silently wrapping.
func SafeInt32(port int) int32 {
	if port < 0 || port > 65535 {
		return 0
	}
	return int32(port) //#nosec G115 -- bounded by guard above
}
