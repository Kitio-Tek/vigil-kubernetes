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

// Package statefulset provides utilities for building and managing
// Kubernetes StatefulSet resources for PostgreSQL clusters.
package statefulset

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	pgv1alpha1 "github.com/Kitio-Tek/vigil-kubernetes/api/v1alpha1"
	"github.com/Kitio-Tek/vigil-kubernetes/internal/postgres"
)

const (
	// DataVolumeName is the name of the PVC mounted at the postgres data directory.
	DataVolumeName = "pgdata"

	// DataMountPath is where the data volume is mounted inside the postgres container.
	DataMountPath = "/var/lib/postgresql/data"

	// WALVolumeName is the name of the optional dedicated WAL volume.
	WALVolumeName = "pgwal"

	// WALMountPath is where the WAL volume is mounted.
	WALMountPath = "/var/lib/postgresql/wal"

	// ConfigVolumeName holds postgresql.conf and pg_hba.conf rendered from ConfigMaps.
	ConfigVolumeName = "pgconfig"

	// ConfigMountPath is where the config volume is mounted.
	ConfigMountPath = "/etc/postgresql"

	// ScriptsVolumeName holds init/lifecycle scripts.
	ScriptsVolumeName = "pgscripts"

	// ScriptsMountPath is where operator scripts are mounted.
	ScriptsMountPath = "/usr/local/bin/vigil"

	// CertVolumeName holds TLS certificates when TLS is enabled.
	CertVolumeName = "pgcerts"

	// CertMountPath is where TLS certs are mounted.
	CertMountPath = "/etc/postgresql/tls"
)

// Options carries optional overrides for StatefulSet generation.
type Options struct {
	// Image overrides the default PostgreSQL image.
	Image string

	// InitContainers are extra init containers injected before the postgres one.
	InitContainers []corev1.Container

	// Sidecars are extra containers added alongside postgres (e.g. pgbackrest).
	Sidecars []corev1.Container

	// ExtraVolumes are additional volumes to mount.
	ExtraVolumes []corev1.Volume

	// ExtraVolumeMounts are additional mounts added to the postgres container.
	ExtraVolumeMounts []corev1.VolumeMount

	// TLSSecretName is the name of the Secret holding server.crt / server.key.
	// If non-empty, a cert volume is added and TLS is configured.
	TLSSecretName string

	// ConfigMapName is the name of the ConfigMap holding postgresql.conf.
	ConfigMapName string

	// ScriptsConfigMapName is the ConfigMap holding lifecycle scripts.
	ScriptsConfigMapName string
}

// Build constructs the desired StatefulSet for a PostgresCluster.
func Build(cluster *pgv1alpha1.PostgresCluster, opts Options) *appsv1.StatefulSet {
	image := opts.Image
	if image == "" {
		image = postgres.PostgresImageTag(cluster.Spec.PostgresVersion)
	}

	replicas := cluster.Spec.Instances
	labels := postgres.CommonLabels(cluster)
	selectorLabels := postgres.SelectorLabels(cluster)

	containers := buildContainers(cluster, image, opts)
	initContainers := opts.InitContainers
	volumes := buildVolumes(opts)
	pvcs := buildPVCs(cluster)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:          postgres.HeadlessServiceName(cluster),
			Replicas:             &replicas,
			PodManagementPolicy:  appsv1.OrderedReadyPodManagement,
			UpdateStrategy:       rollingUpdate(),
			Selector:             &metav1.LabelSelector{MatchLabels: selectorLabels},
			VolumeClaimTemplates: pvcs,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: corev1.PodSpec{
					InitContainers:   initContainers,
					Containers:       containers,
					Volumes:          volumes,
					SecurityContext:  defaultPodSecurityContext(),
					ImagePullSecrets: cluster.Spec.ImagePullSecrets,
					Affinity:         buildAffinity(cluster),
					Tolerations:      cluster.Spec.Tolerations,
				},
			},
		},
	}

	return sts
}

