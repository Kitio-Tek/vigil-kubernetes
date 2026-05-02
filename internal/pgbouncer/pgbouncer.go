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

// Package pgbouncer provides helpers for deploying and configuring a PgBouncer
// connection pooler alongside a PostgresCluster. PgBouncer is deployed as a
// sidecar or separate Deployment depending on the cluster topology.
package pgbouncer

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// DefaultListenPort is the port PgBouncer listens on.
	DefaultListenPort = 5432

	// AdminListenPort is the port for PgBouncer admin console.
	AdminListenPort = 6432

	// DefaultPoolMode is the default connection pooling mode.
	DefaultPoolMode = "transaction"

	// DefaultMaxClientConn is the maximum number of client connections.
	DefaultMaxClientConn = 1000

	// DefaultDefaultPoolSize is the default pool size per database/user pair.
	DefaultDefaultPoolSize = 20

	// DefaultMinPoolSize is the minimum number of connections kept alive.
	DefaultMinPoolSize = 0

	// DefaultReservePoolSize is extra connections that can be used on demand.
	DefaultReservePoolSize = 5

	// DefaultReservePoolTimeout is seconds to wait before using reserve connections.
	DefaultReservePoolTimeout = 5

	// DefaultServerIdleTimeout is seconds before idle server connections are closed.
	DefaultServerIdleTimeout = 600

	// DefaultServerLifetime is maximum seconds a server connection is kept.
	DefaultServerLifetime = 3600

	// DefaultQueryTimeout is seconds before an unfinished query is cancelled.
	DefaultQueryTimeout = 0

	// DefaultClientIdleTimeout is seconds before idle client connections are closed.
	DefaultClientIdleTimeout = 0

	// DefaultLogConnections controls whether connections are logged.
	DefaultLogConnections = 0

	// DefaultLogDisconnections controls whether disconnections are logged.
	DefaultLogDisconnections = 0

	// DefaultLogPoolerErrors controls whether pool errors are logged.
	DefaultLogPoolerErrors = 1

	// Image is the official PgBouncer image reference.
	Image = "pgbouncer/pgbouncer:1.22.1"
)

// PoolMode represents a PgBouncer pooling mode.
type PoolMode string

const (
	// PoolModeSession means a server connection is assigned to the client for
	// the duration of the client session.
	PoolModeSession PoolMode = "session"

	// PoolModeTransaction means a server connection is assigned per transaction.
	PoolModeTransaction PoolMode = "transaction"

	// PoolModeStatement means the server connection is released after each
	// statement. Prepared statements are not supported in this mode.
	PoolModeStatement PoolMode = "statement"
)

// Config holds the PgBouncer configuration for a cluster.
type Config struct {
	// ListenPort is the port PgBouncer listens on.
	ListenPort int
	// PoolMode is the connection pooling mode.
	PoolMode PoolMode
	// MaxClientConn is the maximum number of client connections.
	MaxClientConn int
	// DefaultPoolSize is the default pool size per database/user pair.
	DefaultPoolSize int
	// MinPoolSize is the minimum connections to keep alive.
	MinPoolSize int
	// ReservePoolSize is the extra connections available on demand.
	ReservePoolSize int
	// ReservePoolTimeout is seconds before reserve pool is used.
	ReservePoolTimeout int
	// ServerIdleTimeout is the idle server connection timeout in seconds.
	ServerIdleTimeout int
	// ServerLifetime is the maximum server connection lifetime in seconds.
	ServerLifetime int
	// QueryTimeout is the query timeout in seconds (0 disables it).
	QueryTimeout int
	// ClientIdleTimeout is the idle client timeout in seconds (0 disables it).
	ClientIdleTimeout int
}

// DefaultConfig returns a Config with production-oriented defaults.
func DefaultConfig() Config {
	return Config{
		ListenPort:         DefaultListenPort,
		PoolMode:           PoolModeTransaction,
		MaxClientConn:      DefaultMaxClientConn,
		DefaultPoolSize:    DefaultDefaultPoolSize,
		MinPoolSize:        DefaultMinPoolSize,
		ReservePoolSize:    DefaultReservePoolSize,
		ReservePoolTimeout: DefaultReservePoolTimeout,
		ServerIdleTimeout:  DefaultServerIdleTimeout,
		ServerLifetime:     DefaultServerLifetime,
		QueryTimeout:       DefaultQueryTimeout,
		ClientIdleTimeout:  DefaultClientIdleTimeout,
	}
}

