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

// Package duration extends time.ParseDuration with the larger units that
// CRD authors typically reach for: days, weeks. The Kubernetes API server
// accepts ISO 8601 PostgresClusterSpec.RetentionDays values; this package
// is for the cases where a free-form string is the user-facing field.
package duration

import (
	"fmt"
	"strings"
	"time"
)

// Day, Week and Month are the additional units this package recognises.
// They are based on a 24-hour day; calendar arithmetic (e.g. preserving the
// day-of-month when adding a month) is not attempted.
const (
	Day   = 24 * time.Hour
	Week  = 7 * Day
	Month = 30 * Day
)

// Parse extends time.ParseDuration with `d`, `w` and `mo` units. Multiple
// segments may be concatenated, e.g. "2w3d" or "1mo5h".
func Parse(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("duration: empty input")
	}

	var total time.Duration
	rest := s

	for len(rest) > 0 {
		segment, parsed, consumed, err := parseSegment(rest)
		if err != nil {
			return 0, fmt.Errorf("duration: %w", err)
		}
		_ = segment
		total += parsed
		rest = rest[consumed:]
	}
	return total, nil
}

// extendedUnits maps the units this package adds on top of time.ParseDuration
// to their absolute durations.
var extendedUnits = map[string]time.Duration{
	"d":  Day,
	"w":  Week,
	"mo": Month,
}

// parseSegment consumes one <number><unit> segment from the front of rest and
// returns the segment as written, its decoded duration, and the number of
// bytes consumed.
func parseSegment(rest string) (string, time.Duration, int, error) {
	num := 0
	for num < len(rest) && (isDigit(rest[num]) || rest[num] == '-' || rest[num] == '+') {
		num++
	}
	if num == 0 {
		return "", 0, 0, fmt.Errorf("missing number in %q", rest)
	}
	unit := num
	for unit < len(rest) && isLetter(rest[unit]) {
		unit++
	}
	if unit == num {
		return "", 0, 0, fmt.Errorf("missing unit in %q", rest)
	}

	segment := rest[:unit]
	key := strings.ToLower(rest[num:unit])
	if mult, ok := extendedUnits[key]; ok {
		n, err := scanInt(rest[:num])
		if err != nil {
			return "", 0, 0, err
		}
		return segment, time.Duration(n) * mult, unit, nil
	}
	parsed, err := time.ParseDuration(segment)
	if err != nil {
		return "", 0, 0, err
	}
	return segment, parsed, unit, nil
}

// MustParse is like Parse but panics on error. Reserved for tests.
func MustParse(s string) time.Duration {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Format renders d in the most compact form using days/hours/minutes.
// Sub-second precision is dropped.
func Format(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	negative := d < 0
	if negative {
		d = -d
	}
	days := d / Day
	d -= days * Day
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	var parts []string
	if days != 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours != 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes != 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds != 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	out := strings.Join(parts, "")
	if negative {
		out = "-" + out
	}
	return out
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func scanInt(s string) (int64, error) {
	var n int64
	var sign int64 = 1
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("invalid number")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		n = n*10 + int64(c-'0')
	}
	return n * sign, nil
}
