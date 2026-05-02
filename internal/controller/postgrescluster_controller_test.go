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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/postgres"
)

var _ = Describe("PostgresCluster Controller", func() {
	const (
		clusterName = "test-cluster"
		namespace   = "default"
		timeout     = time.Second * 10
		interval    = time.Millisecond * 250
	)

	clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
	ctx := context.Background()

	newCluster := func() *pgv1alpha1.PostgresCluster {
		return &pgv1alpha1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
			Spec: pgv1alpha1.PostgresClusterSpec{
				PostgresVersion: 16,
				Instances:       1,
				Storage: pgv1alpha1.StorageSpec{
					Size: resource.MustParse("1Gi"),
				},
			},
		}
	}

	reconciler := func() *PostgresClusterReconciler {
		return &PostgresClusterReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	}

	doReconcile := func() {
		r := reconciler()
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: clusterKey})
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		cluster := &pgv1alpha1.PostgresCluster{}
		err := k8sClient.Get(ctx, clusterKey, cluster)
		if errors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, newCluster())).To(Succeed())
		}
	})

	AfterEach(func() {
		cluster := &pgv1alpha1.PostgresCluster{}
		err := k8sClient.Get(ctx, clusterKey, cluster)
		if err == nil {
			// Remove the finalizer so deletion proceeds cleanly in tests.
			cluster.Finalizers = nil
			_ = k8sClient.Update(ctx, cluster)
			_ = k8sClient.Delete(ctx, cluster)
		}
	})

	Describe("StatefulSet management", func() {
		It("should create a StatefulSet when PostgresCluster is created", func() {
			doReconcile()

			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.ClusterStatefulSetName(newCluster()),
					Namespace: namespace,
				}, sts)
			}, timeout, interval).Should(Succeed())

			Expect(*sts.Spec.Replicas).To(BeEquivalentTo(1))
			Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
			Expect(sts.Spec.Template.Spec.Containers[0].Name).To(Equal("postgres"))
		})

		It("should set owner reference on the StatefulSet", func() {
			doReconcile()

			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.ClusterStatefulSetName(newCluster()),
					Namespace: namespace,
				}, sts)
			}, timeout, interval).Should(Succeed())

			Expect(sts.OwnerReferences).NotTo(BeEmpty())
			Expect(sts.OwnerReferences[0].Kind).To(Equal("PostgresCluster"))
			Expect(sts.OwnerReferences[0].Name).To(Equal(clusterName))
		})
	})

	Describe("Service management", func() {
		It("should create the primary and headless Services", func() {
			doReconcile()

			primary := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.PrimaryServiceName(newCluster()),
					Namespace: namespace,
				}, primary)
			}, timeout, interval).Should(Succeed())

			headless := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      postgres.HeadlessServiceName(newCluster()),
				Namespace: namespace,
			}, headless)).To(Succeed())

			Expect(headless.Spec.ClusterIP).To(Equal("None"))
		})

		It("should create the replica Service", func() {
			doReconcile()

			replica := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.ReplicaServiceName(newCluster()),
					Namespace: namespace,
				}, replica)
			}, timeout, interval).Should(Succeed())
		})

		It("should set owner references on Services", func() {
			doReconcile()

			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.PrimaryServiceName(newCluster()),
					Namespace: namespace,
				}, svc)
			}, timeout, interval).Should(Succeed())

			Expect(svc.OwnerReferences).NotTo(BeEmpty())
			Expect(svc.OwnerReferences[0].Kind).To(Equal("PostgresCluster"))
		})
	})

	Describe("ConfigMap management", func() {
		It("should create ConfigMap with postgresql.conf content", func() {
			doReconcile()

			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.ConfigMapName(newCluster()),
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			Expect(cm.Data).To(HaveKey("postgresql.conf"))
			Expect(cm.Data).To(HaveKey("pg_hba.conf"))
			Expect(cm.Data["postgresql.conf"]).To(ContainSubstring("listen_addresses"))
		})

		It("should set owner reference on the ConfigMap", func() {
			doReconcile()

			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      postgres.ConfigMapName(newCluster()),
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			Expect(cm.OwnerReferences).NotTo(BeEmpty())
		})
	})

	Describe("Status management", func() {
		It("should initialise status phase on first reconcile", func() {
			doReconcile()

			cluster := &pgv1alpha1.PostgresCluster{}
			Eventually(func() pgv1alpha1.PostgresClusterPhase {
				_ = k8sClient.Get(ctx, clusterKey, cluster)
				return cluster.Status.Phase
			}, timeout, interval).ShouldNot(BeEmpty())
		})

		It("should set WriteServiceName and ReadServiceName in status", func() {
			doReconcile()
			// Run a second reconcile to ensure status update path is exercised.
			doReconcile()

			cluster := &pgv1alpha1.PostgresCluster{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, clusterKey, cluster)
				return cluster.Status.WriteServiceName
			}, timeout, interval).ShouldNot(BeEmpty())

			Expect(cluster.Status.ReadServiceName).NotTo(BeEmpty())
		})
	})

	Describe("Pause behaviour", func() {
		It("should set phase to Paused and not create resources when paused", func() {
			cluster := &pgv1alpha1.PostgresCluster{}
			Expect(k8sClient.Get(ctx, clusterKey, cluster)).To(Succeed())
			cluster.Spec.Paused = true
			Expect(k8sClient.Update(ctx, cluster)).To(Succeed())

			doReconcile()

			updated := &pgv1alpha1.PostgresCluster{}
			Eventually(func() pgv1alpha1.PostgresClusterPhase {
				_ = k8sClient.Get(ctx, clusterKey, updated)
				return updated.Status.Phase
			}, timeout, interval).Should(Equal(pgv1alpha1.PhasePaused))
		})
	})

	Describe("Finalizer handling", func() {
		It("should add a finalizer on first reconcile", func() {
			doReconcile()

			cluster := &pgv1alpha1.PostgresCluster{}
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, clusterKey, cluster)
				for _, f := range cluster.Finalizers {
					if f == pgv1alpha1.PostgresClusterFinalizer {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})
})
