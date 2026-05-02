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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
)

var _ = Describe("PostgresUser Controller", func() {
	const (
		userName = "test-user"
		userNS   = "default"
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	ctx := context.Background()
	userKey := types.NamespacedName{Name: userName, Namespace: userNS}

	newUser := func(clusterRef string) *pgv1alpha1.PostgresUser {
		return &pgv1alpha1.PostgresUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userName,
				Namespace: userNS,
			},
			Spec: pgv1alpha1.PostgresUserSpec{
				ClusterName:     clusterRef,
				Superuser:       false,
				ConnectionLimit: -1,
			},
		}
	}

	userReconciler := func() *PostgresUserReconciler {
		return &PostgresUserReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			// RestConfig intentionally nil in unit tests — execSQL short-circuits.
		}
	}

	AfterEach(func() {
		u := &pgv1alpha1.PostgresUser{}
		err := k8sClient.Get(ctx, userKey, u)
		if err == nil {
			_ = k8sClient.Delete(ctx, u)
		}
	})

	Describe("Cluster reference", func() {
		It("should set Applied=false when the referenced cluster is not found", func() {
			user := newUser("missing-cluster")
			Expect(k8sClient.Create(ctx, user)).To(Succeed())

			r := userReconciler()
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: userKey})
			Expect(err).NotTo(HaveOccurred())

			updated := &pgv1alpha1.PostgresUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, userKey, updated)
				return !updated.Status.Applied
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("Status conditions", func() {
		It("should record a ClusterNotFound condition when the cluster is absent", func() {
			user := newUser("missing-cluster")
			Expect(k8sClient.Create(ctx, user)).To(Succeed())

			r := userReconciler()
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: userKey})
			Expect(err).NotTo(HaveOccurred())

			updated := &pgv1alpha1.PostgresUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, userKey, updated)
				return len(updated.Status.Conditions) > 0
			}, timeout, interval).Should(BeTrue())

			found := false
			for _, c := range updated.Status.Conditions {
				if c.Reason == "ClusterNotFound" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should not error when the cluster exists but is not running", func() {
			const notReadyCluster = "user-not-ready-cluster"

			cluster := &pgv1alpha1.PostgresCluster{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: notReadyCluster, Namespace: userNS}, cluster)
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, &pgv1alpha1.PostgresCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      notReadyCluster,
						Namespace: userNS,
					},
					Spec: pgv1alpha1.PostgresClusterSpec{
						PostgresVersion: 16,
						Instances:       1,
					},
				})).To(Succeed())
			}

			DeferCleanup(func() {
				c := &pgv1alpha1.PostgresCluster{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: notReadyCluster, Namespace: userNS}, c); err == nil {
					c.Finalizers = nil
					_ = k8sClient.Update(ctx, c)
					_ = k8sClient.Delete(ctx, c)
				}
			})

			user := newUser(notReadyCluster)
			Expect(k8sClient.Create(ctx, user)).To(Succeed())

			r := userReconciler()
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: userKey})
			Expect(err).NotTo(HaveOccurred())

			updated := &pgv1alpha1.PostgresUser{}
			Expect(k8sClient.Get(ctx, userKey, updated)).To(Succeed())
			Expect(updated.Status.Applied).To(BeFalse())
		})
	})
})
