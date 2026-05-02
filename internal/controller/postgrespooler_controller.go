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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/pgbouncer"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/postgres"
)

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// PostgresPoolerReconciler manages PgBouncer connection poolers for
// PostgresCluster instances that have the pooler feature enabled.
type PostgresPoolerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager registers the pooler controller with the manager.
func (r *PostgresPoolerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pgv1alpha1.PostgresCluster{}).
		Owns(&appsv1.Deployment{}).
		Named("postgrespooler").
		Complete(r)
}

// Reconcile ensures that PgBouncer resources match the desired state declared
// in the PostgresCluster spec.
func (r *PostgresPoolerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cluster := &pgv1alpha1.PostgresCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("getting PostgresCluster: %w", err)
	}

	if cluster.DeletionTimestamp != nil || cluster.Spec.Paused {
		return ctrl.Result{}, nil
	}

	// Only reconcile when a pooler is explicitly requested via annotation.
	if cluster.GetAnnotations()["pg.vigil.io/enable-pooler"] != "true" {
		return ctrl.Result{}, nil
	}

	log.Info("reconciling PgBouncer pooler", "cluster", cluster.Name)

	cfg := pgbouncer.DefaultConfig()
	primarySvc := postgres.PrimaryServiceName(cluster)

	if err := r.reconcilePoolerConfigMap(ctx, cluster, cfg, primarySvc); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcilePoolerDeployment(ctx, cluster, cfg); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcilePoolerService(ctx, cluster, cfg); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PostgresPoolerReconciler) reconcilePoolerConfigMap(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
	cfg pgbouncer.Config,
	primarySvc string,
) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgbouncer.ConfigMapName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Data: map[string]string{
			"pgbouncer.ini": cfg.INI(cluster.Name, primarySvc, cluster.Name),
		},
	}
	if err := controllerutil.SetControllerReference(cluster, cm, r.Scheme); err != nil {
		return fmt.Errorf("setting controller reference on pooler ConfigMap: %w", err)
	}

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting pooler ConfigMap: %w", err)
		}
		return r.Create(ctx, cm)
	}

	existing.Data = cm.Data
	return r.Update(ctx, existing)
}

func (r *PostgresPoolerReconciler) reconcilePoolerDeployment(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
	cfg pgbouncer.Config,
) error {
	replicas := int32(2)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgbouncer.DeploymentName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "pooler",
					"pg.vigil.io/cluster":         cluster.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "pooler",
						"pg.vigil.io/cluster":         cluster.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pgbouncer",
							Image: pgbouncer.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "pgbouncer",
									ContainerPort: int32(cfg.ListenPort),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pgbouncer-config",
									MountPath: "/etc/pgbouncer",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt32(int32(cfg.ListenPort)),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "pgbouncer-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: pgbouncer.ConfigMapName(cluster.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cluster, deploy, r.Scheme); err != nil {
		return fmt.Errorf("setting controller reference on pooler Deployment: %w", err)
	}

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting pooler Deployment: %w", err)
		}
		return r.Create(ctx, deploy)
	}

	existing.Spec = deploy.Spec
	return r.Update(ctx, existing)
}

func (r *PostgresPoolerReconciler) reconcilePoolerService(
	ctx context.Context,
	cluster *pgv1alpha1.PostgresCluster,
	cfg pgbouncer.Config,
) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgbouncer.ServiceName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    postgres.CommonLabels(cluster),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/component": "pooler",
				"pg.vigil.io/cluster":         cluster.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "pgbouncer",
					Port:       int32(cfg.ListenPort),
					TargetPort: intstr.FromInt32(int32(cfg.ListenPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cluster, svc, r.Scheme); err != nil {
		return fmt.Errorf("setting controller reference on pooler Service: %w", err)
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, existing)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting pooler Service: %w", err)
		}
		return r.Create(ctx, svc)
	}

	existing.Spec.Ports = svc.Spec.Ports
	return r.Update(ctx, existing)
}
