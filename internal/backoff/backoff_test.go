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

package backoff_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kitio-Tek/athos-kubernetes/internal/backoff"
)

func TestExponential_Doubles(t *testing.T) {
	s := &backoff.Exponential{Initial: 100 * time.Millisecond, Factor: 2}
	want := []time.Duration{100, 200, 400, 800, 1600}
	for i, w := range want {
		d, ok := s.Next()
		if !ok {
			t.Fatalf("Next %d: ok=false", i)
		}
		if d != w*time.Millisecond {
			t.Errorf("step %d delay = %v, want %dms", i, d, w)
		}
	}
}

func TestExponential_Cap(t *testing.T) {
	s := &backoff.Exponential{Initial: 100 * time.Millisecond, Factor: 2, Cap: 300 * time.Millisecond}
	want := []time.Duration{100, 200, 300, 300}
	for i, w := range want {
		d, _ := s.Next()
		if d != w*time.Millisecond {
			t.Errorf("step %d delay = %v, want %dms", i, d, w)
		}
	}
}

func TestExponential_MaxRetries(t *testing.T) {
	s := &backoff.Exponential{Initial: time.Millisecond, MaxRetries: 3}
	for i := 0; i < 3; i++ {
		if _, ok := s.Next(); !ok {
			t.Fatalf("step %d unexpectedly aborted", i)
		}
	}
	if _, ok := s.Next(); ok {
		t.Error("expected ok=false after MaxRetries")
	}
}

func TestExponential_FactorDefaultsToTwo(t *testing.T) {
	s := &backoff.Exponential{Initial: time.Millisecond}
	d1, _ := s.Next()
	d2, _ := s.Next()
	if d2 != 2*d1 {
		t.Errorf("expected default factor of 2, got %v -> %v", d1, d2)
	}
}

func TestExponential_Reset(t *testing.T) {
	s := &backoff.Exponential{Initial: time.Millisecond}
	_, _ = s.Next()
	_, _ = s.Next()
	s.Reset()
	d, _ := s.Next()
	if d != time.Millisecond {
		t.Errorf("post-reset delay = %v, want %v", d, time.Millisecond)
	}
}

func TestConstant(t *testing.T) {
	s := &backoff.Constant{Delay: 5 * time.Millisecond, MaxRetries: 2}
	d, ok := s.Next()
	if !ok || d != 5*time.Millisecond {
		t.Errorf("first = %v ok=%v", d, ok)
	}
	d, ok = s.Next()
	if !ok || d != 5*time.Millisecond {
		t.Errorf("second = %v ok=%v", d, ok)
	}
	if _, ok = s.Next(); ok {
		t.Error("third call should be aborted")
	}
}

func TestRetry_SuccessFirstTry(t *testing.T) {
	err := backoff.Retry(&backoff.Constant{Delay: time.Millisecond, MaxRetries: 3}, func(time.Duration) {}, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestRetry_RecoversAfterRetry(t *testing.T) {
	calls := 0
	err := backoff.Retry(&backoff.Constant{Delay: time.Millisecond, MaxRetries: 5}, func(time.Duration) {}, func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetry_ExhaustsAndReturnsError(t *testing.T) {
	called := 0
	op := func() error { called++; return errors.New("boom") }
	err := backoff.Retry(&backoff.Constant{Delay: time.Millisecond, MaxRetries: 2}, func(time.Duration) {}, op)
	if err == nil {
		t.Error("expected error after retries exhausted")
	}
	// Initial call + 2 retries = 3 invocations.
	if called != 3 {
		t.Errorf("called = %d, want 3", called)
	}
}

func TestRetry_NilSchedule(t *testing.T) {
	calls := 0
	err := backoff.Retry(nil, nil, func() error { calls++; return errors.New("once") })
	if err == nil {
		t.Error("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}
