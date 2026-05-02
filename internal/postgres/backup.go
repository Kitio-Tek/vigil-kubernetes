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
	pgv1alpha1 "github.com/Kitio-Tek/vigil/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// backupImage is the container image used to run pg_basebackup.
	backupImage = "docker.io/library/postgres:16-alpine"
	// pgdumpImage is the container image used to run pg_dump.
	pgdumpImage = "docker.io/library/postgres:16-alpine"
)

// BuildBackupJob constructs a Kubernetes Job that performs a backup of the
// given PostgresCluster using the method specified in the PostgresBackup spec.
func BuildBackupJob(
	backup *pgv1alpha1.PostgresBackup,
	cluster *pgv1alpha1.PostgresCluster,
) *batchv1.Job {
	backupLabels := map[string]string{
		LabelCluster:   cluster.Name,
		LabelManagedBy: OperatorName,
		"pg.vigil.io/backup": backup.Name,
	}

	image := PostgresImageTag(cluster.Spec.PostgresVersion)
	var command []string

	primaryHost := PrimaryServiceName(cluster) + "." + cluster.Namespace + ".svc.cluster.local"

	switch backup.Spec.Method {
	case pgv1alpha1.BackupMethodPgDump:
		command = []string{
			"sh", "-c",
			"pg_dump -h " + primaryHost + " -U postgres -Fc -f /backup/dump.pgdump postgres",
		}
	default:
		// basebackup is the default method.
		command = []string{
			"sh", "-c",
			"pg_basebackup -h " + primaryHost + " -U postgres -D /backup -Fp -Xs -P",
		}
	}

	backoffLimit := int32(3)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackupJobName(backup),
			Namespace: backup.Namespace,
			Labels:    backupLabels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: backupLabels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            "backup",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         command,
							Env: []corev1.EnvVar{
								{
									Name: "PGPASSWORD",
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
									Name:      "backup-storage",
									MountPath: "/backup",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "backup-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// Inject S3 credentials if configured on the cluster backup destination.
	if cluster.Spec.Backup != nil &&
		cluster.Spec.Backup.Destination != nil &&
		cluster.Spec.Backup.Destination.S3 != nil {
		s3 := cluster.Spec.Backup.Destination.S3
		if s3.CredentialsSecret != nil {
			job.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{
				{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: *s3.CredentialsSecret,
					},
				},
			}
		}
		if s3.Endpoint != "" {
			job.Spec.Template.Spec.Containers[0].Env = append(
				job.Spec.Template.Spec.Containers[0].Env,
				corev1.EnvVar{Name: "AWS_ENDPOINT_URL", Value: s3.Endpoint},
			)
		}
	}

	_ = image
	_ = pgdumpImage
	_ = backupImage

	return job
}
