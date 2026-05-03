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

package password_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/Kitio-Tek/athos-kubernetes/internal/password"
)

func TestDefaultGenerator(t *testing.T) {
	g := password.DefaultGenerator()
	p, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p) != password.DefaultPasswordLength {
		t.Errorf("expected length %d, got %d", password.DefaultPasswordLength, len(p))
	}
}

func TestGeneratorLength(t *testing.T) {
	tests := []struct {
		length  int
		wantErr bool
	}{
		{password.MinPasswordLength, false},
		{password.MaxPasswordLength, false},
		{password.DefaultPasswordLength, false},
		{password.MinPasswordLength - 1, true},
		{password.MaxPasswordLength + 1, true},
		{0, true},
	}
	for _, tt := range tests {
		g, err := password.NewGenerator(password.DefaultAlphabet, tt.length)
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected error for length %d, got nil", tt.length)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for length %d: %v", tt.length, err)
			continue
		}
		p, err := g.Generate()
		if err != nil {
			t.Errorf("Generate() error for length %d: %v", tt.length, err)
		}
		if len(p) != tt.length {
			t.Errorf("expected length %d, got %d", tt.length, len(p))
		}
	}
}

func TestGeneratorEmptyAlphabet(t *testing.T) {
	_, err := password.NewGenerator("", password.DefaultPasswordLength)
	if err == nil {
		t.Error("expected error for empty alphabet")
	}
}

func TestGeneratorAlphabetConstraint(t *testing.T) {
	alphabet := "abc"
	g, err := password.NewGenerator(alphabet, password.MinPasswordLength)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	for _, c := range p {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("character %q not in alphabet %q", c, alphabet)
		}
	}
}

func TestGenerateUniqueness(t *testing.T) {
	g := password.DefaultGenerator()
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		p, err := g.Generate()
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		if seen[p] {
			t.Error("duplicate password generated (extremely unlikely, may indicate RNG issue)")
		}
		seen[p] = true
	}
}

func TestDefaultAlphabetNoSpecialChars(t *testing.T) {
	for _, c := range password.DefaultAlphabet {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			t.Errorf("unexpected special character %q in default alphabet", c)
		}
	}
}

func TestGenerateToken(t *testing.T) {
	tok, err := password.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}
	if len(tok) == 0 {
		t.Error("expected non-empty token")
	}
}

func TestGenerateTokenNegativeLength(t *testing.T) {
	_, err := password.GenerateToken(0)
	if err == nil {
		t.Error("expected error for zero byte length")
	}
	_, err = password.GenerateToken(-1)
	if err == nil {
		t.Error("expected error for negative byte length")
	}
}

func TestGenerateTokenUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		tok, err := password.GenerateToken(16)
		if err != nil {
			t.Fatalf("GenerateToken() error: %v", err)
		}
		if seen[tok] {
			t.Error("duplicate token generated")
		}
		seen[tok] = true
	}
}
