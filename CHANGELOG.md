# Changelog

All notable changes to the Athos Kubernetes operator are recorded in this
file. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0] - 2026-05-04

### Security

- Hardened `internal/userstate.Plan` and the PostgresUser controller's
  `buildUserSQL` against SQL injection: every identifier is escaped via
  `internal/sqlescape.Identifier`, every string literal via
  `StringLiteral`, and identifier validity is enforced before any
  interpolation. Rejects passwords containing NUL bytes.
- Whitelisted the SQL privileges accepted by `buildUserSQL` so a
  malformed CRD value cannot smuggle SQL fragments through the
  `Databases[].Privileges` field.
- Redacted `PASSWORD '<value>'` literals from psql stderr before they
  surface in Kubernetes Conditions or operator logs.
- Cluster pods now run with a non-root SecurityContext (`RunAsUser=999`,
  `RunAsGroup=999`, `FSGroup=999`, `RuntimeDefault` seccomp). The
  postgres container drops every Linux capability and forbids
  privilege escalation.

### Added

- Dependabot config covering `gomod`, `github-actions`, and `docker`
  ecosystems, with grouped PRs for kubernetes, controller-runtime,
  and ginkgo families.
- Auto-merge workflow that approves and squash-merges Dependabot
  patch / minor PRs and comments on major bumps.
- PR template and issue templates under `.github/`.
- `make security` and `make govulncheck` targets, plus a Security CI
  job that runs `govulncheck` on every push.
- Cluster reconciler now emits Kubernetes Events on credentials
  Secret / ConfigMap / StatefulSet creation and on scale operations,
  and a Warning event when reconciliation is paused.
- `PodDisruptionBudget` reconciliation tied to the cluster, with
  HA-aware sizing.

### Changed

- CI Go toolchain bumped from 1.24.2 to 1.25.1 to clear the stdlib
  advisories surfaced by govulncheck.
- Lint CI step now runs `make verify-fmt` before `golangci-lint`.

## [0.6.0] - 2026-05-03

### Added

- `LICENSE` Apache-2.0 file at the repository root.
- `CHANGELOG.md` following Keep-a-Changelog conventions.
- README badges for CI, latest release, Go report card, license.
- `internal/sidecar` builders for the postgres-exporter metrics
  container and a wal-g WAL uploader container.
- `internal/storageclass` PVC template builder and expansion checks.
- `internal/resourcerequests` CPU / memory defaults plus a Merge helper.
- `internal/poolerconfig` renderer for `pgbouncer.ini` and
  `userlist.txt`.

### Changed

- The PostgresCluster admission webhook now validates
  `spec.backup.schedule` via `internal/cronexpr` instead of an inline
  regex.

## [0.5.0] - 2026-05-03

### Fixed

- The cluster reconciler now generates a 32-character superuser password
  and writes it to a `<cluster>-credentials` Secret before the StatefulSet
  is rendered. Pods previously failed to start with
  `CreateContainerConfigError` because the StatefulSet referenced a
  Secret that no controller produced.
- Helm CRD templates are now wrapped in `{{- if .Values.crds.install }}`
  so installs against clusters that already have the CRDs (out-of-band
  install, multi-tenant clusters) do not collide.

### Added

- `make verify-fmt` and `make pre-commit` Makefile targets that fail fast
  on any un-formatted Go file. The CI lint job calls `verify-fmt` before
  `golangci-lint` for clearer error messages.
- Controller envtest cases asserting that the credentials Secret is
  created on first reconcile and that subsequent reconciles preserve a
  rotated password.

### Changed

- README "Connecting to the Cluster" section now matches what the
  operator produces: three Services per cluster, the credentials Secret,
  and a `kubectl run psql ...` example verified end-to-end against a
  local kind cluster.

## [0.4.0] - 2026-05-03

### Added

- Build identity baked into the manager binary via `-ldflags`. The
  manager logs version, commit, build date, Go toolchain and platform
  at startup; values are exposed via `internal/version`.
- `internal/backoff` retry schedules (exponential with cap, constant)
  and a `Retry` helper.
- `internal/validation` field-level validators producing aggregated
  errors for the admission webhooks.

## [0.3.0] - 2026-05-03

### Changed

- **Project rename**: module is now
  `github.com/Kitio-Tek/athos-kubernetes` and the CRD group is
  `pg.athos.io`. Existing v1alpha1 schemas are preserved.

### Added

- In-process event bus (`internal/eventbus`) so reconcilers can broadcast
  lifecycle hints without going through the API server.
- Health-check primitives with a precedence rule that respects critical
  checks.
- Probe builders for liveness, readiness and startup, plus PgBouncer
  port helpers used by the pooler controller.

## [0.2.0] - 2026-05-03

### Added

- PostgresPooler controller for an opt-in PgBouncer deployment fronting
  the cluster's read-write Service.
- NetworkPolicy builders restricting cluster ingress to namespace-local
  workloads, plus a separate replication-traffic NetworkPolicy.
- CSI VolumeSnapshot support: the `internal/snapshot` package builds
  and prunes snapshots based on a count + age retention policy.
- WAL archive helpers with archive_command/restore_command builders for
  wal-g and barman-cloud against S3, GCS, Azure and local filesystem
  endpoints.
- `kubectl-athos` plugin scaffold under `cmd/plugin`.

### Fixed

- Container build copies the full `internal/` tree so all imports
  resolve at build time.

## [0.1.0] - 2026-05-03

### Added

- PostgresCluster CRD with a reconciler that owns StatefulSet, Services
  (rw/ro/headless), ConfigMap and ServiceAccount sub-resources.
- PostgresBackup CRD with Job-based backup execution for both
  pg_basebackup and pg_dump methods.
- PostgresUser CRD with role and grant management against the primary
  instance.
- Helm chart `charts/athos-kubernetes` bundling CRDs, RBAC and the
  Deployment manifest.

[Unreleased]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Kitio-Tek/athos-kubernetes/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Kitio-Tek/athos-kubernetes/releases/tag/v0.1.0
