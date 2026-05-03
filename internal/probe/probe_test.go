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

package probe_test

import (
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/probe"
)

func TestLivenessProbe_DefaultsApplied(t *testing.T) {
	p := probe.LivenessProbe(probe.Timing{})
	if p.PeriodSeconds != probe.DefaultPeriodSeconds {
		t.Errorf("PeriodSeconds = %d, want %d", p.PeriodSeconds, probe.DefaultPeriodSeconds)
	}
	if p.InitialDelaySeconds != probe.DefaultInitialDelaySeconds {
		t.Errorf("InitialDelaySeconds = %d, want %d", p.InitialDelaySeconds, probe.DefaultInitialDelaySeconds)
	}
	if p.TimeoutSeconds != probe.DefaultTimeoutSeconds {
		t.Errorf("TimeoutSeconds = %d", p.TimeoutSeconds)
	}
	if p.FailureThreshold != probe.DefaultFailureThreshold {
		t.Errorf("FailureThreshold = %d", p.FailureThreshold)
	}
}

func TestLivenessProbe_OverrideKeptIntact(t *testing.T) {
	p := probe.LivenessProbe(probe.Timing{PeriodSeconds: 30, FailureThreshold: 12})
	if p.PeriodSeconds != 30 {
		t.Errorf("PeriodSeconds override lost: %d", p.PeriodSeconds)
	}
	if p.FailureThreshold != 12 {
		t.Errorf("FailureThreshold override lost: %d", p.FailureThreshold)
	}
}

func TestLivenessProbe_UsesPgIsReady(t *testing.T) {
	p := probe.LivenessProbe(probe.Timing{})
	if p.Exec == nil {
		t.Fatal("expected Exec handler for liveness")
	}
	if p.Exec.Command[0] != "pg_isready" {
		t.Errorf("expected pg_isready, got %q", p.Exec.Command[0])
	}
}

func TestReadinessProbe_UsesPgIsReady(t *testing.T) {
	p := probe.ReadinessProbe(probe.Timing{})
	if p.Exec == nil || p.Exec.Command[0] != "pg_isready" {
		t.Errorf("readiness probe should use pg_isready")
	}
}

func TestStartupProbe_HasMinimumFailureThreshold(t *testing.T) {
	p := probe.StartupProbe(probe.Timing{FailureThreshold: 5})
	if p.FailureThreshold < 30 {
		t.Errorf("startup FailureThreshold = %d, want >=30", p.FailureThreshold)
	}
}

func TestStartupProbe_UsesTCPSocket(t *testing.T) {
	p := probe.StartupProbe(probe.Timing{})
	if p.TCPSocket == nil {
		t.Fatal("expected TCPSocket handler for startup")
	}
	if p.TCPSocket.Port.IntValue() != probe.PostgresPort {
		t.Errorf("TCPSocket port = %d, want %d", p.TCPSocket.Port.IntValue(), probe.PostgresPort)
	}
}

func TestPgBouncerLivenessProbe_PortDefault(t *testing.T) {
	p := probe.PgBouncerLivenessProbe(0)
	if p.TCPSocket.Port.IntValue() != 6432 {
		t.Errorf("default pgbouncer port = %d", p.TCPSocket.Port.IntValue())
	}
}

func TestPgBouncerLivenessProbe_PortOverride(t *testing.T) {
	p := probe.PgBouncerLivenessProbe(7432)
	if p.TCPSocket.Port.IntValue() != 7432 {
		t.Errorf("pgbouncer port override = %d", p.TCPSocket.Port.IntValue())
	}
}

func TestPgBouncerReadinessProbe_AliasesLiveness(t *testing.T) {
	live := probe.PgBouncerLivenessProbe(6432)
	ready := probe.PgBouncerReadinessProbe(6432)
	if live.TCPSocket.Port.IntValue() != ready.TCPSocket.Port.IntValue() {
		t.Errorf("expected pgbouncer readiness to mirror liveness")
	}
}

func TestHTTPGetProbe_ShapesHandler(t *testing.T) {
	p := probe.HTTPGetProbe("/healthz", 8081, probe.Timing{})
	if p.HTTPGet == nil {
		t.Fatal("expected HTTPGet handler")
	}
	if p.HTTPGet.Path != "/healthz" {
		t.Errorf("path = %q", p.HTTPGet.Path)
	}
	if p.HTTPGet.Port.IntValue() != 8081 {
		t.Errorf("port = %d", p.HTTPGet.Port.IntValue())
	}
}