func buildContainers(
	cluster *pgv1alpha1.PostgresCluster,
	image string,
	opts Options,
) []corev1.Container {
	mounts := []corev1.VolumeMount{
		{Name: DataVolumeName, MountPath: DataMountPath},
	}
	if opts.ConfigMapName != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: ConfigMountPath,
		})
	}
	if opts.ScriptsConfigMapName != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      ScriptsVolumeName,
			MountPath: ScriptsMountPath,
		})
	}
	if opts.TLSSecretName != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      CertVolumeName,
			MountPath: CertMountPath,
			ReadOnly:  true,
		})
	}
	mounts = append(mounts, opts.ExtraVolumeMounts...)

	pg := corev1.Container{
		Name:            "postgres",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{Name: "postgres", ContainerPort: 5432, Protocol: corev1.ProtocolTCP},
		},
		VolumeMounts:    mounts,
		Resources:       cluster.Spec.Resources,
		SecurityContext: defaultContainerSecurityContext(),
		LivenessProbe:   livenessProbe(),
		ReadinessProbe:  readinessProbe(),
		Env:             baseEnv(cluster),
	}

	containers := []corev1.Container{pg}
	containers = append(containers, opts.Sidecars...)
	return containers
}

func buildVolumes(opts Options) []corev1.Volume {
	var volumes []corev1.Volume

	if opts.ConfigMapName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: ConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: opts.ConfigMapName},
				},
			},
		})
	}

	if opts.ScriptsConfigMapName != "" {
		mode := int32(0755)
		volumes = append(volumes, corev1.Volume{
			Name: ScriptsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: opts.ScriptsConfigMapName},
					DefaultMode:          &mode,
				},
			},
		})
	}

	if opts.TLSSecretName != "" {
		mode := int32(0640)
		volumes = append(volumes, corev1.Volume{
			Name: CertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  opts.TLSSecretName,
					DefaultMode: &mode,
				},
			},
		})
	}

	volumes = append(volumes, opts.ExtraVolumes...)
	return volumes
}

func buildPVCs(cluster *pgv1alpha1.PostgresCluster) []corev1.PersistentVolumeClaim {
	pvcs := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: DataVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: cluster.Spec.Storage.Size,
					},
				},
				StorageClassName: cluster.Spec.Storage.StorageClass,
			},
		},
	}

	return pvcs
}

func rollingUpdate() appsv1.StatefulSetUpdateStrategy {
	return appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			MaxUnavailable: func() *intstr.IntOrString {
				v := intstr.FromInt(1)
				return &v
			}(),
		},
	}
}

func defaultPodSecurityContext() *corev1.PodSecurityContext {
	uid := int64(999)
	gid := int64(999)
	return &corev1.PodSecurityContext{
		RunAsUser:  &uid,
		RunAsGroup: &gid,
		FSGroup:    &gid,
	}
}

func defaultContainerSecurityContext() *corev1.SecurityContext {
	readOnly := false
	allowPrivEscalation := false
	return &corev1.SecurityContext{
		ReadOnlyRootFilesystem:   &readOnly,
		AllowPrivilegeEscalation: &allowPrivEscalation,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

func livenessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"pg_isready", "-U", "postgres"},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    6,
	}
}

func readinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"pg_isready", "-U", "postgres"},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		FailureThreshold:    3,
	}
}

func baseEnv(cluster *pgv1alpha1.PostgresCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "POSTGRES_DB", Value: cluster.Name},
		{Name: "PGDATA", Value: DataMountPath + "/pgdata"},
	}
}

func buildAffinity(cluster *pgv1alpha1.PostgresCluster) *corev1.Affinity {
	if cluster.Spec.Affinity != nil {
		return cluster.Spec.Affinity
	}
	// Default: spread pods across nodes to avoid single points of failure.
	if cluster.Spec.Instances <= 1 {
		return nil
	}
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: postgres.SelectorLabels(cluster),
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

// DefaultStorage returns a 1Gi storage request used in tests and defaults.
func DefaultStorage() resource.Quantity {
	return resource.MustParse("1Gi")
}

// OrdinalFromName extracts the pod ordinal from a StatefulSet pod name.
// Returns -1 if the name does not match the expected pattern.
func OrdinalFromName(podName, stsName string) int {
	prefix := stsName + "-"
	if len(podName) <= len(prefix) {
		return -1
	}
	suffix := podName[len(prefix):]
	ordinal := 0
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return -1
		}
		ordinal = ordinal*10 + int(c-'0')
	}
	return ordinal
}
