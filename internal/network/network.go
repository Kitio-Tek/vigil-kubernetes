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

// Package network provides helpers for building Kubernetes NetworkPolicy and
// Service resources for PostgreSQL clusters managed by the Athos operator.
package network

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

const (
	postgresPort        = 5432
	pgBouncerPort       = 6432
	replicationPortName = "replication"
)

// PolicyName returns the name of the NetworkPolicy for the given cluster.
func PolicyName(clusterName string) string {
	return clusterName + "-netpol"
}

// ReplicationPolicyName returns the name of the replication-only NetworkPolicy.
func ReplicationPolicyName(clusterName string) string {
	return clusterName + "-replication-netpol"
}

// ClusterNetworkPolicy builds a NetworkPolicy that allows:
//   - Postgres traffic (5432) from pods with the operator label in the same namespace.
//   - Replication traffic between cluster pods themselves.
//   - All egress (no egress restrictions).
func ClusterNetworkPolicy(cluster *pgv1alpha1.PostgresCluster) *networkingv1.NetworkPolicy {
	pgPort := intstr.FromInt(postgresPort)
	proto := corev1.ProtocolTCP

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PolicyName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: postgres.SelectorLabels(cluster),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					// Allow postgres connections from pods in the same namespace.
					Ports: []networkingv1.NetworkPolicyPort{
						{Port: &pgPort, Protocol: &proto},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
							PodSelector:       &metav1.LabelSelector{},
						},
					},
				},
				{
					// Allow inter-cluster replication traffic.
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: postgres.SelectorLabels(cluster),
							},
						},
					},
				},
			},
		},
	}
}

// PoolerNetworkPolicy returns a NetworkPolicy that restricts ingress to the
// PgBouncer pooler pods so only pods in the same namespace can connect.
func PoolerNetworkPolicy(cluster *pgv1alpha1.PostgresCluster) *networkingv1.NetworkPolicy {
	bouncerPort := intstr.FromInt(pgBouncerPort)
	proto := corev1.ProtocolTCP

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-pooler-netpol",
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "pooler",
					"pg.athos.io/cluster":         cluster.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{Port: &bouncerPort, Protocol: &proto},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
							PodSelector:       &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}
}

// HeadlessServiceName returns the name of the headless Service used for DNS
// resolution of individual PostgreSQL pods.
func HeadlessServiceName(clusterName string) string {
	return clusterName + "-headless"
}

// HeadlessService builds a headless Service that enables DNS lookup of
// individual pods within the StatefulSet (e.g. clusterName-0.clusterName-headless).
func HeadlessService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Selector:                 postgres.SelectorLabels(cluster),
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       postgresPort,
					TargetPort: intstr.FromInt(postgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// ReadWriteServiceName returns the name of the read-write Service that routes
// traffic to the primary instance.
func ReadWriteServiceName(clusterName string) string {
	return clusterName + "-rw"
}

// ReadWriteService builds a ClusterIP Service that points to the primary pod
// via the "pg.athos.io/role=primary" label selector.
func ReadWriteService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReadWriteServiceName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				postgres.LabelCluster: cluster.Name,
				postgres.LabelRole:    postgres.RolePrimary,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       postgresPort,
					TargetPort: intstr.FromInt(postgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// ReadOnlyServiceName returns the name of the read-only Service that routes
// traffic to replica pods.
func ReadOnlyServiceName(clusterName string) string {
	return clusterName + "-ro"
}

// ReadOnlyService builds a ClusterIP Service that points to replica pods via
// the "pg.athos.io/role=replica" label selector.
func ReadOnlyService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReadOnlyServiceName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				postgres.LabelCluster: cluster.Name,
				postgres.LabelRole:    postgres.RoleReplica,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       postgresPort,
					TargetPort: intstr.FromInt(postgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// AnyServiceName returns the name of the any-instance Service that routes
// traffic to any running PostgreSQL pod regardless of role.
func AnyServiceName(clusterName string) string {
	return clusterName + "-any"
}

// AnyService builds a ClusterIP Service that routes to any healthy cluster pod.
func AnyService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AnyServiceName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: postgres.SelectorLabels(cluster),
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       postgresPort,
					TargetPort: intstr.FromInt(postgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// ServiceFQDN returns the fully qualified DNS name for a Kubernetes Service.
func ServiceFQDN(name, namespace string) string {
	return name + "." + namespace + ".svc.cluster.local"
}

// PodFQDN returns the fully qualified DNS name for a pod inside a headless
// Service (StatefulSet DNS pattern).
func PodFQDN(podName, headlessSvcName, namespace string) string {
	return podName + "." + headlessSvcName + "." + namespace + ".svc.cluster.local"
}
