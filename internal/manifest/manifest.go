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

// Package manifest provides utilities for splitting and merging multi-document
// YAML files. The operator's release pipeline uses this package to assemble
// the install bundle (CRDs + RBAC + Deployment) from individual manifests
// produced by controller-gen.
package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Document is a single YAML document along with its 1-based index in the
// source stream.
type Document struct {
	Index int
	Body  string
}

// Split parses a multi-document YAML stream (documents separated by `---`)
// and returns one Document per non-empty document. Leading and trailing
// whitespace is preserved within each document body so re-assembling them is
// idempotent.
func Split(r io.Reader) ([]Document, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var docs []Document
	var current strings.Builder
	idx := 1

	flush := func() {
		body := current.String()
		if strings.TrimSpace(body) == "" {
			return
		}
		docs = append(docs, Document{Index: idx, Body: body})
		idx++
		current.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		if isDocumentSeparator(line) {
			flush()
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("manifest: scan: %w", err)
	}
	flush()
	return docs, nil
}

func isDocumentSeparator(line string) bool {
	return strings.TrimSpace(line) == "---"
}

// Join concatenates the documents back into a single YAML stream with `---`
// separators between them. The output ends in a trailing newline.
func Join(docs []Document) string {
	if len(docs) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i, d := range docs {
		if i > 0 {
			buf.WriteString("---\n")
		}
		body := strings.TrimRight(d.Body, "\n")
		buf.WriteString(body)
		buf.WriteByte('\n')
	}
	return buf.String()
}

// Filter returns only the documents for which keep returns true.
func Filter(docs []Document, keep func(Document) bool) []Document {
	out := make([]Document, 0, len(docs))
	for _, d := range docs {
		if keep(d) {
			out = append(out, d)
		}
	}
	return out
}

// Has reports whether any document in docs matches the given Kubernetes Kind.
// The match is case-sensitive and only inspects YAML lines beginning with
// "kind:" — if you need full schema validation, use a real YAML parser.
func Has(docs []Document, kind string) bool {
	for _, d := range docs {
		for _, line := range strings.Split(d.Body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "kind:") {
				value := strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
				if value == kind {
					return true
				}
			}
		}
	}
	return false
}

// CountByKind returns a histogram of Kubernetes Kinds across the documents.
func CountByKind(docs []Document) map[string]int {
	out := map[string]int{}
	for _, d := range docs {
		for _, line := range strings.Split(d.Body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "kind:") {
				value := strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
				if value != "" {
					out[value]++
					break
				}
			}
		}
	}
	return out
}
