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

// Package poolerconfig renders the pgbouncer.ini and userlist.txt files
// that the PostgresPooler controller writes into the pooler ConfigMap.
//
// The renderer is parametrised on a small Spec struct rather than the full
// PostgresPooler API type so the package can be unit tested without
// pulling in the controller-runtime test harness.
package poolerconfig

import (
	"fmt"
	"sort"
	"strings"
)

// PoolMode mirrors the upstream pgbouncer pool_mode values.
type PoolMode string

// Recognised pool modes.
const (
	PoolModeSession     PoolMode = "session"
	PoolModeTransaction PoolMode = "transaction"
	PoolModeStatement   PoolMode = "statement"
)

// AuthType captures the authentication mechanism used by clients
// connecting to PgBouncer.
type AuthType string

// Recognised auth types.
const (
	AuthMD5      AuthType = "md5"
	AuthScramSHA AuthType = "scram-sha-256"
	AuthPlain    AuthType = "plain"
	AuthTrust    AuthType = "trust"
)

// Spec captures the user-facing configuration for a pooler.
type Spec struct {
	// ListenPort is the TCP port pgbouncer listens on. Defaults to 6432.
	ListenPort int
	// PoolMode is one of the recognised PoolMode values. Defaults to
	// transaction.
	PoolMode PoolMode
	// MaxClientConn is the maximum total number of client connections.
	MaxClientConn int
	// DefaultPoolSize is the per-database default pool size.
	DefaultPoolSize int
	// AuthType is the authentication mechanism. Defaults to scram-sha-256.
	AuthType AuthType
	// AuthUser is the role pgbouncer uses to look up auth queries.
	AuthUser string
	// AuthQuery is the SQL pgbouncer runs to authenticate users.
	AuthQuery string
	// Databases maps a logical database name to a connection URI.
	Databases map[string]string
	// IgnoreStartupParameters is a comma-separated list of GUCs that
	// pgbouncer should pass through unchanged.
	IgnoreStartupParameters string
}

// Render returns the contents of pgbouncer.ini for the given spec.
func Render(s Spec) string {
	if s.ListenPort == 0 {
		s.ListenPort = 6432
	}
	if s.PoolMode == "" {
		s.PoolMode = PoolModeTransaction
	}
	if s.AuthType == "" {
		s.AuthType = AuthScramSHA
	}
	if s.MaxClientConn == 0 {
		s.MaxClientConn = 100
	}
	if s.DefaultPoolSize == 0 {
		s.DefaultPoolSize = 20
	}
	if s.IgnoreStartupParameters == "" {
		s.IgnoreStartupParameters = "extra_float_digits"
	}

	var b strings.Builder
	b.WriteString("[databases]\n")
	for _, name := range sortedKeys(s.Databases) {
		fmt.Fprintf(&b, "%s = %s\n", name, s.Databases[name])
	}

	b.WriteString("\n[pgbouncer]\n")
	fmt.Fprintf(&b, "listen_addr = 0.0.0.0\n")
	fmt.Fprintf(&b, "listen_port = %d\n", s.ListenPort)
	fmt.Fprintf(&b, "auth_type = %s\n", s.AuthType)
	if s.AuthUser != "" {
		fmt.Fprintf(&b, "auth_user = %s\n", s.AuthUser)
	}
	if s.AuthQuery != "" {
		fmt.Fprintf(&b, "auth_query = %s\n", s.AuthQuery)
	}
	fmt.Fprintf(&b, "auth_file = /etc/pgbouncer/userlist.txt\n")
	fmt.Fprintf(&b, "pool_mode = %s\n", s.PoolMode)
	fmt.Fprintf(&b, "max_client_conn = %d\n", s.MaxClientConn)
	fmt.Fprintf(&b, "default_pool_size = %d\n", s.DefaultPoolSize)
	fmt.Fprintf(&b, "ignore_startup_parameters = %s\n", s.IgnoreStartupParameters)
	return b.String()
}

// RenderUserList renders the userlist.txt file used by pgbouncer's
// auth_file. Each entry is a quoted username and quoted password hash.
func RenderUserList(users map[string]string) string {
	var b strings.Builder
	for _, u := range sortedKeys(users) {
		fmt.Fprintf(&b, "\"%s\" \"%s\"\n", u, users[u])
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IsValidPoolMode reports whether p is a recognised PoolMode value.
func IsValidPoolMode(p PoolMode) bool {
	switch p {
	case PoolModeSession, PoolModeTransaction, PoolModeStatement:
		return true
	}
	return false
}
