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

package cronexpr_test

import (
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/cronexpr"
)

func TestExpand_Aliases(t *testing.T) {
	cases := map[string]string{
		"@daily":   "0 0 * * *",
		"@hourly":  "0 * * * *",
		"@weekly":  "0 0 * * 0",
		"@yearly":  "0 0 1 1 *",
		"@monthly": "0 0 1 * *",
	}
	for in, want := range cases {
		got, err := cronexpr.Expand(in)
		if err != nil {
			t.Errorf("Expand(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("Expand(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpand_Empty(t *testing.T) {
	if _, err := cronexpr.Expand(""); err == nil {
		t.Error("expected error for empty input")
	}
}

func TestValidate_Star(t *testing.T) {
	if err := cronexpr.Validate("* * * * *"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_Numeric(t *testing.T) {
	if err := cronexpr.Validate("0 12 * * 1"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_Range(t *testing.T) {
	if err := cronexpr.Validate("0 9-17 * * 1-5"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_Step(t *testing.T) {
	if err := cronexpr.Validate("*/5 * * * *"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_List(t *testing.T) {
	if err := cronexpr.Validate("0,15,30,45 * * * *"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_Alias(t *testing.T) {
	if err := cronexpr.Validate("@daily"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_TooFewFields(t *testing.T) {
	if err := cronexpr.Validate("0 0 0"); err == nil {
		t.Error("expected error")
	}
}

func TestValidate_OutOfRange(t *testing.T) {
	if err := cronexpr.Validate("60 0 * * *"); err == nil {
		t.Error("minute=60 should be invalid")
	}
	if err := cronexpr.Validate("0 24 * * *"); err == nil {
		t.Error("hour=24 should be invalid")
	}
	if err := cronexpr.Validate("0 0 0 * *"); err == nil {
		t.Error("day=0 should be invalid")
	}
	if err := cronexpr.Validate("0 0 * 13 *"); err == nil {
		t.Error("month=13 should be invalid")
	}
}

func TestValidate_BadStep(t *testing.T) {
	if err := cronexpr.Validate("*/abc * * * *"); err == nil {
		t.Error("expected error for non-numeric step")
	}
	if err := cronexpr.Validate("/5 * * * *"); err == nil {
		t.Error("expected error for missing left side")
	}
}

func TestValidate_BadRange(t *testing.T) {
	if err := cronexpr.Validate("0 5-3 * * *"); err == nil {
		t.Error("expected error for inverted range")
	}
}

func TestIsAlias(t *testing.T) {
	if !cronexpr.IsAlias("@daily") {
		t.Error("@daily should be alias")
	}
	if cronexpr.IsAlias("0 0 * * *") {
		t.Error("explicit cron is not alias")
	}
	if cronexpr.IsAlias("@nope") {
		t.Error("unknown @ is not alias")
	}
}
