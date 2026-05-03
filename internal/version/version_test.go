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

package version_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/version"
)

func TestInfo_PopulatesPlatform(t *testing.T) {
	info := version.Info()
	if info.Platform == "" {
		t.Error("Platform should be populated")
	}
	if !strings.Contains(info.Platform, runtime.GOOS) {
		t.Errorf("Platform = %q, expected to contain %q", info.Platform, runtime.GOOS)
	}
}

func TestInfo_GoVersion(t *testing.T) {
	info := version.Info()
	if info.GoVersion == "" {
		t.Error("GoVersion should be populated")
	}
}

func TestString_NotEmpty(t *testing.T) {
	if version.String() == "" {
		t.Error("String() should not be empty")
	}
	if !strings.Contains(version.String(), version.Product) {
		t.Errorf("String() should contain product name, got %q", version.String())
	}
}

func TestShortVersion_StripsV(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()

	cases := map[string]string{
		"v1.2.3":           "1.2",
		"1.2.3":            "1.2",
		"v2.0.0-rc1":       "2.0",
		"v1.0":             "1.0",
		"v1.2.3+build.42":  "1.2",
	}
	for in, want := range cases {
		version.Version = in
		if got := version.ShortVersion(); got != want {
			t.Errorf("ShortVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShortVersion_Unparseable(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()

	version.Version = "weird"
	if got := version.ShortVersion(); got != "weird" {
		t.Errorf("Unparseable version should be returned unchanged, got %q", got)
	}
}

func TestUserAgent(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()

	version.Version = "v1.2.3"
	got := version.UserAgent()
	want := version.Product + "/1.2"
	if got != want {
		t.Errorf("UserAgent = %q, want %q", got, want)
	}
}
