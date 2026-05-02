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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
)

var _ = Describe("PostgresBackup Controller", func() {
	const (
		backupName = "test-backup"
		backupNS   = "default"
		timeout    = time.Second * 10
		interval   = time.Millisecond * 250
	)

	ctx := context.Background()
	backupKey := types.NamespacedName{Name: backupName, Namespace: backupNS}

	newBackup := func(clusterRef string) *pgv1alpha1.PostgresBackup {
		return &pgv1alpha1.PostgresBackup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupName,
				Namespace: backupNS,
			},
			Spec: pgv1alpha1.PostgresBackupSpec{
				ClusterName: clusterRef,
				Method:      pgv1alpha1.BackupMethodBaseBackup,
				Online:      true,
			},
		}
	}

	backupReconciler := func() *PostgresBackupReconciler {
		return &PostgresBackupReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	}

	AfterEach(func() {
		backup := &pgv1alpha1.PostgresBackup{}
		err := k8sClient.Get(ctx, backupKey, backup)
		if err == nil {
			_ = k8sClient.Delete(ctx, backup)
		}
	})

	Describe("Cluster reference validation", func() {
		It("should not create a backup Job when the referenced cluster does not exist", func() {
			backup := newBackup("nonexistent-cluster")
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			r := backupReconciler()
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: backupKey})
			Expect(err).NotTo(HaveOccurred())

			updated := &pgv1alpha1.PostgresBackup{}
			Expect(k8sClient.Get(ctx, backupKey, updated)).To(Succeed())
			// The backup should not be in Running phase since the cluster is absent.
			Expect(updated.Status.Phase).NotTo(Equal(pgv1alpha1.BackupPhaseRunning))
		})
	})

	Describe("Cluster not ready", func() {
		const notReadyCluster = "not-ready-cluster"

		BeforeEach(func() {
			cluster := &pgv1alpha1.PostgresCluster{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: notReadyCluster, Namespace: backupNS}, cluster)
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, &pgv1alpha1.PostgresCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      notReadyCluster,
						Namespace: backupNS,
					},
					Spec: pgv1alpha1.PostgresClusterSpec{
						PostgresVersion: 16,
						Instances:       1,
						Storage: pgv1alpha1.StorageSpec{
							Size: resource.MustParse("1Gi"),
						},
					},
				})).To(Succeed())
			}
		})

		AfterEach(func() {
			cluster := &pgv1alpha1.PostgresCluster{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: notReadyCluster, Namespace: backupNS}, cluster)
			if err == nil {
				cluster.Finalizers = nil
				_ = k8sClient.Update(ctx, cluster)
				_ = k8sClient.Delete(ctx, cluster)
			}
		})

		It("should not create a backup Job when the cluster is not in Running phase", func() {
			backup := newBackup(notReadyCluster)
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			r := backupReconciler()
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: backupKey})
			Expect(err).NotTo(HaveOccurred())

			updated := &pgv1alpha1.PostgresBackup{}
			Expect(k8sClient.Get(ctx, backupKey, updated)).To(Succeed())
			// Phase should still be empty or Pending — not Running.
			Expect(updated.Status.Phase).NotTo(Equal(pgv1alpha1.BackupPhaseRunning))
		})
	})

	Describe("Terminal state", func() {
		It("should not re-reconcile a Completed backup", func() {
			backup := newBackup("some-cluster")
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			// Manually move to Completed.
			backup.Status.Phase = pgv1alpha1.BackupPhaseCompleted
			Expect(k8sClient.Status().Update(ctx, backup)).To(Succeed())

			r := backupReconciler()
			result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: backupKey})
			Expect(err).NotTo(HaveOccurred())
			// No requeue expected for completed backups.
			Expect(result.RequeueAfter).To(BeZero())
		})
	})
})
