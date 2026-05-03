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

// Package sidecar builds the optional containers that run alongside the
// main postgres container: the postgres-exporter for Prometheus metrics
// and the wal-archive uploader.
package sidecar

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Default images. The values are picked to match the upstream's
// ghcr-published tags so a fresh install does not need a private registry.
const (
	DefaultExporterImage = "ghcr.io/prometheus-community/postgres-exporter:v0.15.0"
	DefaultWalUploader   = "ghcr.io/wal-g/wal-g:v3.0.0"
)

// ExporterPort is the port the postgres-exporter listens on.
const ExporterPort int32 = 9187

// ExporterSpec captures the inputs the operator needs to render the
// Prometheus exporter sidecar.
type ExporterSpec struct {
	Image            string
	DataSourceSecret string
	DataSourceKey    string
}

// ExporterContainer returns the corev1.Container used as the metrics
// exporter sidecar.
func ExporterContainer(s ExporterSpec) corev1.Container {
	if s.Image == "" {
		s.Image = DefaultExporterImage
	}
	if s.DataSourceKey == "" {
		s.DataSourceKey = "uri"
	}
	return corev1.Container{
		Name:            "exporter",
		Image:           s.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{{
			Name:          "metrics",
			ContainerPort: ExporterPort,
			Protocol:      corev1.ProtocolTCP,
		}},
		Env: []corev1.EnvVar{{
			Name: "DATA_SOURCE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: s.DataSourceSecret},
					Key:                  s.DataSourceKey,
				},
			},
		}},
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
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(ExporterPort)},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
		},
	}
}

// WALUploaderSpec describes a WAL uploader sidecar.
type WALUploaderSpec struct {
	Image       string
	BucketURL   string
	CredsSecret string
}

// WALUploaderContainer returns the wal-g uploader sidecar that streams
// WAL segments to the configured object store.
func WALUploaderContainer(s WALUploaderSpec) (corev1.Container, error) {
	if s.BucketURL == "" {
		return corev1.Container{}, fmt.Errorf("sidecar: bucket URL is required")
	}
	if s.Image == "" {
		s.Image = DefaultWalUploader
	}
	env := []corev1.EnvVar{
		{Name: "WALG_S3_PREFIX", Value: s.BucketURL},
	}
	if s.CredsSecret != "" {
		env = append(env,
			corev1.EnvVar{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: s.CredsSecret},
						Key:                  "access-key-id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: s.CredsSecret},
						Key:                  "secret-access-key",
					},
				},
			},
		)
	}
	return corev1.Container{
		Name:            "wal-uploader",
		Image:           s.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/bin/wal-g", "wal-push-stream"},
		Env:             env,
	}, nil
}