// INI renders the pgbouncer.ini configuration file content for the given
// cluster. The primaryHost is the hostname of the primary service.
func (c Config) INI(clusterName, primaryHost, database string) string {
	var b strings.Builder

	b.WriteString("[databases]\n")
	b.WriteString(fmt.Sprintf("%s = host=%s port=5432 dbname=%s\n", database, primaryHost, database))
	b.WriteString("* = host=" + primaryHost + " port=5432\n\n")

	b.WriteString("[pgbouncer]\n")
	b.WriteString(fmt.Sprintf("listen_port = %d\n", c.ListenPort))
	b.WriteString("listen_addr = *\n")
	b.WriteString(fmt.Sprintf("pool_mode = %s\n", c.PoolMode))
	b.WriteString(fmt.Sprintf("max_client_conn = %d\n", c.MaxClientConn))
	b.WriteString(fmt.Sprintf("default_pool_size = %d\n", c.DefaultPoolSize))
	b.WriteString(fmt.Sprintf("min_pool_size = %d\n", c.MinPoolSize))
	b.WriteString(fmt.Sprintf("reserve_pool_size = %d\n", c.ReservePoolSize))
	b.WriteString(fmt.Sprintf("reserve_pool_timeout = %d\n", c.ReservePoolTimeout))
	b.WriteString(fmt.Sprintf("server_idle_timeout = %d\n", c.ServerIdleTimeout))
	b.WriteString(fmt.Sprintf("server_lifetime = %d\n", c.ServerLifetime))
	if c.QueryTimeout > 0 {
		b.WriteString(fmt.Sprintf("query_timeout = %d\n", c.QueryTimeout))
	}
	if c.ClientIdleTimeout > 0 {
		b.WriteString(fmt.Sprintf("client_idle_timeout = %d\n", c.ClientIdleTimeout))
	}
	b.WriteString("auth_type = scram-sha-256\n")
	b.WriteString("auth_file = /etc/pgbouncer/userlist.txt\n")
	b.WriteString("admin_users = pgbouncer_admin\n")
	b.WriteString(fmt.Sprintf("stats_users = %s_stats\n", clusterName))
	b.WriteString("server_tls_sslmode = prefer\n")
	b.WriteString(fmt.Sprintf("log_connections = %d\n", DefaultLogConnections))
	b.WriteString(fmt.Sprintf("log_disconnections = %d\n", DefaultLogDisconnections))
	b.WriteString(fmt.Sprintf("log_pooler_errors = %d\n", DefaultLogPoolerErrors))
	b.WriteString("ignore_startup_parameters = extra_float_digits\n")

	return b.String()
}

// UserList renders the contents of the userlist.txt file from a map of
// username to scram-sha-256 password hash.
func UserList(users map[string]string) string {
	var b strings.Builder
	for user, hash := range users {
		b.WriteString(fmt.Sprintf("%q %q\n", user, hash))
	}
	return b.String()
}

// ServiceName returns the name of the Kubernetes Service that fronts PgBouncer.
func ServiceName(clusterName string) string {
	return clusterName + "-pooler"
}

// DeploymentName returns the name of the PgBouncer Deployment.
func DeploymentName(clusterName string) string {
	return clusterName + "-pooler"
}

// ConfigMapName returns the name of the ConfigMap holding pgbouncer.ini.
func ConfigMapName(clusterName string) string {
	return clusterName + "-pgbouncer-config"
}

// SecretName returns the name of the Secret holding userlist.txt.
func SecretName(clusterName string) string {
	return clusterName + "-pgbouncer-users"
}

// ListenAddr returns a formatted "host:port" string for the given config.
func ListenAddr(c Config) string {
	return ":" + strconv.Itoa(c.ListenPort)
}
