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

package network_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/network"
)

const testClusterName = "mypg"

func newCluster() *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testClusterName,
			Namespace: "default",
		},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       3,
			Storage: pgv1alpha1.StorageSpec{
				Size: resource.MustParse("10Gi"),
			},
		},
	}
}

func TestClusterNetworkPolicy_Name(t *testing.T) {
	c := newCluster()
	pol := network.ClusterNetworkPolicy(c)
	if pol.Name != network.PolicyName(testClusterName) {
		t.Errorf("unexpected policy name %q", pol.Name)
	}
	if pol.Namespace != "default" {
		t.Errorf("unexpected namespace %q", pol.Namespace)
	}
}

func TestClusterNetworkPolicy_HasIngressRules(t *testing.T) {
	c := newCluster()
	pol := network.ClusterNetworkPolicy(c)
	if len(pol.Spec.Ingress) == 0 {
		t.Errorf("expected ingress rules, got none")
	}
}

func TestPoolerNetworkPolicy_Name(t *testing.T) {
	c := newCluster()
	pol := network.PoolerNetworkPolicy(c)
	if pol.Name != testClusterName+"-pooler-netpol" {
		t.Errorf("unexpected pooler policy name %q", pol.Name)
	}
}

func TestHeadlessServiceName(t *testing.T) {
	if got := network.HeadlessServiceName("foo"); got != "foo-headless" {
		t.Errorf("HeadlessServiceName(\"foo\") = %q, want foo-headless", got)
	}
}

func TestHeadlessService_HasClusterIPNone(t *testing.T) {
	c := newCluster()
	svc := network.HeadlessService(c)
	if svc.Spec.ClusterIP != "None" {
		t.Errorf("expected ClusterIP=None, got %q", svc.Spec.ClusterIP)
	}
	if !svc.Spec.PublishNotReadyAddresses {
		t.Error("headless service must publish not-ready addresses")
	}
}

func TestReadWriteServiceName(t *testing.T) {
	if got := network.ReadWriteServiceName("foo"); got != "foo-rw" {
		t.Errorf("ReadWriteServiceName(\"foo\") = %q, want foo-rw", got)
	}
}

func TestReadOnlyServiceName(t *testing.T) {
	if got := network.ReadOnlyServiceName("foo"); got != "foo-ro" {
		t.Errorf("ReadOnlyServiceName(\"foo\") = %q, want foo-ro", got)
	}
}

func TestAnyServiceName(t *testing.T) {
	if got := network.AnyServiceName("foo"); got != "foo-any" {
		t.Errorf("AnyServiceName(\"foo\") = %q, want foo-any", got)
	}
}

func TestReadWriteService_SelectsPrimary(t *testing.T) {
	c := newCluster()
	svc := network.ReadWriteService(c)
	if svc.Spec.Selector["pg.vigil.io/role"] != "primary" {
		t.Errorf("read-write service selector should target role=primary, got %v", svc.Spec.Selector)
	}
}

func TestReadOnlyService_SelectsReplica(t *testing.T) {
	c := newCluster()
	svc := network.ReadOnlyService(c)
	if svc.Spec.Selector["pg.vigil.io/role"] != "replica" {
		t.Errorf("read-only service selector should target role=replica, got %v", svc.Spec.Selector)
	}
}

func TestAnyService_SelectsAny(t *testing.T) {
	c := newCluster()
	svc := network.AnyService(c)
	if _, ok := svc.Spec.Selector["pg.vigil.io/cluster"]; !ok {
		t.Errorf("any service selector should include cluster label, got %v", svc.Spec.Selector)
	}
}

func TestServiceFQDN(t *testing.T) {
	got := network.ServiceFQDN("svc", "ns")
	want := "svc.ns.svc.cluster.local"
	if got != want {
		t.Errorf("ServiceFQDN = %q, want %q", got, want)
	}
}

func TestPodFQDN(t *testing.T) {
	got := network.PodFQDN("pod-0", "head", "ns")
	want := "pod-0.head.ns.svc.cluster.local"
	if got != want {
		t.Errorf("PodFQDN = %q, want %q", got, want)
	}
}

func TestReplicationPolicyName(t *testing.T) {
	if got := network.ReplicationPolicyName("foo"); got != "foo-replication-netpol" {
		t.Errorf("ReplicationPolicyName(\"foo\") = %q", got)
	}
}
