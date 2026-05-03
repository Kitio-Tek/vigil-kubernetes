# Athos Kubernetes

Athos Kubernetes is a Kubernetes operator for PostgreSQL. It manages the full lifecycle
of PostgreSQL clusters, providing high availability, automated backups, point-in-time
recovery, and TLS encryption via Kubernetes-native Custom Resource Definitions.

## Installation

### Prerequisites

- Kubernetes 1.25 or later
- Helm 3.x
- kubectl

### Installing with Helm

Install the operator into a dedicated namespace:

```bash
helm install athos-kubernetes charts/athos-kubernetes/ \
  --namespace athos-system \
  --create-namespace
```

Verify the operator pod is running:

```bash
kubectl get pods -n athos-system
```

## Configuration

### Creating a PostgreSQL Cluster

Apply a `PostgresCluster` resource:

```yaml
apiVersion: pg.athos.io/v1alpha1
kind: PostgresCluster
metadata:
  name: my-cluster
  namespace: default
spec:
  postgresVersion: 16
  instances: 1
  storage:
    size: 10Gi
```

```bash
kubectl apply -f cluster.yaml
kubectl get pgc -n default
kubectl describe pgc my-cluster -n default
```

### Connecting to the Cluster

Athos Kubernetes creates two Services per cluster:

- `<name>-primary` routes write traffic to the current primary
- `<name>-replicas` routes read traffic across healthy replicas

```bash
kubectl run psql --rm -it --image postgres:16-alpine -- \
  psql -h my-cluster-primary.default.svc.cluster.local -U postgres
```

### Taking a Backup

```yaml
apiVersion: pg.athos.io/v1alpha1
kind: PostgresBackup
metadata:
  name: my-backup
  namespace: default
spec:
  clusterName: my-cluster
  method: basebackup
```

### Managing Database Users

```yaml
apiVersion: pg.athos.io/v1alpha1
kind: PostgresUser
metadata:
  name: app-user
  namespace: default
spec:
  clusterName: my-cluster
  passwordSecret:
    name: app-user-password
  databases:
    - name: myapp
      privileges:
        - SELECT
        - INSERT
        - UPDATE
```

## API Reference

### PostgresCluster

| Field | Type | Description |
|---|---|---|
| `spec.postgresVersion` | int32 | PostgreSQL major version (14-17, default 16) |
| `spec.instances` | int32 | Number of instances (1-10, default 1) |
| `spec.storage.size` | Quantity | PVC size, e.g. "10Gi" |
| `spec.storage.storageClass` | string | StorageClass name |
| `spec.resources` | ResourceRequirements | CPU/memory requests and limits |
| `spec.backup` | BackupSpec | Scheduled backup configuration |
| `spec.tls` | TLSSpec | TLS configuration |
| `spec.monitoring` | MonitoringSpec | Prometheus metrics configuration |
| `spec.paused` | bool | Suspend reconciliation |

### PostgresBackup

| Field | Type | Description |
|---|---|---|
| `spec.clusterName` | string | Name of the PostgresCluster to back up |
| `spec.method` | string | basebackup (default) or pgdump |
| `spec.online` | bool | Take backup from hot standby |

### PostgresUser

| Field | Type | Description |
|---|---|---|
| `spec.clusterName` | string | Name of the PostgresCluster |
| `spec.passwordSecret` | LocalObjectReference | Secret with "password" key |
| `spec.databases` | []DatabaseGrant | Databases and privilege grants |
| `spec.roles` | []string | PostgreSQL roles to grant |
| `spec.superuser` | bool | Grant superuser privileges |
| `spec.connectionLimit` | int32 | Maximum connections (-1 for unlimited) |

## Architecture

Athos Kubernetes consists of three reconcilers:

**PostgresCluster** manages StatefulSets, Services, ConfigMaps, and ServiceAccounts
for each database cluster. Every reconcile cycle drives the cluster toward the desired
state expressed in the spec, updating the status with the current phase, ready instance
count, and primary pod name.

**PostgresBackup** manages Kubernetes Jobs that perform physical (pg_basebackup) or
logical (pg_dump) backups. Backups are immutable once they reach a terminal state
(Completed or Failed).

**PostgresUser** manages PostgreSQL roles and database-level grants by executing SQL
directly against the primary instance using the Kubernetes exec API. The Applied status
field reflects whether the last reconcile succeeded.

Each reconciler sets owner references on all managed resources so that Kubernetes garbage
collection removes them when the parent CR is deleted.

## Features

- Automated PostgreSQL cluster provisioning
- Primary/replica streaming replication
- Automated failover
- Scheduled and on-demand backups (S3, GCS)
- Point-in-time recovery
- TLS encryption for client connections
- Prometheus metrics via postgres_exporter sidecar
- Database user management via PostgresUser CRD
- Rolling updates
- Pause/resume reconciliation
- Topology spread constraints and affinity rules

## Development

See [DEVELOPER.md](DEVELOPER.md) for local development setup, including how to run the
operator against a kind cluster and how to run unit and integration tests.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

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
