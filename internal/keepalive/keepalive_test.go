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

package keepalive_test

import (
	"testing"
	"time"

	"github.com/Kitio-Tek/athos-kubernetes/internal/keepalive"
)

func TestWithDefaults_FillsZeroFields(t *testing.T) {
	got := (keepalive.Config{}).WithDefaults()
	if got.Idle != keepalive.DefaultIdle {
		t.Errorf("Idle = %v", got.Idle)
	}
	if got.Interval != keepalive.DefaultInterval {
		t.Errorf("Interval = %v", got.Interval)
	}
	if got.MaxProbes != keepalive.DefaultMaxProbes {
		t.Errorf("MaxProbes = %v", got.MaxProbes)
	}
}

func TestWithDefaults_PreservesNonZero(t *testing.T) {
	c := keepalive.Config{Idle: time.Minute, Interval: time.Second, MaxProbes: 7}
	got := c.WithDefaults()
	if got != c {
		t.Errorf("WithDefaults mutated populated config: %+v", got)
	}
}

func TestDetectionWindow(t *testing.T) {
	c := keepalive.Config{Idle: 10 * time.Second, Interval: time.Second, MaxProbes: 5}
	got := c.DetectionWindow()
	want := 15 * time.Second
	if got != want {
		t.Errorf("DetectionWindow = %v, want %v", got, want)
	}
}

func TestDetectionWindow_DefaultIs120s(t *testing.T) {
	got := (keepalive.Config{}).DetectionWindow()
	want := keepalive.DefaultIdle + keepalive.DefaultInterval*time.Duration(keepalive.DefaultMaxProbes)
	if got != want {
		t.Errorf("DetectionWindow default = %v, want %v", got, want)
	}
}

func TestIsAggressive(t *testing.T) {
	if !(keepalive.Config{Interval: time.Second, MaxProbes: 1}).IsAggressive() {
		t.Error("1s interval should be classified aggressive")
	}
	if (keepalive.Config{Interval: 30 * time.Second, MaxProbes: 1}).IsAggressive() {
		t.Error("30s interval should not be aggressive")
	}
}

func TestIsAggressive_NegativeInterval(t *testing.T) {
	c := keepalive.Config{Idle: time.Second, Interval: -1, MaxProbes: 1}
	if !c.IsAggressive() {
		t.Error("negative interval should be aggressive")
	}
}
