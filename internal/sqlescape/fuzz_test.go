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

// FuzzIdentifier asserts that no input can produce an output that breaks out
// of the surrounding double-quoted identifier grammar. Concretely, every
// double quote that appears in the encoded form must be part of either the
// outer wrapping pair or a doubled escape sequence.
func FuzzIdentifier(f *testing.F) {
	seeds := []string{
		"",
		"users",
		"weird\"name",
		"weird\"\"name",
		"a\x00b",
		"abc; DROP TABLE users; --",
		"abc' OR '1'='1",
		"\xff\xfe\x00",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		out := sqlescape.Identifier(name)
		if !strings.HasPrefix(out, `"`) || !strings.HasSuffix(out, `"`) {
			t.Fatalf("Identifier(%q) = %q: missing wrapping quotes", name, out)
		}
		inner := out[1 : len(out)-1]
		// After stripping the wrapping pair, every double quote left in
		// the inner body must be part of a doubled-quote escape sequence.
		for i := 0; i < len(inner); i++ {
			if inner[i] != '"' {
				continue
			}
			if i+1 >= len(inner) || inner[i+1] != '"' {
				t.Fatalf("Identifier(%q) = %q: lone double quote at offset %d", name, out, i)
			}
			i++
		}
	})
}

// FuzzStringLiteral asserts the same invariant for single-quoted string
// literals. No input may escape the surrounding pair of single quotes.
func FuzzStringLiteral(f *testing.F) {
	seeds := []string{
		"",
		"it's",
		"plain",
		"'; DROP TABLE users; --",
		"a\x00b",
		"\\'",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		out := sqlescape.StringLiteral(in)
		if !strings.HasPrefix(out, "'") || !strings.HasSuffix(out, "'") {
			t.Fatalf("StringLiteral(%q) = %q: missing wrapping quotes", in, out)
		}
		inner := out[1 : len(out)-1]
		for i := 0; i < len(inner); i++ {
			if inner[i] != '\'' {
				continue
			}
			if i+1 >= len(inner) || inner[i+1] != '\'' {
				t.Fatalf("StringLiteral(%q) = %q: lone single quote at offset %d", in, out, i)
			}
			i++
		}
	})
}

// FuzzIsValidIdentifier asserts that any name accepted by IsValidIdentifier
// round-trips through Identifier unchanged in its escape body (no characters
// require doubling).
func FuzzIsValidIdentifier(f *testing.F) {
	seeds := []string{
		"users",
		"_users",
		"weird$name",
		"good-name",
		"with.dot",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		if !sqlescape.IsValidIdentifier(name) {
			return
		}
		got := sqlescape.Identifier(name)
		want := `"` + name + `"`
		if got != want {
			t.Fatalf("IsValidIdentifier(%q) accepted but Identifier produced %q", name, got)
		}
	})
}
