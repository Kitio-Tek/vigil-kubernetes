# Athos Kubernetes

[![CI](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/ci.yml)
[![Security](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/security.yml)
[![CodeQL](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/codeql.yml)
[![Trivy](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/trivy.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/trivy.yml)
[![Latest release](https://img.shields.io/github/v/release/Kitio-Tek/athos-kubernetes?sort=semver)](https://github.com/Kitio-Tek/athos-kubernetes/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/Kitio-Tek/athos-kubernetes)](https://goreportcard.com/report/github.com/Kitio-Tek/athos-kubernetes)
[![Go Reference](https://pkg.go.dev/badge/github.com/Kitio-Tek/athos-kubernetes.svg)](https://pkg.go.dev/github.com/Kitio-Tek/athos-kubernetes)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

Athos Kubernetes is a Kubernetes operator for PostgreSQL. It manages the full
lifecycle of PostgreSQL clusters, providing high availability, automated backups,
point-in-time recovery, and TLS encryption via Kubernetes-native Custom Resource
Definitions.

## Project Status

Athos is in active preview (0.x). The API surface, Helm values and CRD shape
may change between minor releases. See [SUPPORTED_VERSIONS.md](SUPPORTED_VERSIONS.md)
for the support matrix, [CHANGELOG.md](CHANGELOG.md) for release notes, and
[CHARTS.md](CHARTS.md) for the chart catalogue.

## Installation

### Prerequisites

- Kubernetes 1.25 or later
- Helm 3.x
- kubectl

### Installing with Helm

Add the chart repository:

```bash
helm repo add athos https://kitio-tek.github.io/athos-kubernetes
helm repo update
```

Install the operator into a dedicated namespace:

```bash
helm install athos-kubernetes athos/athos-kubernetes \
  --namespace athos-system \
  --create-namespace
```

Or install directly from this repository:

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

Athos Kubernetes creates three Services per cluster:

- `<name>-primary` routes write traffic to the current primary
- `<name>-replicas` routes read traffic across healthy replicas
- `<name>-pods` is a headless Service used for in-cluster DNS resolution
  of individual instances

The operator generates a Secret named `<name>-credentials` containing the
superuser password and a libpq URI. Read it with `kubectl get secret`:

```bash
PGPASSWORD=$(kubectl get secret my-cluster-credentials \
  -o jsonpath='{.data.password}' | base64 -d)

kubectl run psql --rm -i --restart=Never \
  --image postgres:16-alpine \
  --env PGPASSWORD="$PGPASSWORD" \
  -- psql -h my-cluster-primary -U postgres -d postgres -c 'SELECT version();'
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

## Quick Start (kind)

The repository ships everything needed to run Athos end-to-end against a local
kind cluster:

```bash
make kind-create
make docker-build IMG=athos-kubernetes:dev
make kind-load    IMG=athos-kubernetes:dev
make helm-install IMG=athos-kubernetes:dev

kubectl apply -f config/samples/pg_v1alpha1_postgrescluster.yaml
kubectl get pgc -A -w
```

To connect a throw-away psql client:

```bash
kubectl run psql --rm -it --image postgres:16-alpine -- \
  psql -h my-cluster-primary.default.svc.cluster.local -U postgres
```

Tear it back down with `make helm-uninstall && make kind-delete`.

## Testing

Athos ships four test layers, plus three supporting quality gates. Each
row maps one-to-one onto a Makefile target so contributors can reproduce
every CI gate locally.

### Test layers

| Layer | Command | What it covers |
|---|---|---|
| Unit | `make test` | Pure Go tests under `internal/`, `api/`, `internal/sqlescape`, controller helpers. |
| Integration (envtest) | `make test` | Controller behaviour against a real apiserver/etcd via `controller-runtime/tools/setup-envtest`. |
| End-to-end (Chainsaw) | `make e2e-test` | Default e2e suite, declarative `apiVersion: chainsaw.kyverno.io` tests against a live kind cluster. |
| End-to-end (KUTTL) | `make e2e-test-kuttl` | Legacy KUTTL suite kept for parity; same coverage as the Chainsaw suite. |

### Supporting gates

| Gate | Command | What it covers |
|---|---|---|
| Helm chart | `make helm-package && helm lint charts/athos-kubernetes` | Schema and template render. |
| Security | `make security` | `govulncheck` and `gosec` against the whole module. |
| Secret scan | `make gitleaks` | Scans the working tree and git history against `.gitleaks.toml`. |

The Chainsaw suites live under `tests/e2e/chainsaw/tests/` and follow the
[Kyverno Chainsaw](https://kyverno.github.io/chainsaw/) declarative
`Test` resource format. Each case is a directory with a `chainsaw-test.yaml`
defining apply/assert/error/cleanup steps. The default `make e2e-test`
target runs them all against the cluster pointed at by the current
kubectl context.

```bash
chainsaw test \
  --config tests/e2e/chainsaw/.chainsaw.yaml \
  tests/e2e/chainsaw/tests/
```

The legacy [KUTTL](https://github.com/kudobuilder/kuttl) suites live under
`tests/e2e/kuttl/tests/` and are exercised via `make e2e-test-kuttl`; they
remain for parity while we phase Chainsaw in across all scenarios.

Coverage profiles produced by `make test` are uploaded as `coverage-*.out`
artifacts on every CI run.

## Security

- See [SECURITY.md](SECURITY.md) for the disclosure process and supported
  versions.
- See [HARDENING.md](HARDENING.md) for the threat model and production
  hardening recommendations.
- Every push runs CodeQL, govulncheck, gosec, gitleaks and Trivy filesystem
  scans (see badges above). Container images are scanned by Trivy on every
  tag push via `trivy-image.yml`.
- SQL paths that quote untrusted identifiers route through
  `internal/sqlescape` and are unit-tested with adversarial fixtures.

## Roadmap

- Pluggable backup adapters (S3, GCS, Azure Blob) with retention policies.
- Continuous archiving with WAL streaming for tighter RPO.
- Validating + mutating admission webhooks for `PostgresCluster`.
- `pg_basebackup` over TLS to remote endpoints.
- Conformance suite against `cnpg.io` interoperability patterns.

See the GitHub Issues labeled `roadmap` for the live tracker.

## Project Maturity

- Apache-2.0 licensed (see [LICENSE](LICENSE)).
- DCO sign-off required on every commit (see [DCO.md](DCO.md)).
- Governance, maintainers, and support documented in
  [GOVERNANCE.md](GOVERNANCE.md), [MAINTAINERS.md](MAINTAINERS.md),
  [SUPPORT.md](SUPPORT.md).
- CI gates: lint, unit, integration, build, helm-lint, gitleaks, CodeQL,
  govulncheck, gosec and Trivy.
- Built with the [Operator SDK](https://sdk.operatorframework.io/) on top of
  [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).
- End-to-end tests are written against
  [Kyverno Chainsaw](https://github.com/kyverno/chainsaw) with a legacy
  [KUTTL](https://github.com/kudobuilder/kuttl) parity suite kept in tree
  during the migration window.

## Development

See [DEVELOPER.md](DEVELOPER.md) for local development setup, including how
to run the operator against a kind cluster and how to run unit and
integration tests.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community expectations.

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
