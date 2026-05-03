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

package workflows_test

import (
	"errors"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/workflows"
)

func TestDone(t *testing.T) {
	r := workflows.Done()
	if r.Requeue {
		t.Error("Done() should not requeue")
	}
	if r.Err != nil {
		t.Error("Done() should not have an error")
	}
	result, err := r.Ctrl()
	if err != nil || result.Requeue {
		t.Error("Done().Ctrl() should return non-requeue, nil error")
	}
}

func TestRequeueErr(t *testing.T) {
	sentinel := errors.New("test error")
	r := workflows.RequeueErr(sentinel)
	if !r.Requeue {
		t.Error("RequeueErr should set Requeue=true")
	}
	if r.Err != sentinel {
		t.Errorf("expected sentinel error, got %v", r.Err)
	}
	_, err := r.Ctrl()
	if err != sentinel {
		t.Errorf("Ctrl() should propagate the error, got %v", err)
	}
}

func TestRequeueAfter(t *testing.T) {
	r := workflows.RequeueAfter()
	if !r.Requeue {
		t.Error("RequeueAfter should set Requeue=true")
	}
	if r.Err != nil {
		t.Error("RequeueAfter should not have an error")
	}
	result, err := r.Ctrl()
	if err != nil {
		t.Error("RequeueAfter().Ctrl() should return nil error")
	}
	if !result.Requeue {
		t.Error("RequeueAfter().Ctrl() should requeue")
	}
}
