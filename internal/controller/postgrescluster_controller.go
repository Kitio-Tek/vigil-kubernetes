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

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/athos-kubernetes/internal/events"
	"github.com/Kitio-Tek/athos-kubernetes/internal/password"
	"github.com/Kitio-Tek/athos-kubernetes/internal/pdb"
	"github.com/Kitio-Tek/athos-kubernetes/internal/postgres"
)

//+kubebuilder:rbac:groups=pg.athos.io,resources=postgresclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pg.athos.io,resources=postgresclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pg.athos.io,resources=postgresclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// PostgresClusterReconciler reconciles a PostgresCluster object.
type PostgresClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// recordEventf publishes a Kubernetes Event for the cluster, falling back
// to a no-op when no Recorder is configured (which is the case in unit
// tests that drive Reconcile directly).
func (r *PostgresClusterReconciler) recordEventf(
	cluster *pgv1alpha1.PostgresCluster,
	eventType, reason, format string,
	args ...interface{},
) {
	if r.Recorder == nil {
		return
	}
	r.Recorder.Eventf(cluster, eventType, reason, format, args...)
}

// Reconcile is the core reconciliation loop. It drives the cluster from its
// current observed state toward the desired state expressed in the
// PostgresCluster spec.
func (r *PostgresClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the PostgresCluster resource.
	cluster := &pgv1alpha1.PostgresCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle pause: update status and stop reconciling.
	if cluster.Spec.Paused {
		if cluster.Status.Phase != pgv1alpha1.PhasePaused {
			cluster.Status.Phase = pgv1alpha1.PhasePaused
			postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionProgressing,
				metav1.ConditionFalse, "Paused", "Reconciliation is suspended")
			if err := r.Status().Update(ctx, cluster); err != nil {
				return ctrl.Result{}, err
			}
			r.recordEventf(cluster, corev1.EventTypeWarning, events.EventReasonPaused,
				"Reconciliation paused via spec.paused")
		}
		log.Info("cluster is paused, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Add our finalizer when it is absent.
	if !controllerutil.ContainsFinalizer(cluster, pgv1alpha1.PostgresClusterFinalizer) {
		controllerutil.AddFinalizer(cluster, pgv1alpha1.PostgresClusterFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion: clean up and remove the finalizer.
	if !cluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, cluster)
	}

	// Bootstrap the status phase on first reconcile.
	if cluster.Status.Phase == "" {
		cluster.Status.Phase = pgv1alpha1.PhaseInitializing
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionProgressing,
			metav1.ConditionTrue, "Initializing", "Cluster is being initialised")
		if err := r.Status().Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile sub-resources in dependency order.
	if err := r.reconcileServiceAccount(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile serviceaccount: %w", err)
	}

	if err := r.reconcileCredentials(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile credentials: %w", err)
	}

	if err := r.reconcileConfigMap(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile configmap: %w", err)
	}

	if err := r.reconcileStatefulSet(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile statefulset: %w", err)
	}

	if err := r.reconcileServices(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile services: %w", err)
	}

	if err := r.reconcilePDB(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile pdb: %w", err)
	}

	if err := r.updateStatus(ctx, cluster); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileDelete performs cleanup of cluster-owned resources and removes the
// finalizer, allowing the PostgresCluster object itself to be deleted.
func (r *PostgresClusterReconciler) reconcileDelete(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("handling cluster deletion")

	// Sub-resources with ownerReferences will be garbage-collected by Kubernetes.
	// We only need to remove the finalizer here.
	controllerutil.RemoveFinalizer(cluster, pgv1alpha1.PostgresClusterFinalizer)
	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// reconcileServiceAccount ensures a ServiceAccount exists for the cluster.
func (r *PostgresClusterReconciler) reconcileServiceAccount(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	// Skip if the user has pointed to an existing ServiceAccount.
	if cluster.Spec.ServiceAccountName != "" {
		return nil
	}

	desired := postgres.BuildServiceAccount(cluster)
	if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	return err
}

// reconcileCredentials ensures the operator-managed credentials Secret
// exists. The Secret is created once with a freshly generated password and
// is left untouched on subsequent reconciles so user-driven password
// rotation is preserved.
func (r *PostgresClusterReconciler) reconcileCredentials(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      postgres.SecretName(cluster),
		Namespace: cluster.Namespace,
	}, existing)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	pw, err := password.DefaultGenerator().Generate()
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}
	desired := postgres.BuildCredentialSecret(cluster, pw)
	if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, desired); err != nil {
		return err
	}
	r.recordEventf(cluster, corev1.EventTypeNormal, events.EventReasonCreated,
		"Created credentials Secret %q", desired.Name)
	return nil
}

// reconcileConfigMap ensures the postgresql.conf / pg_hba.conf ConfigMap is
// up to date.
func (r *PostgresClusterReconciler) reconcileConfigMap(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	desired := postgres.BuildConfigMap(cluster)
	if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		r.recordEventf(cluster, corev1.EventTypeNormal, events.EventReasonCreated,
			"Created ConfigMap %q", desired.Name)
		return nil
	}
	if err != nil {
		return err
	}

	// Update data if it has diverged.
	existing.Data = desired.Data
	return r.Update(ctx, existing)
}

