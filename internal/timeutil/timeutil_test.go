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

package timeutil_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/timeutil"
)

const (
	hour   = time.Hour
	minute = time.Minute
	second = time.Second
)

func TestParseDuration_Stdlib(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"1h", hour},
		{"1h30m", hour + 30*minute},
		{"45s", 45 * second},
		{"500ms", 500 * time.Millisecond},
	}
	for _, c := range cases {
		got, err := timeutil.ParseDuration(c.in)
		if err != nil {
			t.Errorf("ParseDuration(%q) error = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseDuration_Extensions(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"2d", 2 * timeutil.Day},
		{"1w", timeutil.Week},
		{"1w2d", timeutil.Week + 2*timeutil.Day},
		{"1d12h", timeutil.Day + 12*hour},
		{"0d", 0},
	}
	for _, c := range cases {
		got, err := timeutil.ParseDuration(c.in)
		if err != nil {
			t.Errorf("ParseDuration(%q) error = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseDuration_Errors(t *testing.T) {
	if _, err := timeutil.ParseDuration(""); !errors.Is(err, timeutil.ErrEmptyDuration) {
		t.Errorf("expected ErrEmptyDuration, got %v", err)
	}
	bad := []string{"abc", "10", "d", "1x", "1d2"}
	for _, s := range bad {
		if _, err := timeutil.ParseDuration(s); err == nil {
			t.Errorf("ParseDuration(%q) expected error", s)
		}
	}
}

func TestHumanizeDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{30 * second, "30s"},
		{90 * second, "1m 30s"},
		{hour + 30*minute, "1h 30m"},
		{timeutil.Day + 2*hour, "1d 2h"},
		{timeutil.Week + timeutil.Day, "1w 1d"},
		{-(hour + 30*minute), "-1h 30m"},
		{500 * time.Millisecond, "500ms"},
	}
	for _, c := range cases {
		if got := timeutil.HumanizeDuration(c.in); got != c.want {
			t.Errorf("HumanizeDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEarliestLatest(t *testing.T) {
	a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	c := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	if got := timeutil.Earliest(b, a, c); !got.Equal(a) {
		t.Errorf("Earliest = %v, want %v", got, a)
	}
	if got := timeutil.Latest(b, a, c); !got.Equal(c) {
		t.Errorf("Latest = %v, want %v", got, c)
	}
	if got := timeutil.Earliest(time.Time{}, b); !got.Equal(b) {
		t.Errorf("Earliest skipping zero = %v, want %v", got, b)
	}
	if !timeutil.Earliest().IsZero() {
		t.Error("Earliest of nothing should be zero")
	}
	if !timeutil.Latest().IsZero() {
		t.Error("Latest of nothing should be zero")
	}
}

func TestNextOccurrence(t *testing.T) {
	from := time.Date(2024, 5, 10, 9, 0, 0, 0, time.UTC)
	got := timeutil.NextOccurrence(from, 14, 30)
	want := time.Date(2024, 5, 10, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("NextOccurrence later same day = %v, want %v", got, want)
	}

	from = time.Date(2024, 5, 10, 15, 0, 0, 0, time.UTC)
	got = timeutil.NextOccurrence(from, 9, 0)
	want = time.Date(2024, 5, 11, 9, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("NextOccurrence rollover = %v, want %v", got, want)
	}
}

func TestTruncateToDayHour(t *testing.T) {
	t0 := time.Date(2024, 5, 10, 13, 47, 22, 999, time.UTC)
	if got := timeutil.TruncateToDay(t0); !got.Equal(time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("TruncateToDay = %v", got)
	}
	if got := timeutil.TruncateToHour(t0); !got.Equal(time.Date(2024, 5, 10, 13, 0, 0, 0, time.UTC)) {
		t.Errorf("TruncateToHour = %v", got)
	}
}
