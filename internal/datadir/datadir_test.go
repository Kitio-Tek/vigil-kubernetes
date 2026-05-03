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

package datadir_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/datadir"
)

func TestPGDataPath_Empty(t *testing.T) {
	if got := datadir.PGDataPath(""); got != datadir.PGData {
		t.Errorf("PGDataPath(\"\") = %q", got)
	}
}

func TestPGDataPath_Joining(t *testing.T) {
	if got := datadir.PGDataPath("base/12345"); !strings.HasSuffix(got, "/data/base/12345") {
		t.Errorf("PGDataPath = %q", got)
	}
}

func TestPGDataPath_LeadingSlashTrimmed(t *testing.T) {
	if got := datadir.PGDataPath("/base"); got != datadir.PGData+"/base" {
		t.Errorf("leading slash not trimmed: %q", got)
	}
}

func TestWALPath(t *testing.T) {
	if got := datadir.WALPath(""); got != datadir.WAL {
		t.Errorf("WALPath empty = %q", got)
	}
	if got := datadir.WALPath("0001"); !strings.HasSuffix(got, "wal/0001") {
		t.Errorf("WALPath = %q", got)
	}
}

func TestConfigPath(t *testing.T) {
	if got := datadir.ConfigPath("postgresql.conf"); !strings.HasSuffix(got, "/postgresql.conf") {
		t.Errorf("ConfigPath = %q", got)
	}
}

func TestIsInsidePGData(t *testing.T) {
	cases := map[string]bool{
		datadir.PGData:                true,
		datadir.PGData + "/base":      true,
		datadir.PGData + "/../escape": false,
		"/etc/postgresql":             false,
		"/var/lib/postgresql/wal":     false,
	}
	for path, want := range cases {
		if got := datadir.IsInsidePGData(path); got != want {
			t.Errorf("IsInsidePGData(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestPostgresqlConf(t *testing.T) {
	if !strings.HasSuffix(datadir.PostgresqlConf(), "postgresql.conf") {
		t.Errorf("PostgresqlConf = %q", datadir.PostgresqlConf())
	}
}

func TestPGHBAConf(t *testing.T) {
	if !strings.HasSuffix(datadir.PGHBAConf(), "pg_hba.conf") {
		t.Errorf("PGHBAConf = %q", datadir.PGHBAConf())
	}
}

func TestPIDFile(t *testing.T) {
	if !strings.HasSuffix(datadir.PIDFile(), "postmaster.pid") {
		t.Errorf("PIDFile = %q", datadir.PIDFile())
	}
}

func TestIsValidPGDataLayout_Complete(t *testing.T) {
	present := map[string]bool{
		"PG_VERSION": true,
		"base":       true,
		"global":     true,
		"pg_wal":     true,
		"extra":      true,
	}
	if err := datadir.IsValidPGDataLayout(present); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestIsValidPGDataLayout_Missing(t *testing.T) {
	present := map[string]bool{
		"PG_VERSION": true,
		"base":       true,
	}
	err := datadir.IsValidPGDataLayout(present)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "global") && !strings.Contains(err.Error(), "pg_wal") {
		t.Errorf("expected error to name missing entry, got %q", err.Error())
	}
}
