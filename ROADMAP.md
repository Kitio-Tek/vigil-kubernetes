# Athos Roadmap

This file is the public, honest view of what Athos ships today, what is in
flight, and what is queued. It is intentionally compared against the
three production-grade peer operators that share the most overlap with
Athos: [CloudNativePG](https://github.com/cloudnative-pg/cloudnative-pg),
[Crunchy PGO](https://github.com/CrunchyData/postgres-operator) and
[Zalando postgres-operator](https://github.com/zalando/postgres-operator).

The peer comparison is meant to make scope decisions transparent, not to
claim feature parity. A row marked "shipped" means the matching reconciler
path is exercised by unit / envtest / Chainsaw e2e tests in this repo.

## Feature matrix

Legend: `shipped` = on `main` and tested; `partial` = code paths exist but
not yet end-to-end on every documented surface; `planned` = listed below
with an owning milestone; `not in scope` = explicitly out of scope for the
0.x preview.

| Capability | Athos 0.9.x | CNPG | Crunchy | Zalando |
|---|---|---|---|---|
| StatefulSet-backed PostgreSQL provisioning | shipped | shipped | shipped | shipped |
| Per-cluster credentials Secret + ServiceAccount + PDB | shipped | shipped | shipped | shipped |
| Read / write / pods Services (primary, replicas, headless) | shipped | shipped | shipped | shipped |
| Pause / resume reconciliation | shipped | shipped | shipped | shipped |
| Prometheus `postgres_exporter` sidecar | shipped | shipped | shipped | shipped |
| PgBouncer connection pooler | shipped | shipped | shipped | shipped |
| Database / role / grant management (`PostgresUser`) | shipped | shipped (`Database`) | shipped (`PostgresUser`) | shipped |
| TLS-aware `pg_hba.conf` rendering | shipped | shipped | shipped | shipped |
| On-demand `pg_basebackup` / `pg_dump` Job | shipped | shipped (`Backup`) | shipped (`pgBackRestBackup`) | shipped |
| Streaming replication primary -> replica | partial (helpers in `internal/replication`, not yet wired to replica pods) | shipped | shipped | shipped |
| Automated failover orchestration | partial (helpers in `internal/ha`, no controller loop) | shipped | shipped | shipped (Patroni) |
| S3 / GCS / Azure backup adapters | planned (0.10) | shipped (Barman Cloud) | shipped (pgBackRest) | shipped (WAL-G) |
| Continuous WAL archiving + PITR | planned (0.11) | shipped | shipped | shipped |
| Validating / mutating admission webhook | planned (0.10) | shipped | shipped | shipped |
| Cert-manager `Certificate` integration | planned (0.10) | shipped | shipped | partial |
| Major-version upgrades (`pg_upgrade`) | planned (0.12) | shipped | shipped | shipped |
| Read replica routing by lag / topology | planned (0.12) | shipped | partial | partial |
| Synchronous replication groups | planned (0.13) | shipped | shipped | shipped |
| Pluggable operator extensions | not in scope (0.x) | shipped (CNPG plugins) | partial | not in scope |
| OLM bundle | shipped (`config/manifests/`) | shipped | shipped | shipped |
| Helm chart museum (`gh-pages`) | shipped | shipped | shipped | shipped |

## Milestones

The milestones are intentionally narrow so each one ships a deployable,
e2e-tested unit.

### 0.10 - backup storage + admission gates

Goal: turn `PostgresBackup` from a Job-runner into a backup product, and
stop accepting obviously broken CRs.

- [ ] Backup destination adapters: S3, GCS, Azure Blob, with per-destination
      credentials Secret and retention policy.
- [ ] `PostgresBackup.status` tracks bytes written, duration, and an
      `EvidenceRef` pointing at the storage URL.
- [ ] cert-manager `Certificate` integration: when `spec.tls.issuerRef` is
      set, the operator creates and mounts the cert.
- [ ] ValidatingAdmissionPolicy / validating webhook for `PostgresCluster`
      (version range, storage size minimums, instance count caps).
- [ ] Chainsaw e2e: backup-to-S3 with a kind-hosted MinIO fixture.

### 0.11 - failover + PITR

Goal: claim "production-grade" credibly.

- [ ] Wire `internal/replication` into the StatefulSet builder so replica
      pods join the primary via `primary_conninfo` and a per-replica slot.
- [ ] Continuous WAL archiving to the same backup destination.
- [ ] `PostgresRecovery` CRD for point-in-time recovery into a target
      cluster, with namespace remapping.
- [ ] HA orchestration loop: detect primary loss, promote the
      `FailoverCandidate` from `internal/ha`, update the `*-primary`
      Service endpoint slice.
- [ ] Chainsaw e2e: kill the primary pod and assert the cluster recovers
      with the replica promoted within an SLO.

### 0.12 - upgrades + read routing

Goal: cover the operational surface most users hit second.

- [ ] Major-version upgrade workflow with `pg_upgrade --link`, gated by a
      `spec.upgrade.allow` window.
- [ ] Read-only routing: route the `*-replicas` Service to the replica
      with the lowest replication lag.
- [ ] Topology-aware read replicas (zone affinity / anti-affinity policy
      knobs on `PostgresCluster.spec.topologySpread`).

### 0.13+ - replication groups + ecosystem

- [ ] Synchronous replication groups with quorum settings.
- [ ] Operator metrics on `controller_runtime_reconcile_total{result}`,
      `athos_cluster_phase`, `athos_backup_duration_seconds`.
- [ ] OperatorHub catalog submission.
- [ ] Public end-user docs site (mkdocs Material).

## Inspiration / cross-references

Where Athos borrows ideas from peers, the issue or PR referencing them
goes here so reviewers can trace the lineage:

- Backup-destination CRD shape modeled on CNPG `BackupConfig`.
- Per-instance replication slot naming follows Crunchy PGO's convention
  to avoid the dropped-slot problem when a replica is recreated.
- HA failover SLO measurement is inspired by Zalando's Patroni
  `failsafe_mode` thresholds.

## Non-goals (for the 0.x preview)

- No support for multi-cluster federation across regions.
- No bundled monitoring / alerting (use the ServiceMonitor + your own Prometheus).
- No bundled Postgres extensions (the operator does not install
  pg_partman, pgvector, etc.; users bring their own image).
- No managed-service connector (RDS, Cloud SQL, Aiven, etc.).

## Tracking

Roadmap items are mirrored to GitHub Issues labeled `roadmap`; status
changes here are reflected in `CHANGELOG.md` on every release.
