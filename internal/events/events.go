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

// Package events provides typed Kubernetes event constants and helpers for the
// Vigil operator. All event reasons are defined here to enforce consistency
// across controllers.
package events

const (
	// EventReasonCreated is emitted when the operator creates a Kubernetes resource.
	EventReasonCreated = "Created"

	// EventReasonUpdated is emitted when the operator updates a Kubernetes resource.
	EventReasonUpdated = "Updated"

	// EventReasonDeleted is emitted when the operator removes a Kubernetes resource.
	EventReasonDeleted = "Deleted"

	// EventReasonFailed is emitted when a reconcile operation fails.
	EventReasonFailed = "Failed"

	// EventReasonPaused is emitted when reconciliation is paused via the spec.
	EventReasonPaused = "Paused"

	// EventReasonResumed is emitted when reconciliation is resumed from paused state.
	EventReasonResumed = "Resumed"

	// EventReasonUpgradeStarted is emitted when a minor or major version upgrade begins.
	EventReasonUpgradeStarted = "UpgradeStarted"

	// EventReasonUpgradeCompleted is emitted when an upgrade completes successfully.
	EventReasonUpgradeCompleted = "UpgradeCompleted"

	// EventReasonUpgradeFailed is emitted when an upgrade fails.
	EventReasonUpgradeFailed = "UpgradeFailed"

	// EventReasonBackupStarted is emitted when a backup Job is created.
	EventReasonBackupStarted = "BackupStarted"

	// EventReasonBackupCompleted is emitted when a backup Job completes.
	EventReasonBackupCompleted = "BackupCompleted"

	// EventReasonBackupFailed is emitted when a backup Job fails.
	EventReasonBackupFailed = "BackupFailed"

	// EventReasonFailoverStarted is emitted when a failover is initiated.
	EventReasonFailoverStarted = "FailoverStarted"

	// EventReasonFailoverCompleted is emitted when a failover completes.
	EventReasonFailoverCompleted = "FailoverCompleted"

	// EventReasonSwitchoverStarted is emitted when a planned switchover is initiated.
	EventReasonSwitchoverStarted = "SwitchoverStarted"

	// EventReasonSwitchoverCompleted is emitted when a switchover completes.
	EventReasonSwitchoverCompleted = "SwitchoverCompleted"

	// EventReasonPasswordRotated is emitted when a credential secret is rotated.
	EventReasonPasswordRotated = "PasswordRotated"

	// EventReasonCertificateRenewed is emitted when a TLS certificate is renewed.
	EventReasonCertificateRenewed = "CertificateRenewed"

	// EventTypeNormal is a non-error event.
	EventTypeNormal = "Normal"

	// EventTypeWarning indicates a potentially harmful situation.
	EventTypeWarning = "Warning"
)

// ReconcileMessage returns a human-readable message for a generic resource
// reconcile event.
func ReconcileMessage(kind, name, action string) string {
	return kind + " " + name + " " + action
}

// UpgradeMessage returns a human-readable message for an upgrade event.
func UpgradeMessage(fromVersion, toVersion int32) string {
	return "PostgreSQL upgrade from version " + itoa(fromVersion) + " to " + itoa(toVersion)
}

// BackupMessage returns a human-readable message for a backup event.
func BackupMessage(backupName string) string {
	return "Backup " + backupName
}

// FailoverMessage returns a human-readable message for a failover event.
func FailoverMessage(fromPod, toPod string) string {
	return "Failover from " + fromPod + " to " + toPod
}

func itoa(n int32) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
