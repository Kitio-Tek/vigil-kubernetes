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

// Package keepalive computes the TCP keepalive parameters that the manager
// sets on long-lived connections to PostgreSQL instances. Defaults are tuned
// to detect a network partition within roughly two minutes.
package keepalive

import "time"

// Defaults applied when a Config is zero-valued.
const (
	DefaultIdle      = 30 * time.Second
	DefaultInterval  = 30 * time.Second
	DefaultMaxProbes = 3
)

// Config groups the three TCP keepalive knobs into a single struct so they
// are easy to pass through call sites.
type Config struct {
	// Idle is the time the connection must be idle before the OS starts
	// sending keepalive probes.
	Idle time.Duration
	// Interval is the gap between successive keepalive probes.
	Interval time.Duration
	// MaxProbes is the number of unacknowledged probes after which the
	// connection is considered dead.
	MaxProbes int
}

// WithDefaults returns a copy of c with every zero-valued field replaced by
// the package default.
func (c Config) WithDefaults() Config {
	if c.Idle == 0 {
		c.Idle = DefaultIdle
	}
	if c.Interval == 0 {
		c.Interval = DefaultInterval
	}
	if c.MaxProbes == 0 {
		c.MaxProbes = DefaultMaxProbes
	}
	return c
}

// DetectionWindow returns the worst-case time before a dead connection is
// reported. It is Idle + Interval * MaxProbes.
func (c Config) DetectionWindow() time.Duration {
	c = c.WithDefaults()
	return c.Idle + c.Interval*time.Duration(c.MaxProbes)
}

// IsAggressive reports whether the configuration would generate more than
// `threshold` probes per minute on a busy server. The default threshold is
// 30 probes per minute.
func (c Config) IsAggressive() bool {
	c = c.WithDefaults()
	if c.Interval <= 0 {
		return true
	}
	probesPerMinute := int(time.Minute / c.Interval)
	return probesPerMinute > 30
}
