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

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
	"github.com/Kitio-Tek/vigil/internal/postgres"
)

//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresbackups/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// PostgresBackupReconciler reconciles a PostgresBackup object.
type PostgresBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile drives a PostgresBackup from Pending through Running to Completed
// or Failed by managing the lifecycle of the underlying backup Job.
func (r *PostgresBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	backup := &pgv1alpha1.PostgresBackup{}
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Backups are immutable once they reach a terminal state.
	if backup.Status.Phase == pgv1alpha1.BackupPhaseCompleted ||
		backup.Status.Phase == pgv1alpha1.BackupPhaseFailed {
		log.Info("backup is in terminal state, skipping", "phase", backup.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Resolve the referenced cluster.
	cluster := &pgv1alpha1.PostgresCluster{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      backup.Spec.ClusterName,
		Namespace: backup.Namespace,
	}, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			postgres.SetBackupCondition(backup, "ClusterFound",
				metav1.ConditionFalse, "ClusterNotFound",
				fmt.Sprintf("PostgresCluster %q not found", backup.Spec.ClusterName))
			_ = r.Status().Update(ctx, backup)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Wait for the cluster to be ready before proceeding.
	if !postgres.IsClusterRunning(cluster) {
		log.Info("cluster is not running, waiting", "cluster", cluster.Name, "phase", cluster.Status.Phase)
		postgres.SetBackupCondition(backup, "ClusterReady",
			metav1.ConditionFalse, "ClusterNotReady",
			fmt.Sprintf("PostgresCluster %q is not in Running phase", cluster.Name))
		_ = r.Status().Update(ctx, backup)
		return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
	}

	// Look up an existing Job for this backup.
	job := &batchv1.Job{}
	jobName := postgres.BackupJobName(backup)
	err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: backup.Namespace}, job)

	if errors.IsNotFound(err) {
		return r.createBackupJob(ctx, backup, cluster)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Sync backup status from the Job status.
	return r.syncBackupStatus(ctx, backup, job)
}

// createBackupJob builds and submits the backup Job, then records the start time.
func (r *PostgresBackupReconciler) createBackupJob(
	ctx context.Context,
	backup *pgv1alpha1.PostgresBackup,
	cluster *pgv1alpha1.PostgresCluster,
) (ctrl.Result, error) {
	job := postgres.BuildBackupJob(backup, cluster)
	if err := controllerutil.SetControllerReference(backup, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, job); err != nil {
		return ctrl.Result{}, err
	}

	now := metav1.Now()
	backup.Status.Phase = pgv1alpha1.BackupPhaseRunning
	backup.Status.StartTime = &now
	postgres.SetBackupCondition(backup, "Running",
		metav1.ConditionTrue, "JobCreated", "Backup job has been created")

	return ctrl.Result{RequeueAfter: 15 * time.Second}, r.Status().Update(ctx, backup)
}

// syncBackupStatus reads the Job status and updates the PostgresBackup accordingly.
func (r *PostgresBackupReconciler) syncBackupStatus(
	ctx context.Context,
	backup *pgv1alpha1.PostgresBackup,
	job *batchv1.Job,
) (ctrl.Result, error) {
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == "True" {
			now := metav1.Now()
			backup.Status.Phase = pgv1alpha1.BackupPhaseCompleted
			backup.Status.CompletionTime = &now
			postgres.SetBackupCondition(backup, "Complete",
				metav1.ConditionTrue, "JobSucceeded", "Backup completed successfully")
			return ctrl.Result{}, r.Status().Update(ctx, backup)
		}
		if cond.Type == batchv1.JobFailed && cond.Status == "True" {
			now := metav1.Now()
			backup.Status.Phase = pgv1alpha1.BackupPhaseFailed
			backup.Status.CompletionTime = &now
			postgres.SetBackupCondition(backup, "Complete",
				metav1.ConditionFalse, "JobFailed", "Backup job failed")
			return ctrl.Result{}, r.Status().Update(ctx, backup)
		}
	}

	// Job is still running.
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager registers the backup controller with the manager.
func (r *PostgresBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pgv1alpha1.PostgresBackup{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
