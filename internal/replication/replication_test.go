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

package replication_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/replication"
)

func TestDefaultWalConfig(t *testing.T) {
	cfg := replication.DefaultWalConfig(16)
	if cfg.WalLevel != replication.DefaultWalLevel {
		t.Errorf("expected wal_level %q, got %q", replication.DefaultWalLevel, cfg.WalLevel)
	}
	if cfg.MaxWalSenders != replication.DefaultMaxWalSenders {
		t.Errorf("expected max_wal_senders %d, got %d", replication.DefaultMaxWalSenders, cfg.MaxWalSenders)
	}
}

func TestWalConfigParameters_PG16(t *testing.T) {
	cfg := replication.DefaultWalConfig(16)
	params := cfg.Parameters()

	if params["wal_level"] != "replica" {
		t.Errorf("expected wal_level=replica, got %q", params["wal_level"])
	}
	if _, ok := params["wal_keep_size"]; !ok {
		t.Error("expected wal_keep_size to be set for PG >= 13")
	}
	if _, ok := params["wal_keep_segments"]; ok {
		t.Error("wal_keep_segments should not be set for PG >= 13")
	}
	if params["hot_standby"] != "on" {
		t.Error("expected hot_standby=on")
	}
}

func TestWalConfigParameters_PG11(t *testing.T) {
	cfg := replication.DefaultWalConfig(11)
	params := cfg.Parameters()

	if _, ok := params["wal_keep_segments"]; !ok {
		t.Error("expected wal_keep_segments to be set for PG < 13")
	}
	if _, ok := params["wal_keep_size"]; ok {
		t.Error("wal_keep_size should not be set for PG < 13")
	}
}

func TestWalConfigSynchronousReplication(t *testing.T) {
	cfg := replication.DefaultWalConfig(16)
	cfg.SynchronousStandbyNames = "*"
	params := cfg.Parameters()

	if params["synchronous_standby_names"] != "*" {
		t.Errorf("expected synchronous_standby_names=*, got %q", params["synchronous_standby_names"])
	}
	if params["synchronous_commit"] != "on" {
		t.Error("expected synchronous_commit=on when sync replication is enabled")
	}
}

func TestWalConfigAsynchronousReplication(t *testing.T) {
	cfg := replication.DefaultWalConfig(16)
	params := cfg.Parameters()

	if _, ok := params["synchronous_standby_names"]; ok {
		t.Error("synchronous_standby_names should not be set for async replication")
	}
}

func TestStandbyConfig_PrimaryConnInfo(t *testing.T) {
	cfg := replication.StandbyConfig{
		PrimaryHost:         "my-cluster-primary",
		ReplicationPassword: "s3cr3t",
		ApplicationName:     "my-cluster-1",
		PostgresVersion:     16,
	}
	connInfo := cfg.PrimaryConnInfo()

	if !strings.Contains(connInfo, "my-cluster-primary") {
		t.Error("primary_conninfo should contain host")
	}
	if !strings.Contains(connInfo, replication.ReplicationUser) {
		t.Error("primary_conninfo should contain replication user")
	}
	if !strings.Contains(connInfo, "s3cr3t") {
		t.Error("primary_conninfo should contain password")
	}
	if !strings.Contains(connInfo, "my-cluster-1") {
		t.Error("primary_conninfo should contain application_name")
	}
}

func TestStandbyConfig_RecoveryConfig_PG16(t *testing.T) {
	cfg := replication.StandbyConfig{
		PrimaryHost:         "host",
		ReplicationPassword: "pass",
		ApplicationName:     "app",
		PostgresVersion:     16,
	}
	recovery := cfg.RecoveryConfig()

	if !strings.Contains(recovery, "primary_conninfo") {
		t.Error("recovery config should contain primary_conninfo")
	}
	if strings.Contains(recovery, "standby_mode") {
		t.Error("standby_mode should not appear in PG >= 12 recovery config")
	}
	if !strings.Contains(recovery, "recovery_target_timeline") {
		t.Error("recovery config should contain recovery_target_timeline")
	}
}

func TestStandbyConfig_RecoveryConfig_PG11(t *testing.T) {
	cfg := replication.StandbyConfig{
		PrimaryHost:         "host",
		ReplicationPassword: "pass",
		ApplicationName:     "app",
		PostgresVersion:     11,
	}
	recovery := cfg.RecoveryConfig()

	if !strings.Contains(recovery, "standby_mode") {
		t.Error("standby_mode should appear in PG < 12 recovery.conf")
	}
}

func TestHBAReplicationEntry(t *testing.T) {
	entry := replication.HBAReplicationEntry("10.0.0.0/8")
	if !strings.Contains(entry, "replication") {
		t.Error("HBA entry should reference replication")
	}
	if !strings.Contains(entry, "10.0.0.0/8") {
		t.Error("HBA entry should contain CIDR")
	}
	if !strings.Contains(entry, replication.ReplicationUser) {
		t.Error("HBA entry should contain replication user")
	}
}

func TestReplicationSlotName(t *testing.T) {
	name := replication.ReplicationSlotName("my-cluster", 2)
	if strings.Contains(name, "-") {
		t.Error("slot name should not contain hyphens (invalid in PostgreSQL)")
	}
	if !strings.Contains(name, "my_cluster") {
		t.Errorf("slot name should contain sanitized cluster name, got %q", name)
	}
	if !strings.Contains(name, "2") {
		t.Errorf("slot name should contain ordinal, got %q", name)
	}
}
