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

// Package recovery implements point-in-time recovery (PITR) configuration
// and target validation for the Athos operator. It generates the
// postgresql.auto.conf recovery parameters and validates user-supplied PITR
// targets before a restore operation begins.
package recovery

import (
	"fmt"
	"strings"
	"time"
)

// Target describes a PostgreSQL PITR recovery target.
type Target struct {
	// Kind identifies the type of recovery target.
	Kind TargetKind
	// Time is the recovery target time (used when Kind is TargetKindTime).
	Time time.Time
	// LSN is the recovery target WAL location (used when Kind is TargetKindLSN).
	LSN string
	// XID is the recovery target transaction ID (used when Kind is TargetKindXID).
	XID uint64
	// Name is the recovery target name (used when Kind is TargetKindName).
	Name string
	// Inclusive controls whether the target is inclusive (default: true).
	Inclusive bool
	// Timeline is the recovery target timeline ("latest", or a specific number).
	Timeline string
}

// TargetKind identifies the type of recovery target.
type TargetKind string

const (
	// TargetKindTime recovers to a specific timestamp.
	TargetKindTime TargetKind = "time"
	// TargetKindLSN recovers to a specific WAL log sequence number.
	TargetKindLSN TargetKind = "lsn"
	// TargetKindXID recovers to a specific transaction ID.
	TargetKindXID TargetKind = "xid"
	// TargetKindName recovers to a named restore point.
	TargetKindName TargetKind = "name"
	// TargetKindImmediate stops recovery as soon as a consistent state is reached.
	TargetKindImmediate TargetKind = "immediate"
)

// TimeLayout is the PostgreSQL timestamp format.
const TimeLayout = "2006-01-02 15:04:05 MST"

// Validate returns an error if the target is not a valid PITR specification.
func (t Target) Validate() error {
	switch t.Kind {
	case TargetKindTime:
		if t.Time.IsZero() {
			return fmt.Errorf("recovery target kind %q requires a non-zero time", t.Kind)
		}
		if t.Time.After(time.Now().Add(time.Minute)) {
			return fmt.Errorf("recovery target time %v is in the future", t.Time)
		}
	case TargetKindLSN:
		if !isValidLSN(t.LSN) {
			return fmt.Errorf("recovery target LSN %q is not in the form XXXXXXXX/XXXXXXXX", t.LSN)
		}
	case TargetKindXID:
		if t.XID == 0 {
			return fmt.Errorf("recovery target XID must be non-zero")
		}
	case TargetKindName:
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("recovery target name must not be empty")
		}
	case TargetKindImmediate:
		// No additional fields required.
	default:
		return fmt.Errorf("unknown recovery target kind %q", t.Kind)
	}
	if t.Timeline != "" && t.Timeline != "latest" {
		if _, err := parseUint(t.Timeline); err != nil {
			return fmt.Errorf("recovery target timeline must be \"latest\" or a positive integer")
		}
	}
	return nil
}

// RecoveryParams returns a map of postgresql.auto.conf parameters that
// implement this recovery target.
func (t Target) RecoveryParams() map[string]string {
	params := map[string]string{}

	timeline := t.Timeline
	if timeline == "" {
		timeline = "latest"
	}
	params["recovery_target_timeline"] = timeline

	inclusive := "true"
	if !t.Inclusive {
		inclusive = "false"
	}

	switch t.Kind {
	case TargetKindTime:
		params["recovery_target_time"] = t.Time.UTC().Format(TimeLayout)
		params["recovery_target_inclusive"] = inclusive
	case TargetKindLSN:
		params["recovery_target_lsn"] = t.LSN
		params["recovery_target_inclusive"] = inclusive
	case TargetKindXID:
		params["recovery_target_xid"] = fmt.Sprintf("%d", t.XID)
		params["recovery_target_inclusive"] = inclusive
	case TargetKindName:
		params["recovery_target_name"] = t.Name
	case TargetKindImmediate:
		params["recovery_target"] = "immediate"
	}

	return params
}

// StandbySignalContent returns the content for the standby.signal file, which
// must exist alongside postgresql.auto.conf in PostgreSQL 12+.
func StandbySignalContent() string {
	return "# standby.signal - managed by athos-kubernetes\n"
}

// RecoverySignalContent returns the content for the recovery.signal file used
// to trigger a PITR restore in PostgreSQL 12+.
func RecoverySignalContent() string {
	return "# recovery.signal - managed by athos-kubernetes\n"
}

// ParseTime parses a recovery target time string in PostgreSQL timestamp format.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(TimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing recovery target time %q: use format %q", s, TimeLayout)
	}
	return t, nil
}

// isValidLSN returns true when s matches the PostgreSQL LSN format XX/XX.
func isValidLSN(s string) bool {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 8 {
			return false
		}
		for _, c := range p {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
	}
	return true
}

func parseUint(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid character %q", c)
		}
		n = n*10 + uint64(c-'0')
	}
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}
	return n, nil
}
