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

package store_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/store"
)

func TestDefaultConnParams_Fields(t *testing.T) {
	p := store.DefaultConnParams("pg-primary.default.svc")
	if p.Port != 5432 {
		t.Errorf("Port = %d, want 5432", p.Port)
	}
	if p.User != "postgres" {
		t.Errorf("User = %q, want postgres", p.User)
	}
	if p.Database != "postgres" {
		t.Errorf("Database = %q, want postgres", p.Database)
	}
	if p.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want require", p.SSLMode)
	}
}

func TestDSN_ContainsRequiredKeys(t *testing.T) {
	p := store.DefaultConnParams("myhost")
	dsn := p.DSN()
	for _, key := range []string{"host=", "port=", "user=", "dbname=", "sslmode="} {
		if !strings.Contains(dsn, key) {
			t.Errorf("DSN missing %q: %s", key, dsn)
		}
	}
}

func TestDSN_NoPasswordWhenEmpty(t *testing.T) {
	p := store.DefaultConnParams("myhost")
	dsn := p.DSN()
	if strings.Contains(dsn, "password=") {
		t.Errorf("DSN should not contain password when empty: %s", dsn)
	}
}

func TestDSN_IncludesPassword(t *testing.T) {
	p := store.DefaultConnParams("myhost")
	p.Password = "secret"
	dsn := p.DSN()
	if !strings.Contains(dsn, "password=") {
		t.Errorf("DSN should contain password: %s", dsn)
	}
}

func TestURI_Format(t *testing.T) {
	p := store.DefaultConnParams("pg-rw.production.svc")
	uri := p.URI()
	if !strings.HasPrefix(uri, "postgresql://") {
		t.Errorf("URI should start with postgresql://, got: %s", uri)
	}
	if !strings.Contains(uri, "pg-rw.production.svc") {
		t.Errorf("URI should contain hostname: %s", uri)
	}
	if !strings.Contains(uri, "sslmode=require") {
		t.Errorf("URI should contain sslmode: %s", uri)
	}
}

func TestURI_WithPassword(t *testing.T) {
	p := store.DefaultConnParams("myhost")
	p.Password = "s3cr3t"
	uri := p.URI()
	if !strings.Contains(uri, "s3cr3t@") {
		t.Errorf("URI should contain password in userinfo: %s", uri)
	}
}

func TestWithDatabase(t *testing.T) {
	p := store.DefaultConnParams("h").WithDatabase("mydb")
	if p.Database != "mydb" {
		t.Errorf("Database = %q, want mydb", p.Database)
	}
}

func TestWithUser(t *testing.T) {
	p := store.DefaultConnParams("h").WithUser("appuser", "pass")
	if p.User != "appuser" {
		t.Errorf("User = %q, want appuser", p.User)
	}
	if p.Password != "pass" {
		t.Errorf("Password not set")
	}
}

func TestWithSSLMode(t *testing.T) {
	p := store.DefaultConnParams("h").WithSSLMode("disable")
	if p.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want disable", p.SSLMode)
	}
}

func TestWithExtra(t *testing.T) {
	p := store.DefaultConnParams("h").WithExtra("connect_timeout", "10")
	dsn := p.DSN()
	if !strings.Contains(dsn, "connect_timeout=10") {
		t.Errorf("DSN missing extra param: %s", dsn)
	}
}

func TestWithExtra_DoesNotMutateOriginal(t *testing.T) {
	orig := store.DefaultConnParams("h")
	_ = orig.WithExtra("k", "v")
	if orig.Extra != nil && orig.Extra["k"] == "v" {
		t.Error("WithExtra should not mutate the original ConnParams")
	}
}

func TestReplicationDSN(t *testing.T) {
	dsn := store.ReplicationDSN("host", 5432, "repuser", "reppass")
	if !strings.Contains(dsn, "dbname=replication") {
		t.Errorf("replication DSN should have dbname=replication: %s", dsn)
	}
	if !strings.Contains(dsn, "user=repuser") {
		t.Errorf("replication DSN should have user: %s", dsn)
	}
}

func TestSuperuserDSN(t *testing.T) {
	dsn := store.SuperuserDSN("pg", 5432, "pw")
	if !strings.Contains(dsn, "user=postgres") {
		t.Errorf("superuser DSN should have user=postgres: %s", dsn)
	}
	if !strings.Contains(dsn, "password=") {
		t.Errorf("superuser DSN should contain password: %s", dsn)
	}
}

func TestParseDSN_RoundTrip(t *testing.T) {
	orig := store.DefaultConnParams("pghost").WithUser("alice", "wonderland").WithDatabase("myapp")
	dsn := orig.DSN()

	parsed, err := store.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN error: %v", err)
	}
	if parsed.Host != "pghost" {
		t.Errorf("Host = %q, want pghost", parsed.Host)
	}
	if parsed.User != "alice" {
		t.Errorf("User = %q, want alice", parsed.User)
	}
	if parsed.Database != "myapp" {
		t.Errorf("Database = %q, want myapp", parsed.Database)
	}
	if parsed.Password != "wonderland" {
		t.Errorf("Password not round-tripped correctly")
	}
}

func TestParseDSN_MalformedKey(t *testing.T) {
	_, err := store.ParseDSN("host myhost")
	if err == nil {
		t.Error("expected error for DSN without '='")
	}
}

func TestParseDSN_ExtraParams(t *testing.T) {
	dsn := "host=h port=5432 connect_timeout=5 user=u dbname=db sslmode=require"
	p, err := store.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN error: %v", err)
	}
	if p.Extra["connect_timeout"] != "5" {
		t.Errorf("Extra[connect_timeout] = %q, want 5", p.Extra["connect_timeout"])
	}
}
