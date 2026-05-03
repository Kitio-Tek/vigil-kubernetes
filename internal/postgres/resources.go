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

package postgres

import (
	pgv1alpha1 "github.com/Kitio-Tek/athos-kubernetes/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// DataVolumeName is the name of the PostgreSQL data PVC mount.
	DataVolumeName = "pgdata"
	// DataMountPath is the container path where the data volume is mounted.
	DataMountPath = "/var/lib/postgresql/data"
	// ConfigVolumeName is the name of the configuration ConfigMap volume.
	ConfigVolumeName = "pgconfig"
	// ConfigMountPath is where the ConfigMap is mounted inside the container.
	ConfigMountPath = "/etc/postgresql"
	// PostgresPort is the default PostgreSQL port.
	PostgresPort = 5432
)

// BuildStatefulSet constructs a StatefulSet for the given PostgresCluster.
func BuildStatefulSet(cluster *pgv1alpha1.PostgresCluster) *appsv1.StatefulSet {
	labels := CommonLabels(cluster)
	selector := SelectorLabels(cluster)
	image := PostgresImageTag(cluster.Spec.PostgresVersion)
	replicas := cluster.Spec.Instances
	saName := ServiceAccountName(cluster)

	podLabels := MergeLabels(labels, map[string]string{
		LabelRole: RolePrimary,
	})

	// Build the PostgreSQL container. The security context drops every
	// Linux capability and forbids privilege escalation; the postgres
	// process does not need any of them. Filesystem-write access is
	// retained because the postgres image touches /tmp at startup.
	allowPrivEsc := false
	pgContainer := corev1.Container{
		Name:            "postgres",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivEsc,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "postgres",
				ContainerPort: PostgresPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "POSTGRES_DB",
				Value: "postgres",
			},
			{
				Name:  "PGDATA",
				Value: DataMountPath + "/pgdata",
			},
			{
				Name: "POSTGRES_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: SecretName(cluster),
						},
						Key: "password",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      DataVolumeName,
				MountPath: DataMountPath,
			},
			{
				Name:      ConfigVolumeName,
				MountPath: ConfigMountPath,
			},
		},
		Resources: cluster.Spec.Resources,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"pg_isready", "-U", "postgres"},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			FailureThreshold:    6,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"pg_isready", "-U", "postgres"},
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       15,
			TimeoutSeconds:      5,
			FailureThreshold:    3,
		},
	}

	containers := []corev1.Container{pgContainer}

	// Append the metrics exporter sidecar when monitoring is enabled.
	if cluster.Spec.Monitoring != nil && cluster.Spec.Monitoring.Enabled {
		port := cluster.Spec.Monitoring.Port
		if port == 0 {
			port = 9187
		}
		containers = append(containers, corev1.Container{
			Name:            "metrics",
			Image:           ExporterImageTag(),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					Name:          "metrics",
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "DATA_SOURCE_NAME",
					Value: "postgresql://postgres:$(POSTGRES_PASSWORD)@localhost:5432/postgres?sslmode=disable",
				},
				{
					Name: "POSTGRES_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: SecretName(cluster),
							},
							Key: "password",
						},
					},
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		})
	}

	// PostgreSQL containers must not run as root and the upstream image
	// expects UID 999 / GID 999 (postgres user) for /var/lib/postgresql.
	// FSGroup ensures mounted PVCs are owned by the postgres group so the
	// process can write its data directory without an init container.
	runAsNonRoot := true
	runAsUser := int64(999)
	runAsGroup := int64(999)
	fsGroup := int64(999)
	podSpec := corev1.PodSpec{
		ServiceAccountName:            saName,
		Containers:                    containers,
		TopologySpreadConstraints:     cluster.Spec.TopologySpreadConstraints,
		Affinity:                      cluster.Spec.Affinity,
		Tolerations:                   cluster.Spec.Tolerations,
		ImagePullSecrets:              cluster.Spec.ImagePullSecrets,
		PriorityClassName:             cluster.Spec.PriorityClassName,
		TerminationGracePeriodSeconds: int64Ptr(30),
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
			RunAsGroup:   &runAsGroup,
			FSGroup:      &fsGroup,
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: ConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: ConfigMapName(cluster),
						},
					},
				},
			},
		},
	}

	// Determine storage access modes.
	accessModes := cluster.Spec.Storage.AccessModes
	if len(accessModes) == 0 {
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	pvcSpec := corev1.PersistentVolumeClaimSpec{
		AccessModes: accessModes,
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: cluster.Spec.Storage.Size,
			},
		},
		StorageClassName: cluster.Spec.Storage.StorageClass,
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterStatefulSetName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         HeadlessServiceName(cluster),
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   DataVolumeName,
						Labels: labels,
					},
					Spec: pvcSpec,
				},
			},
		},
	}

	return sts
}

// BuildPrimaryService constructs the Service that routes write traffic to the
// primary PostgreSQL instance.
func BuildPrimaryService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	labels := CommonLabels(cluster)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PrimaryServiceName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				LabelCluster: cluster.Name,
				LabelRole:    RolePrimary,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       PostgresPort,
					TargetPort: intstr.FromInt(PostgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// BuildReplicaService constructs the Service that routes read traffic across
// all replica instances.
func BuildReplicaService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	labels := CommonLabels(cluster)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReplicaServiceName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				LabelCluster: cluster.Name,
				LabelRole:    RoleReplica,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       PostgresPort,
					TargetPort: intstr.FromInt(PostgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// BuildHeadlessService constructs the headless Service required by the
// StatefulSet for stable DNS entries per pod.
func BuildHeadlessService(cluster *pgv1alpha1.PostgresCluster) *corev1.Service {
	labels := CommonLabels(cluster)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Selector:                 SelectorLabels(cluster),
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       PostgresPort,
					TargetPort: intstr.FromInt(PostgresPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// BuildConfigMap constructs the ConfigMap holding postgresql.conf and
// pg_hba.conf for the cluster.
func BuildConfigMap(cluster *pgv1alpha1.PostgresCluster) *corev1.ConfigMap {
	labels := CommonLabels(cluster)

	params := MergeParams(DefaultParams(), cluster.Spec.PostgresParameters)

	tlsEnabled := cluster.Spec.TLS != nil && cluster.Spec.TLS.Enabled
	postgresConf := BuildPostgresConf(params)
	hbaConf := BuildHBAConf(cluster.Spec.PostgresHBA, tlsEnabled)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"postgresql.conf": postgresConf,
			"pg_hba.conf":     hbaConf,
		},
	}
}

// BuildServiceAccount constructs a dedicated ServiceAccount for cluster pods.
func BuildServiceAccount(cluster *pgv1alpha1.PostgresCluster) *corev1.ServiceAccount {
	labels := CommonLabels(cluster)
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName(cluster),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}
}

// int64Ptr returns a pointer to the given int64 value.
func int64Ptr(i int64) *int64 { return &i }
