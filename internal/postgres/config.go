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

package postgres

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultParams returns a conservative set of postgresql.conf parameters suitable
// for a production cluster. Callers may override any of these values through the
// PostgresClusterSpec.PostgresParameters field.
func DefaultParams() map[string]string {
	return map[string]string{
		// Connection settings
		"listen_addresses": "'*'",
		"max_connections":  "100",
		"port":             "5432",

		// Memory
		"shared_buffers":       "128MB",
		"effective_cache_size": "512MB",
		"maintenance_work_mem": "64MB",
		"work_mem":             "4MB",

		// Write-ahead log
		"wal_level":             "replica",
		"max_wal_senders":       "10",
		"max_replication_slots": "10",
		"wal_keep_size":         "1GB",
		"archive_mode":          "off",

		// Checkpoints
		"checkpoint_completion_target": "0.9",
		"checkpoint_timeout":           "10min",

		// Query planner
		"random_page_cost":         "1.1",
		"effective_io_concurrency": "200",

		// Logging
		"log_destination":            "'stderr'",
		"logging_collector":          "off",
		"log_min_duration_statement": "1000",
		"log_connections":            "on",
		"log_disconnections":         "on",
		"log_line_prefix":            "'%m [%p] %q%u@%d '",

		// Autovacuum
		"autovacuum":                      "on",
		"autovacuum_max_workers":          "3",
		"autovacuum_naptime":              "1min",
		"autovacuum_vacuum_scale_factor":  "0.1",
		"autovacuum_analyze_scale_factor": "0.05",
	}
}

// MergeParams produces a merged parameter map. Values in override take precedence
// over values in base.
func MergeParams(base, override map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// BuildPostgresConf serialises a parameter map into a postgresql.conf-compatible
// string. Parameters are written in alphabetical order for deterministic output.
func BuildPostgresConf(params map[string]string) string {
	var sb strings.Builder
	sb.WriteString("# Managed by Vigil. Do not edit manually.\n")

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s = %s\n", k, params[k]))
	}
	return sb.String()
}
