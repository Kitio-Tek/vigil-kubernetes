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

// Package healthcheck contains primitives used by the operator manager and
// per-cluster controllers to evaluate the liveness and readiness of PostgreSQL
// instances.
//
// The package intentionally does not perform any I/O of its own; checks are
// expressed as small structs that the caller assembles into a sequence and
// drives. This keeps the package easy to unit test without spinning up a
// kubernetes API server or PostgreSQL instance.
package healthcheck

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Status represents the outcome of a single health check.
type Status string

const (
	// StatusPassing indicates the check observed a healthy state.
	StatusPassing Status = "Passing"
	// StatusFailing indicates the check observed an unhealthy state.
	StatusFailing Status = "Failing"
	// StatusUnknown indicates the check could not establish a state.
	StatusUnknown Status = "Unknown"
)

// Check captures a single point-in-time observation about an instance.
// Checks are produced by Probe implementations and aggregated into a Report.
type Check struct {
	// Name is a stable identifier for the check, e.g. "wal-replay".
	Name string
	// Status is the outcome of the check.
	Status Status
	// Message is a human-readable description of the outcome.
	Message string
	// ObservedAt records when the check was performed.
	ObservedAt time.Time
	// Duration records how long the underlying probe took.
	Duration time.Duration
	// Critical, when true, marks the check as one whose failure must mark
	// the instance as failing rather than degraded.
	Critical bool
}

// Equal reports whether two checks describe the same observation. ObservedAt
// and Duration are excluded so callers can compare checks across time.
func (c Check) Equal(other Check) bool {
	return c.Name == other.Name &&
		c.Status == other.Status &&
		c.Message == other.Message &&
		c.Critical == other.Critical
}

// Probe is implemented by anything that can observe a single aspect of a
// PostgreSQL instance and report a Check.
type Probe interface {
	// Name returns the stable identifier for the check.
	Name() string
	// Run performs the observation and returns a Check.
	Run() Check
}

// FuncProbe adapts a closure into a Probe. It is the most convenient way to
// write a one-off probe in a unit test.
type FuncProbe struct {
	ProbeName string
	Fn        func() Check
}

// Name returns the probe name.
func (p FuncProbe) Name() string { return p.ProbeName }

// Run executes the closure.
func (p FuncProbe) Run() Check {
	if p.Fn == nil {
		return Check{Name: p.ProbeName, Status: StatusUnknown, Message: "nil func"}
	}
	c := p.Fn()
	if c.Name == "" {
		c.Name = p.ProbeName
	}
	return c
}

// Report aggregates the result of running a sequence of probes against an
// instance. Reports are returned by Run and are safe to log directly.
type Report struct {
	// Instance is a free-form identifier of what was probed (e.g. a pod name).
	Instance string
	// Checks lists the individual check outcomes in the order they were run.
	Checks []Check
}

// Add appends a check to the report.
func (r *Report) Add(c Check) {
	if c.ObservedAt.IsZero() {
		c.ObservedAt = time.Now()
	}
	r.Checks = append(r.Checks, c)
}

// Status reduces a report's checks to a single Status. The rules are:
//   - any critical Failing check marks the report Failing,
//   - otherwise any Failing check marks the report Failing only if every
//     Critical check is Passing,
//   - otherwise any Unknown check marks the report Unknown,
//   - otherwise Passing.
func (r Report) Status() Status {
	if len(r.Checks) == 0 {
		return StatusUnknown
	}
	for _, c := range r.Checks {
		if c.Critical && c.Status == StatusFailing {
			return StatusFailing
		}
	}
	hasFailing := false
	hasUnknown := false
	for _, c := range r.Checks {
		switch c.Status {
		case StatusFailing:
			hasFailing = true
		case StatusUnknown:
			hasUnknown = true
		}
	}
	if hasFailing {
		return StatusFailing
	}
	if hasUnknown {
		return StatusUnknown
	}
	return StatusPassing
}

// SortedChecks returns the checks sorted alphabetically by name. Callers that
// want a stable display order should use this rather than ranging over Checks.
func (r Report) SortedChecks() []Check {
	out := make([]Check, len(r.Checks))
	copy(out, r.Checks)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Summary returns a short human-readable summary of the report. Useful when
// recording a Kubernetes Event message.
func (r Report) Summary() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("instance=%s", r.Instance))
	parts = append(parts, fmt.Sprintf("status=%s", r.Status()))
	parts = append(parts, fmt.Sprintf("checks=%d", len(r.Checks)))
	for _, c := range r.SortedChecks() {
		if c.Status != StatusPassing {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Status))
		}
	}
	return strings.Join(parts, " ")
}

// Run executes every probe in order and returns a Report. If runner is nil,
// probes are run sequentially in the calling goroutine.
func Run(instance string, probes ...Probe) Report {
	report := Report{Instance: instance}
	for _, p := range probes {
		if p == nil {
			continue
		}
		start := time.Now()
		c := p.Run()
		c.Duration = time.Since(start)
		if c.ObservedAt.IsZero() {
			c.ObservedAt = start
		}
		report.Add(c)
	}
	return report
}

// ErrProbeAborted is returned when a probe is cancelled externally before it
// could complete.
var ErrProbeAborted = errors.New("healthcheck: probe aborted")

// SimpleCheck is a convenience constructor for a passing or failing Check.
func SimpleCheck(name, message string, ok bool, critical bool) Check {
	st := StatusPassing
	if !ok {
		st = StatusFailing
	}
	return Check{
		Name:     name,
		Status:   st,
		Message:  message,
		Critical: critical,
	}
}

// CombineReports merges multiple reports into a single one. The instance of
// the returned report is set to instance.
func CombineReports(instance string, reports ...Report) Report {
	out := Report{Instance: instance}
	for _, r := range reports {
		out.Checks = append(out.Checks, r.Checks...)
	}
	return out
}

// Filter returns a new report containing only the checks for which keep
// returns true.
func (r Report) Filter(keep func(Check) bool) Report {
	out := Report{Instance: r.Instance}
	for _, c := range r.Checks {
		if keep(c) {
			out.Checks = append(out.Checks, c)
		}
	}
	return out
}

// FailingChecks returns just the failing checks of a report.
func (r Report) FailingChecks() []Check {
	out := []Check{}
	for _, c := range r.Checks {
		if c.Status == StatusFailing {
			out = append(out, c)
		}
	}
	return out
}
