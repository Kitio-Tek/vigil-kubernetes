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

package userstate_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/userstate"
)

func TestPlan_BasicCreate(t *testing.T) {
	stmts, err := userstate.Plan(userstate.User{Name: "alice"}, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !strings.HasPrefix(stmts[0], "CREATE USER") {
		t.Errorf("first stmt = %q", stmts[0])
	}
	if !strings.Contains(strings.Join(stmts, " "), "NOSUPERUSER") {
		t.Error("expected NOSUPERUSER to be enforced for non-superuser")
	}
}

func TestPlan_BasicAlter(t *testing.T) {
	stmts, err := userstate.Plan(userstate.User{Name: "bob"}, true)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !strings.HasPrefix(stmts[0], "ALTER USER") {
		t.Errorf("first stmt = %q", stmts[0])
	}
}

func TestPlan_PasswordEmitted(t *testing.T) {
	stmts, _ := userstate.Plan(userstate.User{Name: "alice", Password: "secret"}, true)
	found := false
	for _, s := range stmts {
		if strings.Contains(s, "PASSWORD 'secret'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("password ALTER missing: %v", stmts)
	}
}

func TestPlan_Superuser(t *testing.T) {
	stmts, _ := userstate.Plan(userstate.User{Name: "alice", Superuser: true}, true)
	if !strings.Contains(strings.Join(stmts, " "), "SUPERUSER") {
		t.Error("expected SUPERUSER")
	}
}

func TestPlan_ConnectionLimit(t *testing.T) {
	stmts, _ := userstate.Plan(userstate.User{Name: "alice", ConnectionLimit: 5}, true)
	if !strings.Contains(strings.Join(stmts, " "), "CONNECTION LIMIT 5") {
		t.Error("expected connection limit")
	}
}

func TestPlan_RolesGranted(t *testing.T) {
	stmts, _ := userstate.Plan(userstate.User{
		Name:  "alice",
		Roles: []string{"reader", "admin"},
	}, true)
	all := strings.Join(stmts, " ")
	if !strings.Contains(all, `GRANT "admin" TO "alice"`) {
		t.Error("admin role grant missing")
	}
	if !strings.Contains(all, `GRANT "reader" TO "alice"`) {
		t.Error("reader role grant missing")
	}
}

func TestPlan_DatabaseGrants(t *testing.T) {
	stmts, _ := userstate.Plan(userstate.User{
		Name: "alice",
		GrantsByDatabase: map[string][]string{
			"db1": {"SELECT", "INSERT"},
		},
	}, true)
	all := strings.Join(stmts, " ")
	if !strings.Contains(all, `GRANT INSERT ON DATABASE "db1" TO "alice"`) {
		t.Error("INSERT grant missing")
	}
}

func TestPlan_RequiresName(t *testing.T) {
	if _, err := userstate.Plan(userstate.User{}, false); err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestPlanRevoke(t *testing.T) {
	stmts := userstate.PlanRevoke(userstate.User{
		Name: "alice",
		GrantsByDatabase: map[string][]string{
			"db1": {"SELECT"},
		},
	})
	all := strings.Join(stmts, " ")
	for _, want := range []string{"REVOKE ALL ON DATABASE", "REASSIGN OWNED BY", "DROP OWNED BY", `DROP USER IF EXISTS "alice"`} {
		if !strings.Contains(all, want) {
			t.Errorf("revoke plan missing %q: %v", want, stmts)
		}
	}
}

func TestPlan_DeterministicOrder(t *testing.T) {
	u := userstate.User{
		Name:  "alice",
		Roles: []string{"c", "a", "b"},
	}
	a, _ := userstate.Plan(u, true)
	b, _ := userstate.Plan(u, true)
	if strings.Join(a, "|") != strings.Join(b, "|") {
		t.Error("Plan output is not deterministic")
	}
}
