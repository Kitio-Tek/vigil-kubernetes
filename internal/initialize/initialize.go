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

// Package initialize generates the initdb configuration and bootstrap scripts
// that are run when a new PostgresCluster is first created. The init container
// script set up data directory permissions, runs initdb, and writes the
// initial pg_hba.conf and postgresql.conf.
package initialize

import (
	"fmt"
	"strings"
)

const (
	// ScriptConfigMapKey is the key under which the bootstrap script is stored
	// in the cluster ConfigMap.
	ScriptConfigMapKey = "bootstrap.sh"

	// PGDataDir is the PostgreSQL data directory inside the container.
	PGDataDir = "/var/lib/postgresql/data/pgdata"

	// PGWALDir is the write-ahead log directory when a separate WAL volume is
	// configured.
	PGWALDir = "/var/lib/postgresql/wal/pg_wal"

	// PGConfigDir is the directory where operator-managed config files are
	// mounted.
	PGConfigDir = "/etc/postgresql"

	// InitScriptTimeout is the timeout in seconds for the bootstrap script.
	InitScriptTimeout = 120
)

// InitParams holds the parameters required to generate the bootstrap script.
type InitParams struct {
	// PostgresVersion is the major version of PostgreSQL being initialised.
	PostgresVersion int32
	// DatabaseName is the name of the initial application database.
	DatabaseName string
	// SuperuserName is the PostgreSQL superuser name.
	SuperuserName string
	// ReplicationUser is the replication role name.
	ReplicationUser string
	// Locale is the cluster locale (default: en_US.UTF-8).
	Locale string
	// Encoding is the server encoding (default: UTF8).
	Encoding string
	// DataChecksums enables data page checksums via initdb --data-checksums.
	DataChecksums bool
	// WALDir when set moves pg_wal to a separate volume.
	WALDir string
}

// DefaultInitParams returns InitParams with sensible defaults for the given
// PostgreSQL version and cluster name.
func DefaultInitParams(pgVersion int32, clusterName string) InitParams {
	return InitParams{
		PostgresVersion: pgVersion,
		DatabaseName:    clusterName,
		SuperuserName:   "postgres",
		ReplicationUser: "replicator",
		Locale:          "en_US.UTF-8",
		Encoding:        "UTF8",
		DataChecksums:   true,
	}
}

// BootstrapScript returns the content of the init container shell script.
// The script is idempotent: it only runs initdb if PGDATA is empty.
func (p InitParams) BootstrapScript() string {
	var b strings.Builder

	b.WriteString("#!/bin/bash\nset -euo pipefail\n\n")
	b.WriteString(fmt.Sprintf("PGDATA=%q\n\n", PGDataDir))

	b.WriteString("# Create data directory if it does not exist\n")
	b.WriteString("mkdir -p \"${PGDATA}\"\n")
	b.WriteString("chmod 0700 \"${PGDATA}\"\n\n")

	b.WriteString("# Only run initdb if PGDATA is not already initialised\n")
	b.WriteString("if [ ! -f \"${PGDATA}/PG_VERSION\" ]; then\n")
	b.WriteString(fmt.Sprintf("  initdb_args=\"--username=%s --encoding=%s --locale=%s\"\n",
		p.SuperuserName, p.Encoding, p.Locale))
	if p.DataChecksums {
		b.WriteString("  initdb_args=\"${initdb_args} --data-checksums\"\n")
	}
	if p.WALDir != "" {
		b.WriteString(fmt.Sprintf("  mkdir -p %q\n", p.WALDir))
		b.WriteString(fmt.Sprintf("  initdb_args=\"${initdb_args} --waldir=%s\"\n", p.WALDir))
	}
	b.WriteString("  initdb ${initdb_args} -D \"${PGDATA}\"\n")
	b.WriteString("fi\n\n")

	b.WriteString("# Copy operator-managed config files\n")
	b.WriteString(fmt.Sprintf("cp %s/postgresql.conf \"${PGDATA}/postgresql.conf\"\n", PGConfigDir))
	b.WriteString(fmt.Sprintf("cp %s/pg_hba.conf \"${PGDATA}/pg_hba.conf\"\n\n", PGConfigDir))

	b.WriteString("echo \"Bootstrap complete.\"\n")
	return b.String()
}

// PostInitSQL returns SQL statements to run after the first cluster start.
// These statements create the replication user and application database.
func (p InitParams) PostInitSQL(password string) string {
	var b strings.Builder

	b.WriteString("-- Managed by athos-kubernetes - do not edit\n\n")

	b.WriteString(fmt.Sprintf(
		"CREATE USER %s WITH REPLICATION ENCRYPTED PASSWORD %s;\n",
		quoteIdent(p.ReplicationUser),
		quoteLiteral(password),
	))

	if p.DatabaseName != "postgres" && p.DatabaseName != p.SuperuserName {
		b.WriteString(fmt.Sprintf(
			"CREATE DATABASE %s OWNER %s ENCODING %s LC_COLLATE %s LC_CTYPE %s;\n",
			quoteIdent(p.DatabaseName),
			quoteIdent(p.SuperuserName),
			quoteLiteral(p.Encoding),
			quoteLiteral(p.Locale),
			quoteLiteral(p.Locale),
		))
	}

	return b.String()
}

// quoteIdent returns a double-quoted PostgreSQL identifier.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// quoteLiteral returns a single-quoted PostgreSQL string literal.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// DataDirIsEmpty returns a shell expression that evaluates to true when PGDATA
// contains no PostgreSQL cluster files.
func DataDirIsEmpty() string {
	return `[ ! -f "${PGDATA}/PG_VERSION" ]`
}

// ContainerName returns the name of the init container that runs the bootstrap
// script.
func ContainerName() string {
	return "init-postgres"
}
