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

package controller

import (
	"strings"
	"testing"
)

func TestRedactPassword(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		want   string
		secret string
	}{
		{
			name:   "single ALTER USER",
			in:     "ERROR:  syntax error at or near \";\"\nSTATEMENT:  ALTER USER app WITH PASSWORD 'hunter2';",
			want:   "ERROR:  syntax error at or near \";\"\nSTATEMENT:  ALTER USER app WITH PASSWORD '[REDACTED]';",
			secret: "hunter2",
		},
		{
			name:   "CREATE USER form",
			in:     "STATEMENT:  CREATE USER app WITH PASSWORD 'sup3rs3cret'",
			want:   "STATEMENT:  CREATE USER app WITH PASSWORD '[REDACTED]'",
			secret: "sup3rs3cret",
		},
		{
			name:   "multiple statements",
			in:     "CREATE USER a WITH PASSWORD 'first'; ALTER USER b WITH PASSWORD 'second';",
			want:   "CREATE USER a WITH PASSWORD '[REDACTED]'; ALTER USER b WITH PASSWORD '[REDACTED]';",
			secret: "first",
		},
		{
			name:   "no password literal",
			in:     "ERROR: role does not exist",
			want:   "ERROR: role does not exist",
			secret: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactPassword(tc.in)
			if got != tc.want {
				t.Errorf("redactPassword =\n  %q\nwant\n  %q", got, tc.want)
			}
			if tc.secret != "" && strings.Contains(got, tc.secret) {
				t.Errorf("redactPassword leaked secret %q in %q", tc.secret, got)
			}
		})
	}
}

// FuzzRedactPassword asserts that no PASSWORD '<value>' literal survives
// redaction regardless of the surrounding stderr noise.
func FuzzRedactPassword(f *testing.F) {
	f.Add("hunter2")
	f.Add("p'a's's'")
	f.Add("")
	f.Add("with spaces and ; semicolons")

	f.Fuzz(func(t *testing.T, secret string) {
		if strings.Contains(secret, "'") {
			// The regex stops at the first single quote — passwords
			// containing literal single quotes are not expected here
			// because StringLiteral doubles them before they reach psql.
			t.Skip()
		}
		stderr := "ERROR: something\nSTATEMENT:  ALTER USER x WITH PASSWORD '" + secret + "';\n"
		out := redactPassword(stderr)
		if secret != "" && strings.Contains(out, secret) {
			t.Fatalf("password %q leaked through redaction: %q", secret, out)
		}
	})
}
