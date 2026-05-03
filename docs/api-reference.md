# API Reference

This document describes all custom resource definitions (CRDs) provided by Athos.

## Groups and Versions

All Athos resources belong to the `pg.athos.io/v1alpha1` API group.

---

## PostgresCluster

`PostgresCluster` defines a managed PostgreSQL cluster.

**Short names:** `pgc`, `pgcluster`

**Print columns:** Phase, Primary, Ready, Age

### Spec

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `postgresVersion` | int32 | yes | 16 | PostgreSQL major version. Supported: 14, 15, 16, 17. |
| `instances` | int32 | yes | 1 | Number of PostgreSQL instances (1-10). Use an odd number greater than 1 for HA. |
| `storage` | StorageSpec | yes | — | PVC configuration for the data volume. |
| `resources` | ResourceRequirements | no | — | CPU and memory requests/limits for PostgreSQL containers. |
| `postgresParameters` | map[string]string | no | — | postgresql.conf overrides. Keys and values are used verbatim. |
| `postgresHBA` | []string | no | — | Additional lines appended to pg_hba.conf. |
| `backup` | BackupSpec | no | — | Scheduled backup configuration. |
| `tls` | TLSSpec | no | enabled | TLS configuration for client connections. |
| `monitoring` | MonitoringSpec | no | enabled on 9187 | Prometheus metrics sidecar. |
| `topologySpreadConstraints` | []TopologySpreadConstraint | no | — | Pod scheduling spread constraints. |
| `affinity` | Affinity | no | — | Pod affinity and anti-affinity rules. |
| `tolerations` | []Toleration | no | — | Pod tolerations. |
| `imagePullSecrets` | []LocalObjectReference | no | — | Image pull secrets. |
| `serviceAccountName` | string | no | — | Existing ServiceAccount for cluster pods. If omitted, one is created. |
| `priorityClassName` | string | no | — | PriorityClass for cluster pods. |
| `paused` | bool | no | false | When true, the operator stops reconciling the cluster. |

#### StorageSpec

| Field | Type | Required | Description |
|---|---|---|---|
| `size` | Quantity | yes | PVC size, e.g. "10Gi". |
| `storageClass` | string | no | StorageClass name. Omit to use the cluster default. |
| `accessModes` | []PersistentVolumeAccessMode | no | Defaults to ReadWriteOnce. |

#### BackupSpec

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | false | Toggle scheduled backups. |
| `retentionPolicy` | string | "7d" | Duration to retain backups. |
| `schedule` | string | — | Cron expression (required when enabled=true). |
| `destination` | BackupDestinationSpec | — | Storage backend. |

#### BackupDestinationSpec

| Field | Type | Description |
|---|---|---|
| `s3` | S3BackupSpec | S3-compatible object store. |
| `gcs` | GCSBackupSpec | Google Cloud Storage. |

#### S3BackupSpec

| Field | Type | Description |
|---|---|---|
| `bucket` | string | Bucket name. |
| `region` | string | AWS region. |
| `endpoint` | string | Custom endpoint URL for S3-compatible stores (e.g. MinIO). |
| `path` | string | Path prefix within the bucket. |
| `credentialsSecret` | LocalObjectReference | Secret with AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. |

#### GCSBackupSpec

| Field | Type | Description |
|---|---|---|
| `bucket` | string | Bucket name. |
| `path` | string | Path prefix within the bucket. |
| `credentialsSecret` | LocalObjectReference | Secret with GCS JSON service account key. |

#### TLSSpec

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | true | Toggle TLS for client connections. |
| `certificateSecret` | LocalObjectReference | — | Secret with tls.crt and tls.key. |
| `caSecret` | LocalObjectReference | — | Secret with ca.crt for client certificate verification. |

#### MonitoringSpec

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | true | Toggle the postgres_exporter sidecar. |
| `port` | int32 | 9187 | Port for the /metrics endpoint. |

### Status

| Field | Type | Description |
|---|---|---|
| `phase` | string | Lifecycle phase: Initializing, Creating, Running, Degraded, Failed, Paused, Upgrading. |
| `readyInstances` | int32 | Number of ready StatefulSet pods. |
| `currentPrimary` | string | Pod name of the active primary. |
| `writeServiceName` | string | Name of the primary Service. |
| `readServiceName` | string | Name of the replica Service. |
| `postgresVersion` | string | Running PostgreSQL version string. |
| `observedGeneration` | int64 | Last fully reconciled generation. |
| `latestBackup` | Time | Timestamp of the last successful backup. |
| `conditions` | []Condition | Standard Kubernetes conditions. |

#### Condition Types

| Type | Description |
|---|---|
| `Ready` | True when all instances are ready and the cluster is operational. |
| `Available` | True when at least one instance is serving traffic. |
| `Progressing` | True while a transition is underway. |
| `Degraded` | True when the cluster is running with fewer instances than desired. |
| `PrimaryReady` | True when the primary instance is accepting connections. |

---

## PostgresBackup

`PostgresBackup` triggers a one-time backup of a cluster.

**Short names:** `pgb`, `pgbackup`

**Print columns:** Cluster, Phase, Started, Completed

### Spec

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `clusterName` | string | yes | — | Name of the PostgresCluster to back up. |
| `method` | string | no | basebackup | Backup method: `basebackup` or `pgdump`. |
| `online` | bool | no | true | Take the backup from a hot-standby. |

### Status

| Field | Type | Description |
|---|---|---|
| `phase` | string | Backup phase: Pending, Running, Completed, Failed. |
| `startTime` | Time | When the backup job started. |
| `completionTime` | Time | When the backup job finished. |
| `backupSize` | int64 | Backup size in bytes. |
| `destinationPath` | string | Final path where the backup is stored. |
| `conditions` | []Condition | Standard Kubernetes conditions. |

---

## PostgresUser

`PostgresUser` manages a PostgreSQL role and its database grants.

**Short names:** `pgu`, `pguser`

**Print columns:** Cluster, Applied, Age

### Spec

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `clusterName` | string | yes | — | Name of the target PostgresCluster. |
| `passwordSecret` | LocalObjectReference | no | — | Secret with a `password` key. |
| `databases` | []DatabaseGrant | no | — | Databases and privileges to grant. |
| `roles` | []string | no | — | PostgreSQL roles to grant to this user. |
| `superuser` | bool | no | false | Grant SUPERUSER. |
| `connectionLimit` | int32 | no | -1 | Maximum concurrent connections (-1 = unlimited). |

#### DatabaseGrant

| Field | Type | Description |
|---|---|---|
| `name` | string | Database name. |
| `privileges` | []string | Privilege keywords, e.g. "SELECT", "INSERT", "ALL PRIVILEGES". |

### Status

| Field | Type | Description |
|---|---|---|
| `applied` | bool | True when the user spec has been successfully applied to the database. |
| `observedGeneration` | int64 | Last reconciled generation. |
| `conditions` | []Condition | Standard Kubernetes conditions. |
