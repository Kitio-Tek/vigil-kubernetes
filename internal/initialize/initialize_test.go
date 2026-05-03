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

package initialize_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/initialize"
)

func TestDefaultInitParams(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	if p.PostgresVersion != 16 {
		t.Errorf("expected version 16, got %d", p.PostgresVersion)
	}
	if p.SuperuserName != "postgres" {
		t.Errorf("expected superuser postgres, got %q", p.SuperuserName)
	}
	if p.ReplicationUser != "replicator" {
		t.Errorf("expected replication user replicator, got %q", p.ReplicationUser)
	}
	if !p.DataChecksums {
		t.Error("expected data checksums to be enabled by default")
	}
}

func TestBootstrapScript_Idempotent(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	script := p.BootstrapScript()

	if !strings.Contains(script, initialize.PGDataDir) {
		t.Error("script should reference PGDATA")
	}
	if !strings.Contains(script, "PG_VERSION") {
		t.Error("script should check for PG_VERSION to be idempotent")
	}
	if !strings.Contains(script, "initdb") {
		t.Error("script should invoke initdb")
	}
}

func TestBootstrapScript_DataChecksums(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	p.DataChecksums = true
	script := p.BootstrapScript()

	if !strings.Contains(script, "data-checksums") {
		t.Error("script should include --data-checksums flag")
	}
}

func TestBootstrapScript_NoDataChecksums(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	p.DataChecksums = false
	script := p.BootstrapScript()

	if strings.Contains(script, "data-checksums") {
		t.Error("script should not include --data-checksums when disabled")
	}
}

func TestBootstrapScript_WALDir(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	p.WALDir = initialize.PGWALDir
	script := p.BootstrapScript()

	if !strings.Contains(script, initialize.PGWALDir) {
		t.Error("script should include WAL directory")
	}
	if !strings.Contains(script, "waldir") {
		t.Error("script should pass --waldir to initdb")
	}
}

func TestBootstrapScript_CopiesConfig(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	script := p.BootstrapScript()

	if !strings.Contains(script, "postgresql.conf") {
		t.Error("script should copy postgresql.conf")
	}
	if !strings.Contains(script, "pg_hba.conf") {
		t.Error("script should copy pg_hba.conf")
	}
}

func TestBootstrapScript_Locale(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	p.Locale = "fr_FR.UTF-8"
	script := p.BootstrapScript()

	if !strings.Contains(script, "fr_FR.UTF-8") {
		t.Error("script should include specified locale")
	}
}

func TestPostInitSQL_CreatesReplicationUser(t *testing.T) {
	p := initialize.DefaultInitParams(16, "my-cluster")
	sql := p.PostInitSQL("s3cr3t")

	if !strings.Contains(sql, "REPLICATION") {
		t.Error("post-init SQL should create replication user")
	}
	if !strings.Contains(sql, p.ReplicationUser) {
		t.Errorf("post-init SQL should contain replication user %q", p.ReplicationUser)
	}
}

func TestPostInitSQL_CreatesDatabase(t *testing.T) {
	p := initialize.DefaultInitParams(16, "myapp")
	sql := p.PostInitSQL("pass")

	if !strings.Contains(sql, "CREATE DATABASE") {
		t.Error("post-init SQL should create application database")
	}
	if !strings.Contains(sql, "myapp") {
		t.Error("post-init SQL should reference application database name")
	}
}

func TestPostInitSQL_SkipsDatabaseForPostgres(t *testing.T) {
	p := initialize.DefaultInitParams(16, "postgres")
	sql := p.PostInitSQL("pass")

	if strings.Contains(sql, "CREATE DATABASE") {
		t.Error("should not create 'postgres' database as it already exists")
	}
}

func TestDataDirIsEmpty(t *testing.T) {
	expr := initialize.DataDirIsEmpty()
	if !strings.Contains(expr, "PG_VERSION") {
		t.Error("expression should check PG_VERSION file")
	}
}

func TestContainerName(t *testing.T) {
	name := initialize.ContainerName()
	if name == "" {
		t.Error("container name should not be empty")
	}
}
