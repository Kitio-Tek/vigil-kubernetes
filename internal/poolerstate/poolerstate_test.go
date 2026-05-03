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

package poolerstate_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/poolerstate"
)

func TestEvaluate_NeedsCreate(t *testing.T) {
	d := poolerstate.Evaluate(poolerstate.Observed{})
	if d.Action != poolerstate.ActionCreate {
		t.Errorf("got %s, want Create", d.Action)
	}
}

func TestEvaluate_DesiredState(t *testing.T) {
	d := poolerstate.Evaluate(poolerstate.Observed{
		DesiredReplicas: 2, CurrentReplicas: 2, ReadyReplicas: 2,
		ConfigMapExists: true, ServiceExists: true,
		DesiredConfig: "x", CurrentConfig: "x",
		DeploymentReady: true,
	})
	if d.Action != poolerstate.ActionNothing {
		t.Errorf("got %s, want Nothing", d.Action)
	}
	if !d.IsTerminal() {
		t.Error("Nothing should be terminal")
	}
}

func TestEvaluate_ConfigDrift(t *testing.T) {
	d := poolerstate.Evaluate(poolerstate.Observed{
		ConfigMapExists: true, ServiceExists: true,
		DesiredConfig: "new", CurrentConfig: "old",
		DesiredReplicas: 2, CurrentReplicas: 2,
		DeploymentReady: true,
	})
	if d.Action != poolerstate.ActionUpdateConfig {
		t.Errorf("got %s, want UpdateConfig", d.Action)
	}
}

func TestEvaluate_Scale(t *testing.T) {
	d := poolerstate.Evaluate(poolerstate.Observed{
		ConfigMapExists: true, ServiceExists: true,
		DesiredReplicas: 3, CurrentReplicas: 1,
	})
	if d.Action != poolerstate.ActionScale {
		t.Errorf("got %s, want Scale", d.Action)
	}
	if !strings.Contains(d.Reason, "1") || !strings.Contains(d.Reason, "3") {
		t.Errorf("reason missing replica counts: %q", d.Reason)
	}
}

func TestEvaluate_AwaitReady(t *testing.T) {
	d := poolerstate.Evaluate(poolerstate.Observed{
		ConfigMapExists: true, ServiceExists: true,
		DesiredReplicas: 2, CurrentReplicas: 2, ReadyReplicas: 0,
		DeploymentReady: false,
	})
	if d.Action != poolerstate.ActionAwaitReady {
		t.Errorf("got %s, want AwaitReady", d.Action)
	}
	if !d.IsTerminal() {
		t.Error("AwaitReady should be terminal")
	}
}

func TestDecision_String(t *testing.T) {
	d := poolerstate.Decision{Action: poolerstate.ActionScale, Reason: "ok"}
	if got := d.String(); !strings.HasPrefix(got, "Scale:") {
		t.Errorf("String = %q", got)
	}
}
