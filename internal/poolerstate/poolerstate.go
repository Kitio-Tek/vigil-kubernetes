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

// Package poolerstate is the PgBouncer counterpart of clusterstate. It
// computes the next reconcile action for a PostgresPooler from observed
// state and the desired pooler spec.
package poolerstate

import "fmt"

// Action enumerates the high-level decisions the PostgresPooler controller
// can make.
type Action string

const (
	// ActionNothing means the pooler is in the desired state.
	ActionNothing Action = "Nothing"
	// ActionCreate means the deployment, service or config map needs to be created.
	ActionCreate Action = "Create"
	// ActionScale means the replica count must be adjusted.
	ActionScale Action = "Scale"
	// ActionUpdateConfig means the PgBouncer config has drifted from spec.
	ActionUpdateConfig Action = "UpdateConfig"
	// ActionAwaitReady means we are waiting on pods to be ready.
	ActionAwaitReady Action = "AwaitReady"
)

// Observed captures the current state of a pooler deployment.
type Observed struct {
	DesiredReplicas int32
	CurrentReplicas int32
	ReadyReplicas   int32
	DesiredConfig   string
	CurrentConfig   string
	DeploymentReady bool
	ConfigMapExists bool
	ServiceExists   bool
}

// Decision is the output of Evaluate.
type Decision struct {
	Action Action
	Reason string
}

// String renders the decision in a "Action: Reason" form.
func (d Decision) String() string { return fmt.Sprintf("%s: %s", d.Action, d.Reason) }

// Evaluate inspects the observed state and returns the next action.
func Evaluate(o Observed) Decision {
	if !o.ConfigMapExists || !o.ServiceExists {
		return Decision{Action: ActionCreate, Reason: "missing config map or service"}
	}
	if o.DesiredConfig != "" && o.DesiredConfig != o.CurrentConfig {
		return Decision{Action: ActionUpdateConfig, Reason: "config drift detected"}
	}
	if o.DesiredReplicas != o.CurrentReplicas {
		return Decision{
			Action: ActionScale,
			Reason: fmt.Sprintf("scale from %d to %d replicas", o.CurrentReplicas, o.DesiredReplicas),
		}
	}
	if !o.DeploymentReady || o.ReadyReplicas < o.DesiredReplicas {
		return Decision{
			Action: ActionAwaitReady,
			Reason: fmt.Sprintf("%d/%d replicas ready", o.ReadyReplicas, o.DesiredReplicas),
		}
	}
	return Decision{Action: ActionNothing, Reason: "pooler is in the desired state"}
}

// IsTerminal reports whether the decision indicates no further work is
// required for this reconcile cycle.
func (d Decision) IsTerminal() bool {
	switch d.Action {
	case ActionNothing, ActionAwaitReady:
		return true
	}
	return false
}
