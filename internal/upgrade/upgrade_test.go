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

package upgrade_test

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/upgrade"
)

func clusterWithVersion(version int32) *pgv1alpha1.PostgresCluster {
	return &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       pgv1alpha1.PostgresClusterSpec{PostgresVersion: version},
	}
}

func TestClassify_NoChange(t *testing.T) {
	cluster := clusterWithVersion(16)
	plan, err := upgrade.Classify(cluster, 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Kind != upgrade.KindNone {
		t.Errorf("expected KindNone, got %v", plan.Kind)
	}
}

func TestClassify_MajorUpgrade(t *testing.T) {
	cluster := clusterWithVersion(17)
	plan, err := upgrade.Classify(cluster, 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Kind != upgrade.KindMajor {
		t.Errorf("expected KindMajor, got %v", plan.Kind)
	}
	if plan.FromVersion != 16 || plan.ToVersion != 17 {
		t.Errorf("unexpected plan versions: %+v", plan)
	}
}

func TestClassify_Downgrade(t *testing.T) {
	cluster := clusterWithVersion(15)
	_, err := upgrade.Classify(cluster, 16)
	if err == nil {
		t.Error("expected error for downgrade")
	}
}

func TestClassify_UnsupportedVersion(t *testing.T) {
	cluster := clusterWithVersion(12)
	_, err := upgrade.Classify(cluster, 12)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		version int32
		wantErr bool
	}{
		{upgrade.MinSupportedVersion, false},
		{upgrade.MaxSupportedVersion, false},
		{16, false},
		{upgrade.MinSupportedVersion - 1, true},
		{upgrade.MaxSupportedVersion + 1, true},
		{0, true},
	}
	for _, tt := range tests {
		err := upgrade.ValidateVersion(tt.version)
		if tt.wantErr && err == nil {
			t.Errorf("expected error for version %d", tt.version)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("unexpected error for version %d: %v", tt.version, err)
		}
	}
}

func TestMajorUpgradeApproved(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				upgrade.AnnotationUpgradeApproved: "true",
			},
		},
	}
	if !upgrade.MajorUpgradeApproved(cluster) {
		t.Error("expected upgrade to be approved")
	}
}

func TestMajorUpgradeApproved_False(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{}
	if upgrade.MajorUpgradeApproved(cluster) {
		t.Error("expected upgrade to not be approved without annotation")
	}
}

func TestUpgradeInProgress(t *testing.T) {
	cluster := &pgv1alpha1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				upgrade.LabelUpgradeInProgress: "true",
			},
		},
	}
	if !upgrade.UpgradeInProgress(cluster) {
		t.Error("expected upgrade to be in progress")
	}
}

func TestImageTag(t *testing.T) {
	tag := upgrade.ImageTag(16)
	if !strings.Contains(tag, "16") {
		t.Errorf("image tag should contain version, got %q", tag)
	}
	if strings.Contains(tag, "alpine") {
		t.Errorf("non-alpine tag should not contain 'alpine', got %q", tag)
	}
}

func TestImageTagAlpine(t *testing.T) {
	tag := upgrade.ImageTagAlpine(16)
	if !strings.Contains(tag, "alpine") {
		t.Errorf("alpine tag should contain 'alpine', got %q", tag)
	}
}

func TestUpgradeJobName(t *testing.T) {
	name := upgrade.UpgradeJobName("my-cluster", 15, 16)
	if !strings.Contains(name, "my-cluster") {
		t.Errorf("job name should contain cluster name, got %q", name)
	}
	if !strings.Contains(name, "15") || !strings.Contains(name, "16") {
		t.Errorf("job name should contain versions, got %q", name)
	}
}

func TestSupportedVersions(t *testing.T) {
	versions := upgrade.SupportedVersions()
	if len(versions) == 0 {
		t.Error("expected non-empty supported versions list")
	}
	for _, v := range versions {
		if v < upgrade.MinSupportedVersion || v > upgrade.MaxSupportedVersion {
			t.Errorf("version %d is outside supported range", v)
		}
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind upgrade.Kind
		want string
	}{
		{upgrade.KindNone, "none"},
		{upgrade.KindMinor, "minor"},
		{upgrade.KindMajor, "major"},
	}
	for _, tt := range tests {
		if tt.kind.String() != tt.want {
			t.Errorf("Kind.String() = %q, want %q", tt.kind.String(), tt.want)
		}
	}
}
