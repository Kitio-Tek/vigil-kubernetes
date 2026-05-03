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

package recovery_test

import (
	"testing"
	"time"

	"github.com/Kitio-Tek/athos-kubernetes/internal/recovery"
)

func pastTime(d time.Duration) time.Time {
	return time.Now().UTC().Add(-d)
}

func TestTarget_Validate_Time(t *testing.T) {
	target := recovery.Target{
		Kind:      recovery.TargetKindTime,
		Time:      pastTime(24 * time.Hour),
		Inclusive: true,
	}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid time target, got: %v", err)
	}
}

func TestTarget_Validate_TimeFuture(t *testing.T) {
	target := recovery.Target{
		Kind:      recovery.TargetKindTime,
		Time:      time.Now().Add(time.Hour),
		Inclusive: true,
	}
	if err := target.Validate(); err == nil {
		t.Error("expected error for future time target")
	}
}

func TestTarget_Validate_TimeZero(t *testing.T) {
	target := recovery.Target{
		Kind: recovery.TargetKindTime,
	}
	if err := target.Validate(); err == nil {
		t.Error("expected error for zero time target")
	}
}

func TestTarget_Validate_LSN(t *testing.T) {
	target := recovery.Target{
		Kind:      recovery.TargetKindLSN,
		LSN:       "0/15D3C28",
		Inclusive: true,
	}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid LSN target, got: %v", err)
	}
}

func TestTarget_Validate_InvalidLSN(t *testing.T) {
	invalid := []string{"", "notalsnq", "0/", "/AABBCCDD", "XXYY/AABB"}
	for _, lsn := range invalid {
		target := recovery.Target{Kind: recovery.TargetKindLSN, LSN: lsn}
		if err := target.Validate(); err == nil {
			t.Errorf("expected error for invalid LSN %q", lsn)
		}
	}
}

func TestTarget_Validate_XID(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindXID, XID: 12345}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid XID target, got: %v", err)
	}
}

func TestTarget_Validate_XIDZero(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindXID, XID: 0}
	if err := target.Validate(); err == nil {
		t.Error("expected error for XID=0")
	}
}

func TestTarget_Validate_Name(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindName, Name: "before_migration"}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid name target, got: %v", err)
	}
}

func TestTarget_Validate_NameEmpty(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindName, Name: "  "}
	if err := target.Validate(); err == nil {
		t.Error("expected error for empty name target")
	}
}

func TestTarget_Validate_Immediate(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindImmediate}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid immediate target, got: %v", err)
	}
}

func TestTarget_Validate_InvalidKind(t *testing.T) {
	target := recovery.Target{Kind: "unknown"}
	if err := target.Validate(); err == nil {
		t.Error("expected error for unknown target kind")
	}
}

func TestTarget_Validate_Timeline(t *testing.T) {
	target := recovery.Target{
		Kind:     recovery.TargetKindImmediate,
		Timeline: "latest",
	}
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid timeline 'latest', got: %v", err)
	}

	target.Timeline = "3"
	if err := target.Validate(); err != nil {
		t.Errorf("expected valid numeric timeline, got: %v", err)
	}

	target.Timeline = "invalid"
	if err := target.Validate(); err == nil {
		t.Error("expected error for invalid timeline")
	}
}

func TestTarget_RecoveryParams_Time(t *testing.T) {
	ts := pastTime(2 * time.Hour)
	target := recovery.Target{
		Kind:      recovery.TargetKindTime,
		Time:      ts,
		Inclusive: true,
		Timeline:  "latest",
	}
	params := target.RecoveryParams()

	if _, ok := params["recovery_target_time"]; !ok {
		t.Error("expected recovery_target_time param")
	}
	if params["recovery_target_inclusive"] != "true" {
		t.Error("expected recovery_target_inclusive=true")
	}
	if params["recovery_target_timeline"] != "latest" {
		t.Error("expected recovery_target_timeline=latest")
	}
}

func TestTarget_RecoveryParams_LSN(t *testing.T) {
	target := recovery.Target{
		Kind:      recovery.TargetKindLSN,
		LSN:       "0/15D3C28",
		Inclusive: false,
	}
	params := target.RecoveryParams()

	if params["recovery_target_lsn"] != "0/15D3C28" {
		t.Errorf("unexpected recovery_target_lsn: %q", params["recovery_target_lsn"])
	}
	if params["recovery_target_inclusive"] != "false" {
		t.Error("expected recovery_target_inclusive=false")
	}
}

func TestTarget_RecoveryParams_Immediate(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindImmediate}
	params := target.RecoveryParams()

	if params["recovery_target"] != "immediate" {
		t.Errorf("unexpected recovery_target: %q", params["recovery_target"])
	}
}

func TestTarget_RecoveryParams_DefaultTimeline(t *testing.T) {
	target := recovery.Target{Kind: recovery.TargetKindImmediate}
	params := target.RecoveryParams()

	if params["recovery_target_timeline"] != "latest" {
		t.Error("expected default timeline to be 'latest'")
	}
}

func TestParseTime_Valid(t *testing.T) {
	ts, err := recovery.ParseTime("2026-05-01 12:00:00 UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Year() != 2026 || ts.Month() != 5 || ts.Day() != 1 {
		t.Errorf("unexpected parsed time: %v", ts)
	}
}

func TestParseTime_Invalid(t *testing.T) {
	_, err := recovery.ParseTime("not a time")
	if err == nil {
		t.Error("expected error for invalid time string")
	}
}

func TestStandbySignalContent(t *testing.T) {
	content := recovery.StandbySignalContent()
	if content == "" {
		t.Error("standby signal content should not be empty")
	}
}

func TestRecoverySignalContent(t *testing.T) {
	content := recovery.RecoverySignalContent()
	if content == "" {
		t.Error("recovery signal content should not be empty")
	}
}
