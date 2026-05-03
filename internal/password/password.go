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

// Package password provides cryptographically secure password and token
// generation for PostgreSQL credentials managed by the Athos operator.
package password

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

const (
	// DefaultPasswordLength is the default length for generated passwords.
	DefaultPasswordLength = 32

	// DefaultAlphabet is the character set used for password generation.
	// Excludes characters that are ambiguous or problematic in shell contexts.
	DefaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// MinPasswordLength is the minimum allowed password length.
	MinPasswordLength = 16

	// MaxPasswordLength is the maximum allowed password length.
	MaxPasswordLength = 128
)

// Generator generates cryptographically secure passwords.
type Generator struct {
	alphabet string
	length   int
}

// NewGenerator creates a Generator with the supplied alphabet and length.
// Returns an error if the length is outside the valid range.
func NewGenerator(alphabet string, length int) (*Generator, error) {
	if length < MinPasswordLength || length > MaxPasswordLength {
		return nil, fmt.Errorf("password length %d is outside the allowed range [%d, %d]",
			length, MinPasswordLength, MaxPasswordLength)
	}
	if len(alphabet) == 0 {
		return nil, fmt.Errorf("alphabet must not be empty")
	}
	return &Generator{alphabet: alphabet, length: length}, nil
}

// DefaultGenerator returns a Generator with the default settings.
func DefaultGenerator() *Generator {
	return &Generator{alphabet: DefaultAlphabet, length: DefaultPasswordLength}
}

// Generate returns a random password string.
func (g *Generator) Generate() (string, error) {
	buf := make([]byte, g.length)
	alphabetLen := big.NewInt(int64(len(g.alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("generating random byte: %w", err)
		}
		buf[i] = g.alphabet[n.Int64()]
	}
	return string(buf), nil
}

// GenerateToken returns a URL-safe base64-encoded random token of the given
// byte length (the resulting string will be longer due to encoding overhead).
func GenerateToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		return "", fmt.Errorf("token byte length must be positive")
	}
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// MustGenerate is like Generate but panics on error. Only use in tests.
func MustGenerate(g *Generator) string {
	p, err := g.Generate()
	if err != nil {
		panic(err)
	}
	return p
}
