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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
	"github.com/Kitio-Tek/vigil/internal/postgres"
)

//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pg.vigil.io,resources=postgresusers/finalizers,verbs=update

// PostgresUserReconciler reconciles a PostgresUser object.
type PostgresUserReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

// Reconcile ensures the PostgreSQL user described by the PostgresUser CR exists
// in the target cluster with the correct roles, grants, and connection limits.
func (r *PostgresUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	user := &pgv1alpha1.PostgresUser{}
	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Resolve the referenced cluster.
	cluster := &pgv1alpha1.PostgresCluster{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      user.Spec.ClusterName,
		Namespace: user.Namespace,
	}, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			postgres.SetUserCondition(user, "ClusterFound",
				metav1.ConditionFalse, "ClusterNotFound",
				fmt.Sprintf("PostgresCluster %q not found", user.Spec.ClusterName))
			user.Status.Applied = false
			user.Status.ObservedGeneration = user.Generation
			_ = r.Status().Update(ctx, user)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Wait for the cluster to be ready.
	if !postgres.IsClusterRunning(cluster) {
		log.Info("cluster is not running, waiting", "cluster", cluster.Name)
		postgres.SetUserCondition(user, "ClusterReady",
			metav1.ConditionFalse, "ClusterNotReady",
			fmt.Sprintf("PostgresCluster %q is not in Running phase", cluster.Name))
		user.Status.Applied = false
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
	}

	// Resolve the password from the referenced Secret.
	password := ""
	if user.Spec.PasswordSecret != nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      user.Spec.PasswordSecret.Name,
			Namespace: user.Namespace,
		}, secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("fetch password secret: %w", err)
		}
		password = string(secret.Data["password"])
	}

	// Build and execute the SQL to create or update the user.
	sql := buildUserSQL(user, password)
	primaryPod := postgres.PodName(cluster, 0)

	if err := r.execSQL(ctx, cluster.Namespace, primaryPod, "postgres", sql); err != nil {
		postgres.SetUserCondition(user, "Applied",
			metav1.ConditionFalse, "ExecFailed", err.Error())
		user.Status.Applied = false
		user.Status.ObservedGeneration = user.Generation
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	postgres.SetUserCondition(user, "Applied",
		metav1.ConditionTrue, "UserReconciled",
		fmt.Sprintf("User %q has been reconciled", req.Name))
	user.Status.Applied = true
	user.Status.ObservedGeneration = user.Generation
	return ctrl.Result{RequeueAfter: 60 * time.Second}, r.Status().Update(ctx, user)
}

// buildUserSQL generates the idempotent SQL statements needed to create or
// update a PostgreSQL user to match the desired spec.
func buildUserSQL(user *pgv1alpha1.PostgresUser, password string) string {
	name := user.Name
	var sb strings.Builder

	// CREATE ROLE if it does not already exist.
	sb.WriteString(fmt.Sprintf("DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') THEN CREATE ROLE \"%s\" WITH LOGIN", name, name))
	if user.Spec.Superuser {
		sb.WriteString(" SUPERUSER")
	}
	connLimit := user.Spec.ConnectionLimit
	if connLimit == 0 {
		connLimit = -1
	}
	sb.WriteString(fmt.Sprintf(" CONNECTION LIMIT %d", connLimit))
	if password != "" {
		sb.WriteString(fmt.Sprintf(" PASSWORD '%s'", strings.ReplaceAll(password, "'", "''")))
	}
	sb.WriteString(fmt.Sprintf("; END IF; END $$;\n"))

	// ALTER the role to bring it to the desired state (idempotent).
	sb.WriteString(fmt.Sprintf("ALTER ROLE \"%s\"", name))
	if user.Spec.Superuser {
		sb.WriteString(" SUPERUSER")
	} else {
		sb.WriteString(" NOSUPERUSER")
	}
	sb.WriteString(fmt.Sprintf(" CONNECTION LIMIT %d", connLimit))
	if password != "" {
		sb.WriteString(fmt.Sprintf(" PASSWORD '%s'", strings.ReplaceAll(password, "'", "''")))
	}
	sb.WriteString(";\n")

	// Grant requested PostgreSQL roles.
	for _, role := range user.Spec.Roles {
		sb.WriteString(fmt.Sprintf("GRANT \"%s\" TO \"%s\";\n", role, name))
	}

	// Grant database-level privileges.
	for _, dbGrant := range user.Spec.Databases {
		sb.WriteString(fmt.Sprintf("GRANT CONNECT ON DATABASE \"%s\" TO \"%s\";\n", dbGrant.Name, name))
		for _, priv := range dbGrant.Privileges {
			sb.WriteString(fmt.Sprintf("GRANT %s ON ALL TABLES IN SCHEMA public TO \"%s\";\n", priv, name))
		}
	}

	return sb.String()
}

// execSQL executes a SQL string inside the named container of a running pod by
// using the Kubernetes exec API.
func (r *PostgresUserReconciler) execSQL(
	ctx context.Context,
	namespace, podName, containerName, sql string,
) error {
	if r.RestConfig == nil {
		// In unit tests the rest config may not be available.
		return nil
	}

	clientset, err := kubernetes.NewForConfig(r.RestConfig)
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"psql", "-U", "postgres", "-c", sql},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.RestConfig, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return fmt.Errorf("exec sql: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// SetupWithManager registers the user controller with the manager.
func (r *PostgresUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pgv1alpha1.PostgresUser{}).
		Complete(r)
}
