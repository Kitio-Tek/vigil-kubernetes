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

// Package dnsname constructs the in-cluster DNS names used by Athos
// PostgresClusters. The naming follows the standard Kubernetes Service and
// StatefulSet headless DNS layout described in
// https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/.
package dnsname

import "strings"

// ClusterDomain is the suffix appended to in-cluster Service FQDNs. Most
// installations use the default "cluster.local" but some change it; the
// constant is configurable via SetClusterDomain.
var clusterDomain = "cluster.local"

// SetClusterDomain overrides the cluster domain used by FQDN helpers.
// Useful in test environments and on managed clusters.
func SetClusterDomain(domain string) {
	if domain == "" {
		clusterDomain = "cluster.local"
		return
	}
	clusterDomain = strings.TrimSuffix(domain, ".")
}

// ClusterDomain returns the currently configured cluster DNS domain.
func ClusterDomain() string { return clusterDomain }

// ServiceFQDN returns the fully qualified DNS name of a Service.
func ServiceFQDN(svc, namespace string) string {
	return svc + "." + namespace + ".svc." + clusterDomain
}

// PodFQDN returns the fully qualified DNS name of a Pod that participates
// in a headless Service.
func PodFQDN(pod, headlessSvc, namespace string) string {
	return pod + "." + headlessSvc + "." + namespace + ".svc." + clusterDomain
}

// PrimaryFQDN returns the fully qualified DNS name of the read/write
// service for the given cluster.
func PrimaryFQDN(cluster, namespace string) string {
	return ServiceFQDN(cluster+"-rw", namespace)
}

// ReplicaFQDN returns the fully qualified DNS name of the replicas service.
func ReplicaFQDN(cluster, namespace string) string {
	return ServiceFQDN(cluster+"-ro", namespace)
}

// AnyFQDN returns the fully qualified DNS name of the any-instance service.
func AnyFQDN(cluster, namespace string) string {
	return ServiceFQDN(cluster+"-any", namespace)
}

// HeadlessFQDN returns the fully qualified DNS name of the headless service.
func HeadlessFQDN(cluster, namespace string) string {
	return ServiceFQDN(cluster+"-headless", namespace)
}

// SearchPath returns the standard kubernetes pod resolver search list for
// the given namespace.
func SearchPath(namespace string) []string {
	return []string{
		namespace + ".svc." + clusterDomain,
		"svc." + clusterDomain,
		clusterDomain,
	}
}
