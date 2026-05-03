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

package scheduler_test

import (
	"testing"
	"time"

	"github.com/Kitio-Tek/athos-kubernetes/internal/scheduler"
)

func TestParse_ValidExpressions(t *testing.T) {
	exprs := []string{
		"0 * * * *",
		"0 0 * * *",
		"30 6 * * 1",
		"0 0 1 * *",
		"0 0 * * 0",
		scheduler.ScheduleHourly,
		scheduler.ScheduleDaily,
		scheduler.ScheduleWeekly,
		scheduler.ScheduleMonthly,
	}
	for _, e := range exprs {
		_, err := scheduler.Parse(e)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", e, err)
		}
	}
}

func TestParse_InvalidExpressions(t *testing.T) {
	exprs := []string{
		"",
		"* * * *",
		"* * * * * *",
		"60 * * * *",
		"* 24 * * *",
		"* * 0 * *",
		"* * * 13 *",
		"* * * * 8",
		"abc * * * *",
	}
	for _, e := range exprs {
		_, err := scheduler.Parse(e)
		if err == nil {
			t.Errorf("Parse(%q) should return error for invalid expression", e)
		}
	}
}

func TestSchedule_DueAt_Daily(t *testing.T) {
	s := scheduler.MustParse(scheduler.ScheduleDaily)

	midnight := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !s.DueAt(midnight) {
		t.Error("ScheduleDaily should be due at midnight UTC")
	}

	oneAM := time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC)
	if s.DueAt(oneAM) {
		t.Error("ScheduleDaily should not be due at 01:00")
	}
}

func TestSchedule_DueAt_Hourly(t *testing.T) {
	s := scheduler.MustParse(scheduler.ScheduleHourly)

	topOfHour := time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC)
	if !s.DueAt(topOfHour) {
		t.Error("ScheduleHourly should be due at the top of the hour")
	}

	halfHour := time.Date(2026, 5, 1, 15, 30, 0, 0, time.UTC)
	if s.DueAt(halfHour) {
		t.Error("ScheduleHourly should not be due at half-past")
	}
}

func TestSchedule_DueAt_Weekly(t *testing.T) {
	s := scheduler.MustParse(scheduler.ScheduleWeekly)

	// 2026-05-03 is a Sunday
	sunday := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	if !s.DueAt(sunday) {
		t.Error("ScheduleWeekly should be due on Sunday midnight")
	}

	// 2026-05-04 is a Monday
	monday := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	if s.DueAt(monday) {
		t.Error("ScheduleWeekly should not be due on Monday")
	}
}

func TestSchedule_DueAt_Monthly(t *testing.T) {
	s := scheduler.MustParse(scheduler.ScheduleMonthly)

	firstDay := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !s.DueAt(firstDay) {
		t.Error("ScheduleMonthly should be due on the first of the month at midnight")
	}

	secondDay := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	if s.DueAt(secondDay) {
		t.Error("ScheduleMonthly should not be due on the second")
	}
}

func TestSchedule_String(t *testing.T) {
	expr := "30 4 * * 1"
	s := scheduler.MustParse(expr)
	if s.String() != expr {
		t.Errorf("String() = %q, want %q", s.String(), expr)
	}
}

func TestParseRetentionPolicy_Days(t *testing.T) {
	rp, err := scheduler.ParseRetentionPolicy("7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rp.Days != 7 {
		t.Errorf("expected 7 days, got %d", rp.Days)
	}
}

func TestParseRetentionPolicy_Count(t *testing.T) {
	rp, err := scheduler.ParseRetentionPolicy("5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rp.Count != 5 {
		t.Errorf("expected count 5, got %d", rp.Count)
	}
}

func TestParseRetentionPolicy_Invalid(t *testing.T) {
	invalid := []string{"", "0d", "-1d", "0", "abc", "abc d"}
	for _, s := range invalid {
		_, err := scheduler.ParseRetentionPolicy(s)
		if err == nil {
			t.Errorf("expected error for invalid retention policy %q", s)
		}
	}
}

func TestRetentionPolicy_IsExpired(t *testing.T) {
	rp, _ := scheduler.ParseRetentionPolicy("7d")
	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	old := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !rp.IsExpired(old, now) {
		t.Error("9-day old backup should be expired with 7d retention")
	}

	recent := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	if rp.IsExpired(recent, now) {
		t.Error("2-day old backup should not be expired with 7d retention")
	}
}

func TestRetentionPolicy_ExpiresAt(t *testing.T) {
	rp, _ := scheduler.ParseRetentionPolicy("7d")
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	expiry, err := rp.ExpiresAt(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	if !expiry.Equal(expected) {
		t.Errorf("expected expiry %v, got %v", expected, expiry)
	}
}

func TestRetentionPolicy_ExpiresAt_CountBased(t *testing.T) {
	rp, _ := scheduler.ParseRetentionPolicy("5")
	_, err := rp.ExpiresAt(time.Now())
	if err == nil {
		t.Error("count-based retention should not have a fixed expiry")
	}
}
