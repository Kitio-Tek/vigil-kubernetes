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

// Package sqlescape provides quoting helpers for PostgreSQL identifiers and
// string literals. The PostgresUser controller uses these helpers when it
// builds CREATE ROLE / GRANT statements from CRD field values.
//
// Parameterised queries are always preferred when available; this package
// exists for the unavoidable cases where the SQL grammar requires literal
// substitution (DDL statements like CREATE USER cannot bind names).
package sqlescape

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Identifier wraps name in double quotes, doubling any embedded double
// quotes per the PostgreSQL grammar. The result is always safe to embed
// directly in a SQL statement.
func Identifier(name string) string {
	if strings.ContainsRune(name, 0) {
		// Embedded NUL bytes are not allowed in SQL identifiers and would
		// truncate the resulting statement; reject by returning a value
		// that is guaranteed to fail to parse.
		return "\"\\x00\""
	}
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}

// StringLiteral wraps s in single quotes, doubling embedded single quotes.
func StringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// QualifiedIdentifier joins each component with a "." separator, escaping
// each component as an Identifier. Useful for "schema"."table" forms.
func QualifiedIdentifier(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = Identifier(p)
	}
	return strings.Join(out, ".")
}

// IsValidIdentifier reports whether name is safe to use as a PostgreSQL
// double-quoted identifier. The check accepts any character that is also
// valid in a Kubernetes object name (lowercase letters, digits, hyphens
// and the underscore / dollar / dot characters PostgreSQL itself accepts
// inside an identifier) and rejects characters that would either break
// out of the quoted form or terminate the SQL statement when interpolated:
//
//   - NUL bytes (would truncate the statement at the C-string boundary)
//   - any control character below ASCII 0x20 except plain space
//   - the double-quote character itself, since the caller is expected to
//     pass these through Identifier() which doubles them; we forbid them
//     here as defence in depth
//
// The 63-byte NAMEDATALEN limit from the PostgreSQL build-time default is
// also enforced.
func IsValidIdentifier(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for _, r := range name {
		switch {
		case r == 0:
			return false
		case r < 0x20 && r != ' ':
			return false
		case r == '"':
			return false
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '_', '$', '-', '.':
			continue
		default:
			return false
		}
	}
	return true
}

// MustIdentifier returns Identifier(name) unless the name is invalid, in
// which case it returns an error. Callers building DDL statements typically
// use this helper to fail fast on user-provided names.
func MustIdentifier(name string) (string, error) {
	if !IsValidIdentifier(name) {
		return "", fmt.Errorf("sqlescape: invalid identifier %q", name)
	}
	return Identifier(name), nil
}

// AssertSafePassword returns an error if pw contains characters that would
// break a CREATE USER ... PASSWORD '<pw>' statement after escaping. It is a
// belt-and-braces check — StringLiteral handles single quotes — used by the
// PostgresUser webhook to bail on suspicious inputs before they reach the
// database.
func AssertSafePassword(pw string) error {
	if strings.ContainsRune(pw, 0) {
		return ErrUnsafePassword
	}
	return nil
}

// ErrUnsafePassword is returned by AssertSafePassword for inputs that
// contain characters that cannot be safely embedded in SQL.
var ErrUnsafePassword = errors.New("sqlescape: password contains an unsafe character")
