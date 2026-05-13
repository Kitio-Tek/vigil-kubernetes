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

// Package cronexpr is a small utility for validating CRON expressions used
// in the PostgresCluster.spec.backup.schedule field. The package only
// validates and recognises common shorthand strings; computing the next fire
// time is delegated to k8s.io/apimachinery/pkg/util/wait or the operator
// scheduler package.
package cronexpr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Standard predefined schedules accepted by classic cron implementations.
var aliases = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// Expand returns the canonical 5-field cron expression for s, including
// expanding shorthand aliases like "@daily".
func Expand(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("cronexpr: empty expression")
	}
	if v, ok := aliases[s]; ok {
		return v, nil
	}
	return s, nil
}

// Validate parses and validates s. It returns nil if the expression is a
// recognised alias or has five whitespace-separated fields with each field
// matching one of the supported forms ("*", a number, "a-b", "*/n", or a
// comma-separated list of those).
func Validate(s string) error {
	expr, err := Expand(s)
	if err != nil {
		return err
	}
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cronexpr: expected 5 fields, got %d", len(fields))
	}
	ranges := []struct{ min, max int }{
		{0, 59}, // minute
		{0, 23}, // hour
		{1, 31}, // day-of-month
		{1, 12}, // month
		{0, 7},  // day-of-week (0 and 7 both Sunday)
	}
	for i, f := range fields {
		if err := validateField(f, ranges[i].min, ranges[i].max); err != nil {
			return fmt.Errorf("cronexpr: field %d: %w", i+1, err)
		}
	}
	return nil
}

func validateField(f string, min, max int) error {
	if f == "*" {
		return nil
	}
	for _, part := range strings.Split(f, ",") {
		if err := validateAtom(part, min, max); err != nil {
			return err
		}
	}
	return nil
}

func validateAtom(p string, min, max int) error {
	if p == "*" {
		return nil
	}
	if strings.Contains(p, "/") {
		return validateStepAtom(p, min, max)
	}
	if strings.Contains(p, "-") {
		return validateRangeAtom(p, min, max)
	}
	return validateSingleValue(p, min, max)
}

func validateStepAtom(p string, min, max int) error {
	i := strings.Index(p, "/")
	left := p[:i]
	right := p[i+1:]
	if left == "" || right == "" {
		return fmt.Errorf("invalid step %q", p)
	}
	if left != "*" {
		if err := validateAtom(left, min, max); err != nil {
			return err
		}
	}
	n, err := strconv.Atoi(right)
	if err != nil || n <= 0 {
		return fmt.Errorf("invalid step %q", p)
	}
	return nil
}

func validateRangeAtom(p string, min, max int) error {
	i := strings.Index(p, "-")
	left, right := p[:i], p[i+1:]
	a, err := strconv.Atoi(left)
	if err != nil {
		return fmt.Errorf("invalid range bound %q", left)
	}
	b, err := strconv.Atoi(right)
	if err != nil {
		return fmt.Errorf("invalid range bound %q", right)
	}
	if a < min || b > max || a > b {
		return fmt.Errorf("range %d-%d outside [%d,%d]", a, b, min, max)
	}
	return nil
}

func validateSingleValue(p string, min, max int) error {
	n, err := strconv.Atoi(p)
	if err != nil {
		return fmt.Errorf("invalid value %q", p)
	}
	if n < min || n > max {
		return fmt.Errorf("value %d outside [%d,%d]", n, min, max)
	}
	return nil
}

// IsAlias reports whether s is one of the recognised @-prefixed aliases.
func IsAlias(s string) bool {
	_, ok := aliases[strings.TrimSpace(s)]
	return ok
}
