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

package imageutil_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/imageutil"
)

func TestParse_EmptyError(t *testing.T) {
	if _, err := imageutil.Parse(""); err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParse_TagOnly(t *testing.T) {
	r, err := imageutil.Parse("postgres:16")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Repository != "postgres" || r.Tag != "16" || r.Registry != "" {
		t.Errorf("parsed = %+v", r)
	}
}

func TestParse_RegistryRepoTag(t *testing.T) {
	r, err := imageutil.Parse("ghcr.io/kitio-tek/athos:v1.2.3")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Registry != "ghcr.io" || r.Repository != "kitio-tek/athos" || r.Tag != "v1.2.3" {
		t.Errorf("parsed = %+v", r)
	}
}

func TestParse_RegistryWithPort(t *testing.T) {
	r, err := imageutil.Parse("registry.local:5000/foo/bar:1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Registry != "registry.local:5000" || r.Repository != "foo/bar" || r.Tag != "1" {
		t.Errorf("parsed = %+v", r)
	}
}

func TestParse_Digest(t *testing.T) {
	r, err := imageutil.Parse("ghcr.io/foo/bar@sha256:abc")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Digest != "sha256:abc" {
		t.Errorf("Digest = %q", r.Digest)
	}
	if !r.IsPinned() {
		t.Error("expected IsPinned true")
	}
}

func TestParse_TagAndDigest(t *testing.T) {
	r, err := imageutil.Parse("foo/bar:1.0@sha256:abc")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Tag != "1.0" || r.Digest != "sha256:abc" {
		t.Errorf("parsed = %+v", r)
	}
	if r.IsTagged() {
		t.Error("IsTagged should be false when digest is set")
	}
}

func TestParse_Localhost(t *testing.T) {
	r, err := imageutil.Parse("localhost/foo/bar")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Registry != "localhost" || r.Repository != "foo/bar" {
		t.Errorf("parsed = %+v", r)
	}
}

func TestParse_DefaultRegistryShortName(t *testing.T) {
	r, err := imageutil.Parse("library/postgres:16")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Registry != "" || r.Repository != "library/postgres" {
		t.Errorf("parsed = %+v", r)
	}
}

func TestString_Roundtrip(t *testing.T) {
	cases := []string{
		"postgres:16",
		"ghcr.io/foo/bar:v1",
		"foo/bar@sha256:deadbeef",
		"registry.local:5000/foo:1",
	}
	for _, in := range cases {
		r, err := imageutil.Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if r.String() == "" {
			t.Errorf("String() for %q is empty", in)
		}
	}
}

func TestWithTag(t *testing.T) {
	r, _ := imageutil.Parse("ghcr.io/foo/bar@sha256:abc")
	withTag := r.WithTag("v2")
	if withTag.Tag != "v2" || withTag.Digest != "" {
		t.Errorf("WithTag did not clear digest: %+v", withTag)
	}
}

func TestWithDigest(t *testing.T) {
	r, _ := imageutil.Parse("ghcr.io/foo/bar:1")
	withDigest := r.WithDigest("sha256:xyz")
	if withDigest.Digest != "sha256:xyz" {
		t.Errorf("WithDigest = %+v", withDigest)
	}
}

func TestPostgresImage(t *testing.T) {
	got := imageutil.PostgresImage(16)
	if !strings.Contains(got, "postgres:16-alpine") {
		t.Errorf("PostgresImage(16) = %q", got)
	}
}
