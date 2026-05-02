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

package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/metrics"
)

func TestRecordReconcile(t *testing.T) {
	before := testutil.ToFloat64(metrics.ReconcileTotal.WithLabelValues("postgrescluster", "success"))
	metrics.RecordReconcile("postgrescluster", "success")
	after := testutil.ToFloat64(metrics.ReconcileTotal.WithLabelValues("postgrescluster", "success"))
	if after-before != 1 {
		t.Errorf("expected counter to increment by 1, got %v", after-before)
	}
}

func TestRecordBackup(t *testing.T) {
	before := testutil.ToFloat64(metrics.BackupsTotal.WithLabelValues("mycluster", "default", "completed"))
	metrics.RecordBackup("mycluster", "default", "completed")
	after := testutil.ToFloat64(metrics.BackupsTotal.WithLabelValues("mycluster", "default", "completed"))
	if after-before != 1 {
		t.Errorf("expected backup counter to increment by 1, got %v", after-before)
	}
}

func TestRecordUpgrade(t *testing.T) {
	before := testutil.ToFloat64(metrics.UpgradesTotal.WithLabelValues("mycluster", "default", "completed"))
	metrics.RecordUpgrade("mycluster", "default", "completed")
	after := testutil.ToFloat64(metrics.UpgradesTotal.WithLabelValues("mycluster", "default", "completed"))
	if after-before != 1 {
		t.Errorf("expected upgrade counter to increment by 1, got %v", after-before)
	}
}

func TestRecordFailover(t *testing.T) {
	before := testutil.ToFloat64(metrics.FailoversTotal.WithLabelValues("mycluster", "default", "automatic"))
	metrics.RecordFailover("mycluster", "default", "automatic")
	after := testutil.ToFloat64(metrics.FailoversTotal.WithLabelValues("mycluster", "default", "automatic"))
	if after-before != 1 {
		t.Errorf("expected failover counter to increment by 1, got %v", after-before)
	}
}

func TestSetInstancesReady(t *testing.T) {
	metrics.SetInstancesReady("mycluster", "default", 3)
	val := testutil.ToFloat64(metrics.InstancesReady.WithLabelValues("mycluster", "default"))
	if val != 3 {
		t.Errorf("expected instances ready gauge to be 3, got %v", val)
	}
}

func TestSetClustersPhase(t *testing.T) {
	metrics.SetClustersPhase("Running", 5)
	val := testutil.ToFloat64(metrics.ClustersTotal.WithLabelValues("Running"))
	if val != 5 {
		t.Errorf("expected cluster phase gauge to be 5, got %v", val)
	}
}
