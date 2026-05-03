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

	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

func TestPostgresImageTagKnownVersions(t *testing.T) {
	known := []int32{14, 15, 16, 17}
	for _, v := range known {
		tag := postgres.PostgresImageTag(v)
		if tag == "" {
			t.Errorf("empty image tag for version %d", v)
		}
		vStr := strings.Contains(tag, "14") || strings.Contains(tag, "15") ||
			strings.Contains(tag, "16") || strings.Contains(tag, "17")
		if !vStr {
			t.Errorf("image tag %q does not contain version number", tag)
		}
	}
}

func TestPostgresImageTagAlpine(t *testing.T) {
	tag := postgres.PostgresImageTag(16)
	if !strings.Contains(tag, "alpine") {
		t.Errorf("expected alpine-based image, got %q", tag)
	}
}

func TestExporterImageTag(t *testing.T) {
	tag := postgres.ExporterImageTag()
	if tag == "" {
		t.Fatal("empty exporter image tag")
	}
	if !strings.Contains(tag, "postgres-exporter") {
		t.Errorf("exporter image should reference postgres-exporter, got %q", tag)
	}
}
