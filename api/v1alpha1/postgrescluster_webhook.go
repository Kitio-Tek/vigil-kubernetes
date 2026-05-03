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

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/Kitio-Tek/athos-kubernetes/internal/cronexpr"
)

var postgresclusterlog = logf.Log.WithName("postgrescluster-webhook")

// SetupWebhookWithManager registers the defaulting and validation webhooks.
func (r *PostgresCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-pg-athos-io-v1alpha1-postgrescluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=pg.athos.io,resources=postgresclusters,verbs=create;update,versions=v1alpha1,name=mpostgrescluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &PostgresCluster{}

// Default applies defaulting logic to a PostgresCluster.
func (r *PostgresCluster) Default() {
	postgresclusterlog.Info("defaulting", "name", r.Name)

	if r.Spec.PostgresVersion == 0 {
		r.Spec.PostgresVersion = 16
	}

	if r.Spec.Instances == 0 {
		r.Spec.Instances = 1
	}

	if r.Spec.Storage.Size.IsZero() {
		r.Spec.Storage.Size = resource.MustParse("10Gi")
	}

	if len(r.Spec.Storage.AccessModes) == 0 {
		r.Spec.Storage.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	if r.Spec.Backup != nil && r.Spec.Backup.RetentionPolicy == "" {
		r.Spec.Backup.RetentionPolicy = "7d"
	}

	if r.Spec.Monitoring == nil {
		r.Spec.Monitoring = &MonitoringSpec{
			Enabled: true,
			Port:    9187,
		}
	}
	if r.Spec.Monitoring.Port == 0 {
		r.Spec.Monitoring.Port = 9187
	}

	if r.Spec.TLS == nil {
		r.Spec.TLS = &TLSSpec{Enabled: true}
	}
}

// +kubebuilder:webhook:path=/validate-pg-athos-io-v1alpha1-postgrescluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=pg.athos.io,resources=postgresclusters,verbs=create;update,versions=v1alpha1,name=vpostgrescluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PostgresCluster{}

// ValidateCreate validates a new PostgresCluster.
func (r *PostgresCluster) ValidateCreate() (admission.Warnings, error) {
	postgresclusterlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate validates an update to an existing PostgresCluster.
func (r *PostgresCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	postgresclusterlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete validates a deletion request. No restrictions are applied.
func (r *PostgresCluster) ValidateDelete() (admission.Warnings, error) {
	postgresclusterlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

// validate applies all validation rules and accumulates errors.
func (r *PostgresCluster) validate() (admission.Warnings, error) {
	var allErrs field.ErrorList

	allErrs = append(allErrs, r.validateVersion()...)
	allErrs = append(allErrs, r.validateInstances()...)
	allErrs = append(allErrs, r.validateStorage()...)
	allErrs = append(allErrs, r.validateBackup()...)

	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf("validation failed: %s", allErrs.ToAggregate())
}

// validateVersion checks that the PostgreSQL major version is in the supported range.
func (r *PostgresCluster) validateVersion() field.ErrorList {
	var errs field.ErrorList
	v := r.Spec.PostgresVersion
	if v < 14 || v > 17 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "postgresVersion"),
			v,
			"must be between 14 and 17 inclusive",
		))
	}
	return errs
}

// validateInstances checks that the instance count is valid. An instance count
// greater than 1 should be odd so that automatic failover reaches a clear
// majority vote. A count of 1 is always valid (single-instance mode).
func (r *PostgresCluster) validateInstances() field.ErrorList {
	var errs field.ErrorList
	n := r.Spec.Instances
	if n < 1 || n > 10 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "instances"),
			n,
			"must be between 1 and 10",
		))
		return errs
	}
	if n > 1 && n%2 == 0 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "instances"),
			n,
			"must be 1 or an odd number greater than 1 to ensure proper quorum",
		))
	}
	return errs
}

// validateStorage checks that storage size is positive.
func (r *PostgresCluster) validateStorage() field.ErrorList {
	var errs field.ErrorList
	if r.Spec.Storage.Size.Cmp(resource.MustParse("0")) <= 0 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "storage", "size"),
			r.Spec.Storage.Size.String(),
			"must be a positive quantity",
		))
	}
	return errs
}

// validateBackup checks that the backup configuration is coherent.
func (r *PostgresCluster) validateBackup() field.ErrorList {
	var errs field.ErrorList
	b := r.Spec.Backup
	if b == nil || !b.Enabled {
		return errs
	}
	if b.Schedule != "" {
		if err := cronexpr.Validate(b.Schedule); err != nil {
			errs = append(errs, field.Invalid(
				field.NewPath("spec", "backup", "schedule"),
				b.Schedule,
				err.Error(),
			))
		}
	}
	return errs
}
