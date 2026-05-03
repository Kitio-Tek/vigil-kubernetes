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

package finalizers_test

import (
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/finalizers"
)

func TestAdd_NewFinalizer(t *testing.T) {
	out, changed := finalizers.Add(nil, "x")
	if !changed {
		t.Error("expected changed=true when adding new")
	}
	if len(out) != 1 || out[0] != "x" {
		t.Errorf("out = %+v", out)
	}
}

func TestAdd_AlreadyPresent(t *testing.T) {
	out, changed := finalizers.Add([]string{"x"}, "x")
	if changed {
		t.Error("expected changed=false when finalizer already present")
	}
	if len(out) != 1 {
		t.Errorf("len = %d", len(out))
	}
}

func TestAdd_PreservesOrder(t *testing.T) {
	out, _ := finalizers.Add([]string{"a", "b"}, "c")
	if len(out) != 3 || out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Errorf("order = %+v", out)
	}
}

func TestRemove_Present(t *testing.T) {
	out, changed := finalizers.Remove([]string{"a", "b", "c"}, "b")
	if !changed {
		t.Error("expected changed=true")
	}
	if len(out) != 2 {
		t.Errorf("len = %d", len(out))
	}
	if finalizers.Contains(out, "b") {
		t.Error("Remove did not drop b")
	}
}

func TestRemove_Absent(t *testing.T) {
	_, changed := finalizers.Remove([]string{"a"}, "b")
	if changed {
		t.Error("expected changed=false")
	}
}

func TestRemove_AllOccurrences(t *testing.T) {
	out, _ := finalizers.Remove([]string{"a", "x", "b", "x"}, "x")
	if len(out) != 2 {
		t.Errorf("expected 2, got %d", len(out))
	}
}

func TestContains(t *testing.T) {
	if !finalizers.Contains([]string{"a", "b"}, "b") {
		t.Error("expected Contains true")
	}
	if finalizers.Contains([]string{"a"}, "b") {
		t.Error("expected Contains false")
	}
}

func TestAthosFinalizer(t *testing.T) {
	want := []string{
		finalizers.PostgresClusterFinalizer,
		finalizers.PostgresBackupFinalizer,
		finalizers.PostgresUserFinalizer,
		finalizers.PostgresPoolerFinalizer,
	}
	for _, f := range want {
		if !finalizers.AthosFinalizer(f) {
			t.Errorf("expected AthosFinalizer(%q) = true", f)
		}
	}
	if finalizers.AthosFinalizer("foreign.io/finalizer") {
		t.Error("foreign finalizer should not be classified as Athos")
	}
}
