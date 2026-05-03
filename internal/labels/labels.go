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

// Package labels offers small label-manipulation helpers used by every
// reconciler in this operator. The internal/postgres package owns the
// authoritative set of label keys; this package focuses on safe map
// operations like merging, validation and label-selector formatting.
package labels

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Set is an alias for map[string]string. It is provided so callers can write
// code closer to the upstream k8s.io/apimachinery/pkg/labels Set type without
// pulling in that package transitively.
type Set = map[string]string

// Merge returns a new label set containing the union of base and overlay,
// with overlay taking precedence on conflicts.
func Merge(base, overlay Set) Set {
	out := make(Set, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// MergeAll merges any number of sets in order; later maps override earlier ones.
func MergeAll(sets ...Set) Set {
	out := Set{}
	for _, s := range sets {
		for k, v := range s {
			out[k] = v
		}
	}
	return out
}

// Equal reports whether a and b contain exactly the same key/value pairs.
func Equal(a, b Set) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// HasAll reports whether every key/value pair in want is present in got.
func HasAll(got, want Set) bool {
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// SelectorString formats labels as the canonical comma-separated key=value
// expression accepted by `kubectl --selector`.
func SelectorString(s Set) string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, s[k]))
	}
	return strings.Join(parts, ",")
}

// IsValidKey reports whether s is a valid label key per the rules of
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/.
//
// Validation here is intentionally lenient compared to apimachinery's
// validation: this helper exists to provide quick feedback in CRD
// validation webhooks before the apiserver runs full validation.
func IsValidKey(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	if i := strings.Index(s, "/"); i != -1 {
		prefix, name := s[:i], s[i+1:]
		if !isDNSSubdomain(prefix) {
			return false
		}
		return isLabelName(name)
	}
	return isLabelName(s)
}

// IsValidValue reports whether s is a valid label value.
func IsValidValue(s string) bool {
	if len(s) > 63 {
		return false
	}
	if s == "" {
		return true
	}
	return isLabelName(s)
}

func isLabelName(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	first := rune(s[0])
	last := rune(s[len(s)-1])
	if !isAlnum(first) || !isAlnum(last) {
		return false
	}
	for _, r := range s {
		if !isAlnum(r) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func isDNSSubdomain(s string) bool {
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		first := rune(label[0])
		last := rune(label[len(label)-1])
		if !unicode.IsLower(first) && !unicode.IsDigit(first) {
			return false
		}
		if !unicode.IsLower(last) && !unicode.IsDigit(last) {
			return false
		}
		for _, r := range label {
			if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '-' {
				return false
			}
		}
	}
	return true
}

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// Validate returns an error if any key or value in the set is malformed.
func Validate(s Set) error {
	for k, v := range s {
		if !IsValidKey(k) {
			return fmt.Errorf("invalid label key: %q", k)
		}
		if !IsValidValue(v) {
			return fmt.Errorf("invalid label value for key %q: %q", k, v)
		}
	}
	return nil
}

// Subtract returns a new set containing the keys in a that are not in b.
func Subtract(a, b Set) Set {
	out := Set{}
	for k, v := range a {
		if _, ok := b[k]; !ok {
			out[k] = v
		}
	}
	return out
}

// Keys returns the sorted keys of the set.
func Keys(s Set) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
