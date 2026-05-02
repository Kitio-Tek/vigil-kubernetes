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

package postgres_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/postgres"
)

func TestDefaultHBAConfTLSEnabled(t *testing.T) {
	rules := postgres.DefaultHBAConf(true)
	if len(rules) == 0 {
		t.Fatal("expected non-empty default HBA rules")
	}
	hasSSL := false
	for _, r := range rules {
		if strings.Contains(r, "hostssl") {
			hasSSL = true
			break
		}
	}
	if !hasSSL {
		t.Error("TLS-enabled HBA conf should contain hostssl rules")
	}
}

func TestDefaultHBAConfTLSDisabled(t *testing.T) {
	rules := postgres.DefaultHBAConf(false)
	for _, r := range rules {
		if strings.Contains(r, "hostssl") {
			t.Errorf("TLS-disabled HBA conf should not contain hostssl rules, got: %q", r)
		}
	}
}

func TestBuildHBAConf(t *testing.T) {
	custom := []string{"host mydb myuser 10.0.0.0/8 scram-sha-256"}
	conf := postgres.BuildHBAConf(custom, false)

	if !strings.Contains(conf, "host mydb myuser") {
		t.Error("custom rule should appear in generated HBA conf")
	}
	if !strings.HasPrefix(conf, "#") {
		t.Error("HBA conf should start with a comment header")
	}
}

func TestBuildHBAConfNoCustomRules(t *testing.T) {
	conf := postgres.BuildHBAConf(nil, false)
	if conf == "" {
		t.Error("HBA conf should not be empty")
	}
	if !strings.Contains(conf, "local") {
		t.Error("default local rule missing from HBA conf")
	}
}

func TestBuildHBAConfAlwaysHasPeer(t *testing.T) {
	for _, tls := range []bool{true, false} {
		conf := postgres.BuildHBAConf(nil, tls)
		if !strings.Contains(conf, "peer") {
			t.Errorf("HBA conf (tls=%v) should always contain local peer authentication", tls)
		}
	}
}
