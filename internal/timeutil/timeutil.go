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

// Package timeutil provides small helpers for parsing, formatting and
// rounding time.Duration and time.Time values used throughout the
// operator. It extends the standard library to recognise human-friendly
// units (such as "d" for day and "w" for week) without taking on any
// third-party dependencies.
package timeutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Day is the duration of a calendar day used by ParseDuration.
const Day = 24 * time.Hour

// Week is the duration of a calendar week used by ParseDuration.
const Week = 7 * Day

// ErrEmptyDuration is returned by ParseDuration when the input is empty.
var ErrEmptyDuration = errors.New("timeutil: empty duration string")

// ErrInvalidDuration is returned by ParseDuration when the input cannot be
// understood as a duration expression.
var ErrInvalidDuration = errors.New("timeutil: invalid duration")

// ParseDuration parses a duration string, extending time.ParseDuration with
// support for "d" (day) and "w" (week) suffixes. Compound expressions such
// as "1w2d3h" are accepted; segments are summed in left-to-right order.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, ErrEmptyDuration
	}
	// Fast path: stdlib already understands the input.
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	var total time.Duration
	var numStart int
	var sawAny bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || c == '.' || (c == '-' && i == numStart) {
			continue
		}
		// We hit a unit character; collect it (single or multi-char).
		unitEnd := i
		for unitEnd < len(s) {
			b := s[unitEnd]
			if (b >= '0' && b <= '9') || b == '.' || b == '-' {
				break
			}
			unitEnd++
		}
		num := s[numStart:i]
		unit := s[i:unitEnd]
		if num == "" {
			return 0, fmt.Errorf("%w: missing number before %q", ErrInvalidDuration, unit)
		}
		seg, err := parseSegment(num, unit)
		if err != nil {
			return 0, err
		}
		total += seg
		sawAny = true
		numStart = unitEnd
		i = unitEnd - 1
	}
	if !sawAny || numStart != len(s) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, s)
	}
	return total, nil
}

func parseSegment(num, unit string) (time.Duration, error) {
	switch unit {
	case "d":
		v, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: bad number %q", ErrInvalidDuration, num)
		}
		return time.Duration(v * float64(Day)), nil
	case "w":
		v, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: bad number %q", ErrInvalidDuration, num)
		}
		return time.Duration(v * float64(Week)), nil
	default:
		// Defer to stdlib for ns/us/ms/s/m/h.
		d, err := time.ParseDuration(num + unit)
		if err != nil {
			return 0, fmt.Errorf("%w: %s%s: %v", ErrInvalidDuration, num, unit, err)
		}
		return d, nil
	}
}

// HumanizeDuration renders a duration as a compact, space-separated string
// using the largest applicable units down to seconds. Sub-second values
// fall back to time.Duration's native formatting. Negative durations are
// returned with a leading minus sign.
func HumanizeDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	negative := d < 0
	if negative {
		d = -d
	}
	if d < time.Second {
		out := d.String()
		if negative {
			return "-" + out
		}
		return out
	}

	var parts []string
	if d >= Week {
		w := d / Week
		parts = append(parts, fmt.Sprintf("%dw", w))
		d -= w * Week
	}
	if d >= Day {
		v := d / Day
		parts = append(parts, fmt.Sprintf("%dd", v))
		d -= v * Day
	}
	if d >= time.Hour {
		v := d / time.Hour
		parts = append(parts, fmt.Sprintf("%dh", v))
		d -= v * time.Hour
	}
	if d >= time.Minute {
		v := d / time.Minute
		parts = append(parts, fmt.Sprintf("%dm", v))
		d -= v * time.Minute
	}
	if d >= time.Second {
		v := d / time.Second
		parts = append(parts, fmt.Sprintf("%ds", v))
	}
	out := strings.Join(parts, " ")
	if negative {
		return "-" + out
	}
	return out
}

// Earliest returns the earliest non-zero time among the arguments. If no
// argument is non-zero, the zero value of time.Time is returned.
func Earliest(times ...time.Time) time.Time {
	var best time.Time
	for _, t := range times {
		if t.IsZero() {
			continue
		}
		if best.IsZero() || t.Before(best) {
			best = t
		}
	}
	return best
}

// Latest returns the latest non-zero time among the arguments. If no
// argument is non-zero, the zero value of time.Time is returned.
func Latest(times ...time.Time) time.Time {
	var best time.Time
	for _, t := range times {
		if t.IsZero() {
			continue
		}
		if best.IsZero() || t.After(best) {
			best = t
		}
	}
	return best
}

// NextOccurrence returns the next time at or after `from` whose hour and
// minute match the requested values, preserving the location of `from`.
// When the requested wall-clock time has already passed for the day, the
// result is rolled forward by 24 hours. Out-of-range hour/minute values
// are normalised by time.Date.
func NextOccurrence(from time.Time, hour, minute int) time.Time {
	loc := from.Location()
	candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, loc)
	if !candidate.After(from) {
		candidate = candidate.Add(Day)
	}
	return candidate
}

// TruncateToDay returns t rounded down to midnight in t's location.
func TruncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// TruncateToHour returns t rounded down to the start of its hour in t's
// location.
func TruncateToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}
