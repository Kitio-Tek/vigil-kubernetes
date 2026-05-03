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

package events_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/events"
)

func TestReconcileMessage(t *testing.T) {
	msg := events.ReconcileMessage("StatefulSet", "my-cluster", "created")
	if msg != "StatefulSet my-cluster created" {
		t.Errorf("unexpected message: %q", msg)
	}
}

func TestUpgradeMessage(t *testing.T) {
	msg := events.UpgradeMessage(15, 16)
	if !strings.Contains(msg, "15") || !strings.Contains(msg, "16") {
		t.Errorf("upgrade message should contain both versions, got: %q", msg)
	}
}

func TestBackupMessage(t *testing.T) {
	msg := events.BackupMessage("my-backup")
	if !strings.Contains(msg, "my-backup") {
		t.Errorf("backup message should contain name, got: %q", msg)
	}
}

func TestFailoverMessage(t *testing.T) {
	msg := events.FailoverMessage("pod-0", "pod-1")
	if !strings.Contains(msg, "pod-0") || !strings.Contains(msg, "pod-1") {
		t.Errorf("failover message should contain pod names, got: %q", msg)
	}
}

func TestEventConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"EventReasonCreated", events.EventReasonCreated},
		{"EventReasonUpdated", events.EventReasonUpdated},
		{"EventReasonDeleted", events.EventReasonDeleted},
		{"EventReasonFailed", events.EventReasonFailed},
		{"EventReasonPaused", events.EventReasonPaused},
		{"EventReasonResumed", events.EventReasonResumed},
		{"EventReasonUpgradeStarted", events.EventReasonUpgradeStarted},
		{"EventReasonBackupStarted", events.EventReasonBackupStarted},
		{"EventReasonFailoverStarted", events.EventReasonFailoverStarted},
		{"EventTypeNormal", events.EventTypeNormal},
		{"EventTypeWarning", events.EventTypeWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("constant %s should not be empty", tt.name)
			}
		})
	}
}

func TestUpgradeMessageNegativeVersion(t *testing.T) {
	msg := events.UpgradeMessage(0, 16)
	if !strings.Contains(msg, "0") {
		t.Errorf("upgrade message with zero version, got: %q", msg)
	}
}
