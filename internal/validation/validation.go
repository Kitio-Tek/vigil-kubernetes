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

// Package validation centralises the field-level checks performed by the
// admission webhooks. The functions here are pure, take typed inputs, and
// return field-rooted errors that the webhook layer wraps into an
// admission.Response.
package validation

import (
	"errors"
	"fmt"
	"strings"
)

// FieldError describes a single invalid input field.
type FieldError struct {
	// Field is the JSON path of the offending field (e.g. "spec.instances").
	Field string
	// Detail is a human-readable explanation of why the value is invalid.
	Detail string
	// Bad is the offending value, captured for inclusion in error messages.
	Bad any
}

// Error implements the error interface.
func (f *FieldError) Error() string {
	return fmt.Sprintf("%s: %s (got %v)", f.Field, f.Detail, f.Bad)
}

// Errors aggregates several FieldErrors. The zero value is ready to use.
type Errors []*FieldError

// Add appends e to the list. The receiver pointer-method form lets callers
// build up the list across helper calls.
func (e *Errors) Add(field, detail string, bad any) {
	*e = append(*e, &FieldError{Field: field, Detail: detail, Bad: bad})
}

// Err returns nil when the list is empty, or a wrapped error otherwise.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	parts := make([]string, 0, len(e))
	for _, fe := range e {
		parts = append(parts, fe.Error())
	}
	return errors.New(strings.Join(parts, "; "))
}

// Has reports whether the list contains an error for the given field.
func (e Errors) Has(field string) bool {
	for _, fe := range e {
		if fe.Field == field {
			return true
		}
	}
	return false
}

// IntInRange returns nil if v is within [min, max] inclusive, otherwise a
// FieldError describing the violation.
func IntInRange(field string, v, min, max int) *FieldError {
	if v < min || v > max {
		return &FieldError{
			Field:  field,
			Detail: fmt.Sprintf("must be between %d and %d", min, max),
			Bad:    v,
		}
	}
	return nil
}

// OneOf returns nil if v is found in allowed.
func OneOf(field, v string, allowed ...string) *FieldError {
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return &FieldError{
		Field:  field,
		Detail: fmt.Sprintf("must be one of %s", strings.Join(allowed, ", ")),
		Bad:    v,
	}
}

// NonEmpty returns a FieldError if v is the empty string.
func NonEmpty(field, v string) *FieldError {
	if v == "" {
		return &FieldError{Field: field, Detail: "must not be empty", Bad: v}
	}
	return nil
}

// MaxLength returns a FieldError if len(v) exceeds n.
func MaxLength(field, v string, n int) *FieldError {
	if len(v) > n {
		return &FieldError{
			Field:  field,
			Detail: fmt.Sprintf("must be at most %d characters", n),
			Bad:    len(v),
		}
	}
	return nil
}

// MinLength returns a FieldError if len(v) is below n.
func MinLength(field, v string, n int) *FieldError {
	if len(v) < n {
		return &FieldError{
			Field:  field,
			Detail: fmt.Sprintf("must be at least %d characters", n),
			Bad:    len(v),
		}
	}
	return nil
}

// IsDNSLabel reports whether s is a valid RFC1035 DNS label (lowercase,
// 1-63 chars, must start and end with an alphanumeric).
func IsDNSLabel(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		isFirst := i == 0
		isLast := i == len(s)-1
		if isFirst || isLast {
			if !isLowerAlnum(r) {
				return false
			}
			continue
		}
		if !isLowerAlnum(r) && r != '-' {
			return false
		}
	}
	return true
}

func isLowerAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// DNSLabel returns a FieldError if s is not a valid DNS-1035 label.
func DNSLabel(field, s string) *FieldError {
	if IsDNSLabel(s) {
		return nil
	}
	return &FieldError{
		Field:  field,
		Detail: "must be a valid RFC1035 DNS label (lowercase, alphanumeric, hyphens, 1-63 chars)",
		Bad:    s,
	}
}

// Append is a convenience that drops nil errors and returns the updated list.
func Append(errs Errors, e ...*FieldError) Errors {
	for _, x := range e {
		if x != nil {
			errs = append(errs, x)
		}
	}
	return errs
}
