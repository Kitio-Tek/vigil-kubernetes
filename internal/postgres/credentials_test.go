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

package postgres_test

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

func newCredsCluster() *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "pg", Namespace: "default"},
		Spec: pgv1alpha1.PostgresClusterSpec{
			PostgresVersion: 16,
			Instances:       1,
			Storage:         pgv1alpha1.StorageSpec{Size: resource.MustParse("1Gi")},
		},
	}
}

func TestBuildCredentialSecret_NameAndNamespace(t *testing.T) {
	s := postgres.BuildCredentialSecret(newCredsCluster(), "pw")
	if s.Name != postgres.SecretName(newCredsCluster()) {
		t.Errorf("name = %q", s.Name)
	}
	if s.Namespace != "default" {
		t.Errorf("namespace = %q", s.Namespace)
	}
	if s.Type != corev1.SecretTypeOpaque {
		t.Errorf("type = %q", s.Type)
	}
}

func TestBuildCredentialSecret_AllKeysPresent(t *testing.T) {
	s := postgres.BuildCredentialSecret(newCredsCluster(), "pw")
	for _, k := range []string{
		postgres.CredentialKeyUsername,
		postgres.CredentialKeyPassword,
		postgres.CredentialKeyHost,
		postgres.CredentialKeyPort,
		postgres.CredentialKeyDatabase,
		postgres.CredentialKeyURI,
	} {
		if _, ok := s.StringData[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
}

func TestBuildCredentialSecret_PasswordEmbeddedInURI(t *testing.T) {
	s := postgres.BuildCredentialSecret(newCredsCluster(), "supersecret")
	if !strings.Contains(s.StringData[postgres.CredentialKeyURI], "supersecret") {
		t.Errorf("URI does not contain password: %q", s.StringData[postgres.CredentialKeyURI])
	}
}

func TestBuildCredentialSecret_HostUsesPrimaryService(t *testing.T) {
	c := newCredsCluster()
	s := postgres.BuildCredentialSecret(c, "pw")
	if s.StringData[postgres.CredentialKeyHost] != postgres.PrimaryServiceName(c) {
		t.Errorf("host = %q, want %q", s.StringData[postgres.CredentialKeyHost], postgres.PrimaryServiceName(c))
	}
}

func TestBuildCredentialSecret_LabelsApplied(t *testing.T) {
	s := postgres.BuildCredentialSecret(newCredsCluster(), "pw")
	if s.Labels[postgres.LabelManagedBy] != postgres.OperatorName {
		t.Errorf("managed-by label = %q", s.Labels[postgres.LabelManagedBy])
	}
}
