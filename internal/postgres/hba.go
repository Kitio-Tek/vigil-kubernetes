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
	"strings"
)

// DefaultHBAConf returns the baseline pg_hba.conf rules for a cluster.
// When tlsEnabled is true, remote connections are required to use TLS.
func DefaultHBAConf(tlsEnabled bool) []string {
	rules := []string{
		// Local socket connections always use peer authentication.
		"local   all             all                                     peer",
		// Loopback connections use password authentication.
		"host    all             all             127.0.0.1/32            scram-sha-256",
		"host    all             all             ::1/128                 scram-sha-256",
		// Replication connections from any pod in the cluster subnet.
		"host    replication     all             10.0.0.0/8              scram-sha-256",
		"host    replication     all             172.16.0.0/12           scram-sha-256",
		"host    replication     all             192.168.0.0/16          scram-sha-256",
	}

	if tlsEnabled {
		// Require TLS for all remote application connections.
		rules = append(rules,
			"hostssl all             all             0.0.0.0/0               scram-sha-256",
			"hostssl all             all             ::/0                    scram-sha-256",
		)
	} else {
		// Allow plain TCP connections when TLS is disabled.
		rules = append(rules,
			"host    all             all             0.0.0.0/0               scram-sha-256",
			"host    all             all             ::/0                    scram-sha-256",
		)
	}

	return rules
}

// BuildHBAConf serialises a slice of HBA rules into a pg_hba.conf string.
// The provided rules are appended after the standard header comment.
func BuildHBAConf(rules []string, tlsEnabled bool) string {
	var sb strings.Builder
	sb.WriteString("# Managed by Vigil. Do not edit manually.\n")
	sb.WriteString("# TYPE  DATABASE        USER            ADDRESS                 METHOD\n")

	base := DefaultHBAConf(tlsEnabled)
	for _, r := range base {
		sb.WriteString(r)
		sb.WriteString("\n")
	}

	if len(rules) > 0 {
		sb.WriteString("\n# Custom rules from PostgresCluster.spec.postgresHBA\n")
		for _, r := range rules {
			sb.WriteString(r)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
