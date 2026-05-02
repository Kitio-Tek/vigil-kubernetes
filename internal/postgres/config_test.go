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

package postgres_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/postgres"
)

func TestDefaultParams(t *testing.T) {
	p := postgres.DefaultParams()
	if len(p) == 0 {
		t.Fatal("expected non-empty default params")
	}
	required := []string{"listen_addresses", "max_connections", "shared_buffers", "wal_level"}
	for _, k := range required {
		if _, ok := p[k]; !ok {
			t.Errorf("default params missing required key %q", k)
		}
	}
}

func TestMergeParams(t *testing.T) {
	base := map[string]string{
		"shared_buffers": "128MB",
		"max_connections": "100",
	}
	override := map[string]string{
		"shared_buffers": "256MB",
		"work_mem": "8MB",
	}
	merged := postgres.MergeParams(base, override)

	if merged["shared_buffers"] != "256MB" {
		t.Errorf("override value not applied: got %q, want 256MB", merged["shared_buffers"])
	}
	if merged["max_connections"] != "100" {
		t.Errorf("base value should be preserved: got %q, want 100", merged["max_connections"])
	}
	if merged["work_mem"] != "8MB" {
		t.Errorf("new key from override missing: got %q, want 8MB", merged["work_mem"])
	}
}

func TestMergeParamsDoesNotMutateBase(t *testing.T) {
	base := map[string]string{"a": "1"}
	override := map[string]string{"a": "2"}
	postgres.MergeParams(base, override)
	if base["a"] != "1" {
		t.Error("MergeParams must not mutate the base map")
	}
}

func TestBuildPostgresConf(t *testing.T) {
	params := map[string]string{
		"listen_addresses": "'*'",
		"max_connections":  "50",
	}
	conf := postgres.BuildPostgresConf(params)

	if !strings.Contains(conf, "listen_addresses") {
		t.Error("conf missing listen_addresses")
	}
	if !strings.Contains(conf, "max_connections") {
		t.Error("conf missing max_connections")
	}
	// Keys should appear in alphabetical order.
	idxListen := strings.Index(conf, "listen_addresses")
	idxMax := strings.Index(conf, "max_connections")
	if idxListen >= idxMax {
		t.Error("expected listen_addresses before max_connections (alphabetical order)")
	}
}

func TestBuildPostgresConfHeader(t *testing.T) {
	conf := postgres.BuildPostgresConf(map[string]string{})
	if !strings.HasPrefix(conf, "#") {
		t.Error("conf should start with a comment header")
	}
}
