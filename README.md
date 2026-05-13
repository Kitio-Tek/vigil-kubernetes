# Athos Kubernetes

[![CI](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/ci.yml)
[![E2E](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/e2e.yml)
[![Security](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/security.yml)
[![CodeQL](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/codeql.yml)
[![Trivy](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/trivy.yml/badge.svg?branch=main)](https://github.com/Kitio-Tek/athos-kubernetes/actions/workflows/trivy.yml)
[![Latest release](https://img.shields.io/github/v/release/Kitio-Tek/athos-kubernetes?sort=semver)](https://github.com/Kitio-Tek/athos-kubernetes/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/Kitio-Tek/athos-kubernetes)](https://goreportcard.com/report/github.com/Kitio-Tek/athos-kubernetes)
[![Go Reference](https://pkg.go.dev/badge/github.com/Kitio-Tek/athos-kubernetes.svg)](https://pkg.go.dev/github.com/Kitio-Tek/athos-kubernetes)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

Athos Kubernetes is a Kubernetes operator for PostgreSQL. It manages the
full lifecycle of `PostgresCluster`, `PostgresBackup`, `PostgresUser` and
`PostgresPooler` custom resources, built on the
[Operator SDK](https://sdk.operatorframework.io/).

Athos is in active preview (0.x); see
[SUPPORTED_VERSIONS.md](SUPPORTED_VERSIONS.md) for the support matrix.

## Install

```bash
helm repo add athos https://kitio-tek.github.io/athos-kubernetes
helm repo update
helm install athos-kubernetes athos/athos-kubernetes \
  --namespace athos-system --create-namespace
```

## Create a cluster

```yaml
apiVersion: pg.athos.io/v1alpha1
kind: PostgresCluster
metadata:
  name: my-cluster
spec:
  postgresVersion: 16
  instances: 1
  storage:
    size: 10Gi
```

More samples live in [`examples/`](examples/). The full API surface is
documented in [docs/api-reference.md](docs/api-reference.md).

## Documentation

| Topic | Where |
|---|---|
| API reference (`PostgresCluster`, `PostgresBackup`, `PostgresUser`) | [docs/api-reference.md](docs/api-reference.md) |
| Helm chart values, museum, upgrade flow | [CHARTS.md](CHARTS.md) |
| Roadmap  | [ROADMAP.md](ROADMAP.md) |
| Release notes | [CHANGELOG.md](CHANGELOG.md) |
| Local development, kind quick-start, generated code | [DEVELOPER.md](DEVELOPER.md) |
| Contributing, test framework ([Chainsaw](https://github.com/kyverno/chainsaw) / [KUTTL](https://github.com/kudobuilder/kuttl)), commit style | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Security disclosure process and supported versions | [SECURITY.md](SECURITY.md) |
| Production hardening and threat model | [HARDENING.md](HARDENING.md) |
| Governance, maintainers, support channels | [GOVERNANCE.md](GOVERNANCE.md), [MAINTAINERS.md](MAINTAINERS.md), [SUPPORT.md](SUPPORT.md) |
| Code of Conduct and DCO sign-off | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md), [DCO.md](DCO.md) |

## License

Apache License 2.0. See [LICENSE](LICENSE).
