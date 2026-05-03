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

package sqlescape_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/sqlescape"
)

func TestIdentifier_Simple(t *testing.T) {
	if got := sqlescape.Identifier("users"); got != `"users"` {
		t.Errorf("Identifier(users) = %q", got)
	}
}

func TestIdentifier_DoublesEmbeddedQuotes(t *testing.T) {
	got := sqlescape.Identifier(`weird"name`)
	want := `"weird""name"`
	if got != want {
		t.Errorf("Identifier = %q, want %q", got, want)
	}
}

func TestIdentifier_RejectsNul(t *testing.T) {
	got := sqlescape.Identifier("ab\x00cd")
	if !strings.Contains(got, `\x00`) {
		t.Errorf("expected NUL handling, got %q", got)
	}
}

func TestStringLiteral(t *testing.T) {
	if got := sqlescape.StringLiteral("it's"); got != "'it''s'" {
		t.Errorf("StringLiteral = %q", got)
	}
}

func TestStringLiteral_NoQuotes(t *testing.T) {
	if got := sqlescape.StringLiteral("plain"); got != "'plain'" {
		t.Errorf("StringLiteral = %q", got)
	}
}

func TestQualifiedIdentifier(t *testing.T) {
	got := sqlescape.QualifiedIdentifier("public", "users")
	want := `"public"."users"`
	if got != want {
		t.Errorf("QualifiedIdentifier = %q, want %q", got, want)
	}
}

func TestQualifiedIdentifier_Empty(t *testing.T) {
	if got := sqlescape.QualifiedIdentifier(); got != "" {
		t.Errorf("QualifiedIdentifier(empty) = %q", got)
	}
}

func TestIsValidIdentifier(t *testing.T) {
	cases := map[string]bool{
		"users":                true,
		"_users":               true,
		"User1":                true,
		"weird$name":           true,
		"123start":             false,
		"":                     false,
		"with space":           false,
		"good-name":            false,
		strings.Repeat("a", 63): true,
		strings.Repeat("a", 64): false,
	}
	for in, want := range cases {
		if got := sqlescape.IsValidIdentifier(in); got != want {
			t.Errorf("IsValidIdentifier(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestMustIdentifier_Valid(t *testing.T) {
	got, err := sqlescape.MustIdentifier("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `"users"` {
		t.Errorf("got %q", got)
	}
}

func TestMustIdentifier_Invalid(t *testing.T) {
	if _, err := sqlescape.MustIdentifier("123bad"); err == nil {
		t.Error("expected error")
	}
}

func TestAssertSafePassword_OK(t *testing.T) {
	if err := sqlescape.AssertSafePassword("Sup3r$ecure"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAssertSafePassword_NUL(t *testing.T) {
	if err := sqlescape.AssertSafePassword("a\x00b"); err == nil {
		t.Error("expected error for NUL")
	}
}
