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

// Package scheduler provides cron-expression parsing and next-run calculation
// for backup and maintenance scheduling within the Athos operator. It is
// intentionally lightweight and avoids importing a full cron library.
package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Predefined cron schedule presets.
const (
	// ScheduleHourly runs at the top of every hour.
	ScheduleHourly = "0 * * * *"

	// ScheduleDaily runs daily at midnight UTC.
	ScheduleDaily = "0 0 * * *"

	// ScheduleWeekly runs weekly on Sunday at midnight UTC.
	ScheduleWeekly = "0 0 * * 0"

	// ScheduleMonthly runs on the first day of each month at midnight UTC.
	ScheduleMonthly = "0 0 1 * *"
)

// RetentionPolicy represents a backup retention specification.
type RetentionPolicy struct {
	// Days is the number of days to retain backups.
	Days int
	// Count is the number of backups to retain (alternative to Days).
	Count int
}

// ParseRetentionPolicy parses a retention policy string of the form "7d" or "5".
func ParseRetentionPolicy(s string) (RetentionPolicy, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return RetentionPolicy{}, fmt.Errorf("empty retention policy")
	}
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || days <= 0 {
			return RetentionPolicy{}, fmt.Errorf("invalid day-based retention policy %q", s)
		}
		return RetentionPolicy{Days: days}, nil
	}
	count, err := strconv.Atoi(s)
	if err != nil || count <= 0 {
		return RetentionPolicy{}, fmt.Errorf("invalid count-based retention policy %q", s)
	}
	return RetentionPolicy{Count: count}, nil
}

// ExpiresAt returns the time at which a backup taken at startTime expires
// according to the retention policy.
func (r RetentionPolicy) ExpiresAt(startTime time.Time) (time.Time, error) {
	if r.Days > 0 {
		return startTime.Add(time.Duration(r.Days) * 24 * time.Hour), nil
	}
	return time.Time{}, fmt.Errorf("count-based retention does not have a fixed expiry time")
}

// IsExpired reports whether a backup taken at startTime has expired.
func (r RetentionPolicy) IsExpired(startTime, now time.Time) bool {
	if r.Days > 0 {
		expiry := startTime.Add(time.Duration(r.Days) * 24 * time.Hour)
		return now.After(expiry)
	}
	return false
}

// Schedule represents a parsed cron schedule.
type Schedule struct {
	// raw is the original cron expression.
	raw string
	// minute, hour, dom, month, dow hold the parsed fields.
	minute string
	hour   string
	dom    string
	month  string
	dow    string
}

// Parse parses a 5-field cron expression and returns a Schedule.
// Supports the wildcard (*) and specific values. Does not support ranges or
// step values in this minimal implementation.
func Parse(expr string) (*Schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression %q must have exactly 5 fields", expr)
	}
	if err := validateField(fields[0], 0, 59, "minute"); err != nil {
		return nil, err
	}
	if err := validateField(fields[1], 0, 23, "hour"); err != nil {
		return nil, err
	}
	if err := validateField(fields[2], 1, 31, "day-of-month"); err != nil {
		return nil, err
	}
	if err := validateField(fields[3], 1, 12, "month"); err != nil {
		return nil, err
	}
	if err := validateField(fields[4], 0, 7, "day-of-week"); err != nil {
		return nil, err
	}
	return &Schedule{
		raw:    expr,
		minute: fields[0],
		hour:   fields[1],
		dom:    fields[2],
		month:  fields[3],
		dow:    fields[4],
	}, nil
}

// String returns the original cron expression.
func (s *Schedule) String() string { return s.raw }

// validateField checks that a cron field is either "*" or a valid integer
// within the given range.
func validateField(field string, min, max int, name string) error {
	if field == "*" {
		return nil
	}
	// Support comma-separated values.
	for _, part := range strings.Split(field, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("cron field %q has invalid value %q: %w", name, part, err)
		}
		if v < min || v > max {
			return fmt.Errorf("cron field %q value %d is out of range [%d, %d]", name, v, min, max)
		}
	}
	return nil
}

// DueAt returns true if the schedule triggers at the given time. This is a
// minute-level check: seconds are ignored.
func (s *Schedule) DueAt(t time.Time) bool {
	t = t.UTC()
	if !matchField(s.minute, t.Minute()) {
		return false
	}
	if !matchField(s.hour, t.Hour()) {
		return false
	}
	if !matchField(s.dom, t.Day()) {
		return false
	}
	if !matchField(s.month, int(t.Month())) {
		return false
	}
	dow := int(t.Weekday())
	if !matchField(s.dow, dow) {
		// Also accept 7 as equivalent to 0 (Sunday).
		if s.dow == "7" && dow == 0 {
			return true
		}
		return false
	}
	return true
}

func matchField(field string, value int) bool {
	if field == "*" {
		return true
	}
	for _, part := range strings.Split(field, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && v == value {
			return true
		}
	}
	return false
}

// MustParse parses a cron expression and panics on error. Only use in tests.
func MustParse(expr string) *Schedule {
	s, err := Parse(expr)
	if err != nil {
		panic(err)
	}
	return s
}