// reconcileStatefulSet ensures the StatefulSet matches the desired spec.
func (r *PostgresClusterReconciler) reconcileStatefulSet(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	desired := postgres.BuildStatefulSet(cluster)
	if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
		return err
	}

	existing := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		r.recordEventf(cluster, corev1.EventTypeNormal, events.EventReasonCreated,
			"Created StatefulSet %q with %d replicas", desired.Name, *desired.Spec.Replicas)
		return nil
	}
	if err != nil {
		return err
	}

	// Propagate mutable fields and emit a scale event when replicas change.
	if existing.Spec.Replicas != nil && desired.Spec.Replicas != nil &&
		*existing.Spec.Replicas != *desired.Spec.Replicas {
		r.recordEventf(cluster, corev1.EventTypeNormal, events.EventReasonUpdated,
			"Scaling StatefulSet %q from %d to %d replicas",
			desired.Name, *existing.Spec.Replicas, *desired.Spec.Replicas)
	}
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template = desired.Spec.Template
	return r.Update(ctx, existing)
}

// reconcileServices ensures the primary, replica, and headless Services exist.
func (r *PostgresClusterReconciler) reconcileServices(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	services := []*corev1.Service{
		postgres.BuildHeadlessService(cluster),
		postgres.BuildPrimaryService(cluster),
		postgres.BuildReplicaService(cluster),
	}

	for _, svc := range services {
		if err := controllerutil.SetControllerReference(cluster, svc, r.Scheme); err != nil {
			return err
		}

		existing := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, existing)
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, svc); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		// Update ports and selector on existing services.
		existing.Spec.Ports = svc.Spec.Ports
		existing.Spec.Selector = svc.Spec.Selector
		if err := r.Update(ctx, existing); err != nil {
			return err
		}
	}
	return nil
}

// reconcilePDB ensures a PodDisruptionBudget protects the cluster pods
// during voluntary disruptions. The recommended sizing depends on the
// configured instance count: clusters with three or more pods get a
// MinAvailable=N-1 guard, smaller clusters get MaxUnavailable=1.
func (r *PostgresClusterReconciler) reconcilePDB(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	spec := pdb.RecommendedFor(
		cluster.Name+"-pdb",
		cluster.Namespace,
		int(cluster.Spec.Instances),
		postgres.SelectorLabels(cluster),
		postgres.CommonLabels(cluster),
	)
	desired := pdb.Build(spec)
	if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
		return err
	}

	existing := &policyv1.PodDisruptionBudget{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		r.recordEventf(cluster, corev1.EventTypeNormal, events.EventReasonCreated,
			"Created PodDisruptionBudget %q", desired.Name)
		return nil
	}
	if err != nil {
		return err
	}
	existing.Spec = desired.Spec
	return r.Update(ctx, existing)
}

// updateStatus refreshes the PostgresCluster status based on the current
// StatefulSet state.
func (r *PostgresClusterReconciler) updateStatus(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
) error {
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      postgres.ClusterStatefulSetName(cluster),
		Namespace: cluster.Namespace,
	}, sts)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	ready := int32(0)
	if sts.Status.ReadyReplicas > 0 {
		ready = sts.Status.ReadyReplicas
	}
	cluster.Status.ReadyInstances = ready

	desired := cluster.Spec.Instances
	switch {
	case ready == 0:
		cluster.Status.Phase = pgv1alpha1.PhaseCreating
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
			metav1.ConditionFalse, "NoReadyInstances", "No instances are ready yet")
	case ready < desired:
		cluster.Status.Phase = pgv1alpha1.PhaseDegraded
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
			metav1.ConditionFalse, "InsufficientInstances",
			fmt.Sprintf("%d of %d instances are ready", ready, desired))
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionDegraded,
			metav1.ConditionTrue, "InsufficientInstances",
			fmt.Sprintf("%d of %d instances are ready", ready, desired))
	default:
		cluster.Status.Phase = pgv1alpha1.PhaseRunning
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionReady,
			metav1.ConditionTrue, "AllInstancesReady",
			fmt.Sprintf("All %d instances are ready", desired))
		postgres.SetClusterCondition(cluster, pgv1alpha1.ConditionDegraded,
			metav1.ConditionFalse, "AllInstancesReady", "Cluster is fully operational")
	}

	// Identify the primary pod (ordinal 0 in the StatefulSet).
	cluster.Status.CurrentPrimary = postgres.PodName(cluster, 0)
	cluster.Status.WriteServiceName = postgres.PrimaryServiceName(cluster)
	cluster.Status.ReadServiceName = postgres.ReplicaServiceName(cluster)
	cluster.Status.ObservedGeneration = cluster.Generation
	cluster.Status.PostgresVersion = fmt.Sprintf("%d", cluster.Spec.PostgresVersion)

	return r.Status().Update(ctx, cluster)
}

// SetupWithManager registers the controller with the manager and declares the
// set of owned resource types.
func (r *PostgresClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("athos-postgrescluster")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&pgv1alpha1.PostgresCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Secret{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
