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

// Package backoff provides exponential and capped backoff schedules used by
// the operator's retry helpers. The package is small on purpose: more
// elaborate schedules (jitter, decorrelated, etc.) live in
// k8s.io/apimachinery/pkg/util/wait — this package is for the cases where a
// dependency on apimachinery's wait machinery is overkill.
package backoff

import (
	"errors"
	"time"
)

// Schedule produces successive delays. Returning ok=false signals no more
// retries should be attempted.
type Schedule interface {
	Next() (delay time.Duration, ok bool)
	// Reset clears any internal state so the schedule can be re-used.
	Reset()
}

// Exponential is a configurable exponential schedule.
type Exponential struct {
	// Initial is the first delay returned by Next.
	Initial time.Duration
	// Factor is the multiplier applied to each successive delay.
	Factor float64
	// Cap is the maximum delay; once reached, every subsequent Next call
	// returns this value (until MaxRetries, if non-zero, is exceeded).
	Cap time.Duration
	// MaxRetries, if non-zero, limits the total number of Next calls before
	// ok becomes false.
	MaxRetries int

	current time.Duration
	count   int
}

// Next implements Schedule.
func (e *Exponential) Next() (time.Duration, bool) {
	if e.MaxRetries > 0 && e.count >= e.MaxRetries {
		return 0, false
	}
	if e.current == 0 {
		e.current = e.Initial
		if e.current == 0 {
			e.current = time.Second
		}
	} else {
		next := time.Duration(float64(e.current) * e.factor())
		if e.Cap > 0 && next > e.Cap {
			next = e.Cap
		}
		e.current = next
	}
	e.count++
	return e.current, true
}

// Reset clears the schedule state.
func (e *Exponential) Reset() {
	e.current = 0
	e.count = 0
}

func (e *Exponential) factor() float64 {
	if e.Factor <= 0 {
		return 2.0
	}
	return e.Factor
}

// Constant returns a Schedule that always emits the same delay. MaxRetries,
// if positive, bounds the number of emissions.
type Constant struct {
	Delay      time.Duration
	MaxRetries int

	count int
}

// Next implements Schedule.
func (c *Constant) Next() (time.Duration, bool) {
	if c.MaxRetries > 0 && c.count >= c.MaxRetries {
		return 0, false
	}
	c.count++
	return c.Delay, true
}

// Reset clears the schedule state.
func (c *Constant) Reset() { c.count = 0 }

// ErrAborted is returned by Retry when the operation context is cancelled.
var ErrAborted = errors.New("backoff: aborted")

// Retry calls op until it returns nil or the schedule is exhausted. If
// sleep is nil, time.Sleep is used. The function returns the last error
// from op when retries are exhausted.
func Retry(s Schedule, sleep func(time.Duration), op func() error) error {
	if sleep == nil {
		sleep = time.Sleep
	}
	if s == nil {
		return op()
	}
	s.Reset()
	var last error
	for {
		err := op()
		if err == nil {
			return nil
		}
		last = err
		d, ok := s.Next()
		if !ok {
			return last
		}
		sleep(d)
	}
}
