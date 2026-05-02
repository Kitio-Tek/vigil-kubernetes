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

// Package workflows implements high-level reconcile orchestration steps that
// are shared across multiple controllers. Each function represents a discrete
// reconciliation step that can be composed in a controller's Reconcile method.
package workflows

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
)

// ReconcileResult is a structured result returned by workflow steps.
type ReconcileResult struct {
	// Requeue indicates that the reconcile loop should be requeued.
	Requeue bool
	// Err holds any error that caused this step to fail.
	Err error
}

// Done returns a successful, non-requeue result.
func Done() ReconcileResult { return ReconcileResult{} }

// RequeueErr returns a result that requeues after an error.
func RequeueErr(err error) ReconcileResult { return ReconcileResult{Requeue: true, Err: err} }

// RequeueAfter returns a result that requeues without error.
func RequeueAfter() ReconcileResult { return ReconcileResult{Requeue: true} }

// Ctrl converts a ReconcileResult to a controller-runtime Result.
func (r ReconcileResult) Ctrl() (ctrl.Result, error) {
	return ctrl.Result{Requeue: r.Requeue}, r.Err
}

// EnsureNamespace creates the given namespace if it does not exist.
func EnsureNamespace(ctx context.Context, c client.Client, name string) error {
	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: name}, ns); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting namespace %q: %w", name, err)
		}
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
		if err := c.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating namespace %q: %w", name, err)
		}
	}
	return nil
}

// ReconcileSecret ensures a Secret exists with the given data. If the Secret
// already exists, it is updated only when the data differs.
func ReconcileSecret(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner metav1.Object,
	desired *corev1.Secret,
) error {
	log := log.FromContext(ctx)
	if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	existing := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting secret %q: %w", desired.Name, err)
		}
		log.Info("creating Secret", "name", desired.Name)
		return c.Create(ctx, desired)
	}

	if secretDataChanged(existing.Data, desired.Data) {
		existing.Data = desired.Data
		log.Info("updating Secret", "name", desired.Name)
		return c.Update(ctx, existing)
	}
	return nil
}

// ReconcileConfigMap ensures a ConfigMap exists with the given data. If the
// ConfigMap already exists, it is updated only when the data differs.
func ReconcileConfigMap(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner metav1.Object,
	desired *corev1.ConfigMap,
) error {
	log := log.FromContext(ctx)
	if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	existing := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting configmap %q: %w", desired.Name, err)
		}
		log.Info("creating ConfigMap", "name", desired.Name)
		return c.Create(ctx, desired)
	}

	if configMapDataChanged(existing.Data, desired.Data) {
		existing.Data = desired.Data
		log.Info("updating ConfigMap", "name", desired.Name)
		return c.Update(ctx, existing)
	}
	return nil
}

// CleanupOrphanedPVCs removes PersistentVolumeClaims that are no longer
// referenced by any pod in the cluster. This is called when the cluster is
// scaled down.
func CleanupOrphanedPVCs(
	ctx context.Context,
	c client.Client,
	cluster *pgv1alpha1.PostgresCluster,
	labelSelector map[string]string,
) error {
	log := log.FromContext(ctx)
	list := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, list,
		client.InNamespace(cluster.Namespace),
		client.MatchingLabels(labelSelector),
	); err != nil {
		return fmt.Errorf("listing PVCs: %w", err)
	}

	for i := range list.Items {
		pvc := &list.Items[i]
		if pvc.DeletionTimestamp != nil {
			continue
		}
		ordinal := pvcOrdinal(pvc.Name, cluster.Name)
		if ordinal >= int(cluster.Spec.Instances) {
			log.Info("deleting orphaned PVC", "name", pvc.Name)
			if err := c.Delete(ctx, pvc); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("deleting PVC %q: %w", pvc.Name, err)
			}
		}
	}
	return nil
}

func secretDataChanged(old, new map[string][]byte) bool {
	if len(old) != len(new) {
		return true
	}
	for k, v := range old {
		nv, ok := new[k]
		if !ok || string(v) != string(nv) {
			return true
		}
	}
	return false
}

func configMapDataChanged(old, new map[string]string) bool {
	if len(old) != len(new) {
		return true
	}
	for k, v := range old {
		if new[k] != v {
			return true
		}
	}
	return false
}

// pvcOrdinal parses the ordinal from a StatefulSet PVC name of the form
// "<volume>-<cluster>-<n>".
func pvcOrdinal(pvcName, clusterName string) int {
	prefix := "-" + clusterName + "-"
	idx := len(pvcName) - 1
	for idx >= 0 && pvcName[idx] >= '0' && pvcName[idx] <= '9' {
		idx--
	}
	if idx < 0 || pvcName[idx] != '-' {
		return -1
	}
	_ = prefix
	n := 0
	mul := 1
	for j := len(pvcName) - 1; j > idx; j-- {
		n += int(pvcName[j]-'0') * mul
		mul *= 10
	}
	return n
}
