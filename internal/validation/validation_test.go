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

package validation_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/validation"
)

func TestIntInRange(t *testing.T) {
	if e := validation.IntInRange("x", 5, 1, 10); e != nil {
		t.Errorf("expected nil, got %v", e)
	}
	if e := validation.IntInRange("x", 0, 1, 10); e == nil {
		t.Error("expected error, got nil")
	}
	if e := validation.IntInRange("x", 11, 1, 10); e == nil {
		t.Error("expected error, got nil")
	}
}

func TestOneOf(t *testing.T) {
	if e := validation.OneOf("x", "yes", "yes", "no"); e != nil {
		t.Errorf("expected nil, got %v", e)
	}
	if e := validation.OneOf("x", "maybe", "yes", "no"); e == nil {
		t.Error("expected error, got nil")
	}
}

func TestNonEmpty(t *testing.T) {
	if e := validation.NonEmpty("x", ""); e == nil {
		t.Error("expected error for empty string")
	}
	if e := validation.NonEmpty("x", "v"); e != nil {
		t.Errorf("unexpected: %v", e)
	}
}

func TestMaxLength(t *testing.T) {
	if e := validation.MaxLength("x", "abcd", 3); e == nil {
		t.Error("expected error")
	}
	if e := validation.MaxLength("x", "ab", 3); e != nil {
		t.Errorf("unexpected: %v", e)
	}
}

func TestMinLength(t *testing.T) {
	if e := validation.MinLength("x", "ab", 3); e == nil {
		t.Error("expected error")
	}
	if e := validation.MinLength("x", "abcd", 3); e != nil {
		t.Errorf("unexpected: %v", e)
	}
}

func TestIsDNSLabel(t *testing.T) {
	cases := map[string]bool{
		"foo":          true,
		"foo-bar":      true,
		"my-pg-1":      true,
		"":             false,
		"-foo":         false,
		"foo-":         false,
		"FOO":          false,
		"foo bar":      false,
		strings.Repeat("a", 64): false,
		strings.Repeat("a", 63): true,
	}
	for in, want := range cases {
		if got := validation.IsDNSLabel(in); got != want {
			t.Errorf("IsDNSLabel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestErrors_AddAndErr(t *testing.T) {
	var errs validation.Errors
	if errs.Err() != nil {
		t.Error("empty Errors should return nil err")
	}
	errs.Add("a", "bad", 1)
	errs.Add("b", "worse", 2)
	if e := errs.Err(); e == nil {
		t.Fatal("expected non-nil err")
	} else if !strings.Contains(e.Error(), "a:") || !strings.Contains(e.Error(), "b:") {
		t.Errorf("err = %q", e.Error())
	}
}

func TestErrors_Has(t *testing.T) {
	var errs validation.Errors
	errs.Add("spec.instances", "bad", 0)
	if !errs.Has("spec.instances") {
		t.Error("Has should find existing field")
	}
	if errs.Has("spec.unknown") {
		t.Error("Has should not match unknown field")
	}
}

func TestAppend_DropsNil(t *testing.T) {
	out := validation.Append(nil,
		validation.IntInRange("x", 5, 1, 10),
		validation.NonEmpty("y", ""),
	)
	if len(out) != 1 || out[0].Field != "y" {
		t.Errorf("unexpected aggregate: %+v", out)
	}
}

func TestDNSLabel_Wrapper(t *testing.T) {
	if e := validation.DNSLabel("x", "good-label"); e != nil {
		t.Errorf("unexpected: %v", e)
	}
	if e := validation.DNSLabel("x", "BadLabel"); e == nil {
		t.Error("expected error for invalid DNS label")
	}
}
