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

package pgbouncer_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/pgbouncer"
)

func TestDefaultConfig(t *testing.T) {
	cfg := pgbouncer.DefaultConfig()
	if cfg.ListenPort != pgbouncer.DefaultListenPort {
		t.Errorf("expected port %d, got %d", pgbouncer.DefaultListenPort, cfg.ListenPort)
	}
	if cfg.PoolMode != pgbouncer.PoolModeTransaction {
		t.Errorf("expected pool mode %q, got %q", pgbouncer.PoolModeTransaction, cfg.PoolMode)
	}
	if cfg.MaxClientConn != pgbouncer.DefaultMaxClientConn {
		t.Errorf("expected max_client_conn %d, got %d", pgbouncer.DefaultMaxClientConn, cfg.MaxClientConn)
	}
}

func TestINI_ContainsRequiredSections(t *testing.T) {
	cfg := pgbouncer.DefaultConfig()
	ini := cfg.INI("my-cluster", "my-cluster-primary", "mydb")

	required := []string{
		"[databases]",
		"[pgbouncer]",
		"listen_port",
		"pool_mode",
		"max_client_conn",
		"auth_type = scram-sha-256",
		"auth_file",
		"my-cluster-primary",
		"mydb",
	}
	for _, r := range required {
		if !strings.Contains(ini, r) {
			t.Errorf("INI output missing %q", r)
		}
	}
}

func TestINI_QueryTimeout_Zero(t *testing.T) {
	cfg := pgbouncer.DefaultConfig()
	cfg.QueryTimeout = 0
	ini := cfg.INI("my-cluster", "host", "db")
	if strings.Contains(ini, "query_timeout") {
		t.Error("query_timeout should not appear when value is 0")
	}
}

func TestINI_QueryTimeout_Nonzero(t *testing.T) {
	cfg := pgbouncer.DefaultConfig()
	cfg.QueryTimeout = 30
	ini := cfg.INI("my-cluster", "host", "db")
	if !strings.Contains(ini, "query_timeout = 30") {
		t.Error("query_timeout should appear when non-zero")
	}
}

func TestINI_PoolModes(t *testing.T) {
	modes := []pgbouncer.PoolMode{
		pgbouncer.PoolModeSession,
		pgbouncer.PoolModeTransaction,
		pgbouncer.PoolModeStatement,
	}
	for _, mode := range modes {
		cfg := pgbouncer.DefaultConfig()
		cfg.PoolMode = mode
		ini := cfg.INI("cluster", "host", "db")
		if !strings.Contains(ini, string(mode)) {
			t.Errorf("INI should contain pool mode %q", mode)
		}
	}
}

func TestUserList(t *testing.T) {
	users := map[string]string{
		"postgres": "SCRAM-SHA-256$4096:abc=:def=",
		"app":      "SCRAM-SHA-256$4096:xyz=:uvw=",
	}
	ul := pgbouncer.UserList(users)
	if !strings.Contains(ul, "postgres") {
		t.Error("userlist should contain postgres user")
	}
	if !strings.Contains(ul, "app") {
		t.Error("userlist should contain app user")
	}
}

func TestNamingFunctions(t *testing.T) {
	clusterName := "my-cluster"

	svcName := pgbouncer.ServiceName(clusterName)
	if !strings.Contains(svcName, clusterName) {
		t.Errorf("ServiceName should contain cluster name, got %q", svcName)
	}

	depName := pgbouncer.DeploymentName(clusterName)
	if !strings.Contains(depName, clusterName) {
		t.Errorf("DeploymentName should contain cluster name, got %q", depName)
	}

	cmName := pgbouncer.ConfigMapName(clusterName)
	if !strings.Contains(cmName, clusterName) {
		t.Errorf("ConfigMapName should contain cluster name, got %q", cmName)
	}

	secretName := pgbouncer.SecretName(clusterName)
	if !strings.Contains(secretName, clusterName) {
		t.Errorf("SecretName should contain cluster name, got %q", secretName)
	}
}

func TestListenAddr(t *testing.T) {
	cfg := pgbouncer.DefaultConfig()
	addr := pgbouncer.ListenAddr(cfg)
	if !strings.Contains(addr, "5432") {
		t.Errorf("listen addr should contain port, got %q", addr)
	}
}
