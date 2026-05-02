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

// Package replication provides helpers for configuring PostgreSQL streaming
// replication within a Vigil-managed cluster. It generates recovery.conf
// (PostgreSQL < 12) and postgresql.auto.conf fragments (PostgreSQL >= 12) as
// well as standby.signal file content.
package replication

import (
	"fmt"
	"strings"
)

const (
	// DefaultSyncStandbyNames is the synchronous_standby_names value used when
	// synchronous replication is requested for all replicas.
	DefaultSyncStandbyNames = "*"

	// DefaultWalLevel is the minimum wal_level required for streaming replication.
	DefaultWalLevel = "replica"

	// DefaultMaxWalSenders is the default max_wal_senders setting.
	DefaultMaxWalSenders = 10

	// DefaultMaxReplicationSlots is the default max_replication_slots setting.
	DefaultMaxReplicationSlots = 10

	// DefaultWalKeepSize is the default wal_keep_size in megabytes (PG >= 13).
	DefaultWalKeepSize = 128

	// PrimaryConnInfoFormat is the connection string template used in
	// primary_conninfo for a replica connecting to the primary.
	PrimaryConnInfoFormat = "host=%s port=5432 user=%s password=%s application_name=%s"

	// ReplicationUser is the PostgreSQL role used for replication connections.
	ReplicationUser = "replicator"
)

// WalConfig holds the PostgreSQL WAL and replication-related configuration
// parameters that must be set on the primary instance.
type WalConfig struct {
	// WalLevel sets the wal_level parameter. Defaults to DefaultWalLevel.
	WalLevel string
	// MaxWalSenders is the max_wal_senders value.
	MaxWalSenders int
	// MaxReplicationSlots is the max_replication_slots value.
	MaxReplicationSlots int
	// WalKeepSize is the wal_keep_size in MB (ignored if PostgresVersion < 13).
	WalKeepSize int
	// SynchronousStandbyNames sets synchronous_standby_names. Empty means async.
	SynchronousStandbyNames string
	// PostgresVersion is the major version of PostgreSQL.
	PostgresVersion int32
}

// DefaultWalConfig returns a WalConfig with sensible defaults for the given
// PostgreSQL major version.
func DefaultWalConfig(pgVersion int32) WalConfig {
	return WalConfig{
		WalLevel:            DefaultWalLevel,
		MaxWalSenders:       DefaultMaxWalSenders,
		MaxReplicationSlots: DefaultMaxReplicationSlots,
		WalKeepSize:         DefaultWalKeepSize,
		PostgresVersion:     pgVersion,
	}
}

// Parameters returns a map of postgresql.conf parameters for the WAL config.
// These parameters are merged into the ConfigMap by the controller.
func (w WalConfig) Parameters() map[string]string {
	params := map[string]string{
		"wal_level":              w.WalLevel,
		"max_wal_senders":        fmt.Sprintf("%d", w.MaxWalSenders),
		"max_replication_slots":  fmt.Sprintf("%d", w.MaxReplicationSlots),
		"hot_standby":            "on",
		"hot_standby_feedback":   "on",
	}
	if w.PostgresVersion >= 13 {
		params["wal_keep_size"] = fmt.Sprintf("%dMB", w.WalKeepSize)
	} else {
		// In PG < 13, wal_keep_segments is used. 128 MB / 16 MB per segment = 8.
		params["wal_keep_segments"] = fmt.Sprintf("%d", w.WalKeepSize/16)
	}
	if w.SynchronousStandbyNames != "" {
		params["synchronous_standby_names"] = w.SynchronousStandbyNames
		params["synchronous_commit"] = "on"
	}
	return params
}

// StandbyConfig holds the configuration for a replica connecting to the primary.
type StandbyConfig struct {
	// PrimaryHost is the hostname or IP of the primary.
	PrimaryHost string
	// ReplicationPassword is the password for the replication user.
	ReplicationPassword string
	// ApplicationName is the application_name sent in the connection string.
	ApplicationName string
	// PostgresVersion is the major version of PostgreSQL.
	PostgresVersion int32
}

// PrimaryConnInfo returns the primary_conninfo connection string.
func (s StandbyConfig) PrimaryConnInfo() string {
	return fmt.Sprintf(PrimaryConnInfoFormat,
		s.PrimaryHost,
		ReplicationUser,
		s.ReplicationPassword,
		s.ApplicationName,
	)
}

// RecoveryConfig returns the content to write into postgresql.auto.conf for a
// standby instance. For PostgreSQL >= 12 this is written alongside a
// standby.signal file; for older versions it is written to recovery.conf.
func (s StandbyConfig) RecoveryConfig() string {
	var b strings.Builder
	b.WriteString("# Managed by vigil-kubernetes - do not edit\n")
	b.WriteString(fmt.Sprintf("primary_conninfo = '%s'\n", s.PrimaryConnInfo()))
	b.WriteString("restore_command = ''\n")
	b.WriteString("recovery_target_timeline = 'latest'\n")
	if s.PostgresVersion < 12 {
		b.WriteString("standby_mode = 'on'\n")
	}
	return b.String()
}

// HBAReplicationEntry returns a pg_hba.conf line that allows the replication
// user to connect from within the cluster network using password auth.
func HBAReplicationEntry(cidr string) string {
	return fmt.Sprintf("host replication %s %s scram-sha-256", ReplicationUser, cidr)
}

// ReplicationSlotName returns the name of the physical replication slot for a
// given standby pod ordinal.
func ReplicationSlotName(clusterName string, ordinal int) string {
	name := strings.ReplaceAll(clusterName, "-", "_")
	return fmt.Sprintf("%s_slot_%d", name, ordinal)
}
