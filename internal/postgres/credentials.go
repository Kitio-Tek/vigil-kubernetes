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

package postgres

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
)

// Credential keys placed inside the operator-managed Secret. The names are
// chosen so a pod can mount the Secret directly as environment variables
// and have the libpq client pick them up without further translation.
const (
	CredentialKeyUsername = "username"
	CredentialKeyPassword = "password"
	CredentialKeyHost     = "host"
	CredentialKeyPort     = "port"
	CredentialKeyDatabase = "database"
	CredentialKeyURI      = "uri"
)

// BuildCredentialSecret returns the Secret that holds the superuser
// credentials for a freshly provisioned PostgresCluster. Existing Secrets
// are not overwritten by the reconciler so a rotated password is preserved
// across operator restarts.
func BuildCredentialSecret(cluster *pgv1alpha1.PostgresCluster, password string) *corev1.Secret {
	host := PrimaryServiceName(cluster)
	port := "5432"
	user := "postgres"
	database := "postgres"
	uri := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, password, host, port, database)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName(cluster),
			Namespace: cluster.Namespace,
			Labels:    CommonLabels(cluster),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			CredentialKeyUsername: user,
			CredentialKeyPassword: password,
			CredentialKeyHost:     host,
			CredentialKeyPort:     port,
			CredentialKeyDatabase: database,
			CredentialKeyURI:      uri,
		},
	}
}
