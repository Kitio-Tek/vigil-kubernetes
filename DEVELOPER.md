# Developer Guide

This guide covers setting up a development environment, building the operator,
running tests, and debugging locally.

## Prerequisites

- Go 1.22 or later
- Docker or Podman
- kubectl
- Helm 3.x
- kind (for local cluster testing)
- operator-sdk v1.38.0

## Repository Structure

```
athos/
  api/v1alpha1/           Custom resource type definitions
  cmd/main.go             Operator entry point
  config/                 Kustomize manifests
  charts/athos/           Helm chart
  internal/
    controller/           Reconciler implementations
    postgres/             PostgreSQL-specific helpers (naming, config, resources)
  tests/e2e/kuttl/        KUTTL end-to-end tests
  .github/workflows/      CI/CD pipelines
```

## Setting Up the Development Environment

Clone the repository and download dependencies:

```bash
git clone git@github.com:Kitio-Tek/athos.git
cd athos
go mod download
```

Install the code generation tool:

```bash
make controller-gen
```

## Building

Generate code and build the manager binary:

```bash
make generate
make manifests
make build
```

The binary is written to `bin/manager`.

To build the container image:

```bash
make docker-build IMG=ghcr.io/kitio-tek/athos:dev
```

## Running the Operator Locally

Install the CRDs into your current kubectl context:

```bash
make install
```

Run the operator outside the cluster (useful for rapid iteration):

```bash
make run
```

The operator connects to the cluster using your active kubeconfig and watches all
namespaces.

## Testing

### Unit Tests

Unit tests for the internal packages do not require a running cluster:

```bash
go test ./internal/postgres/... -v
```

### Integration Tests (envtest)

Controller integration tests use envtest to spin up a real Kubernetes API server
without any node infrastructure:

```bash
make test
```

The first run downloads the envtest binaries to `bin/`. Subsequent runs use the
cached binaries.

### End-to-End Tests with KUTTL

E2E tests require a running Kubernetes cluster and the operator deployed inside it.
Using kind:

```bash
make kind-create
make docker-build IMG=ghcr.io/kitio-tek/athos:dev
make kind-load IMG=ghcr.io/kitio-tek/athos:dev
make install
kubectl apply -k config/manager/
make e2e-test
```

## Code Generation

After modifying any `*_types.go` file, regenerate:

```bash
make generate    # DeepCopy methods
make manifests   # CRD YAML, RBAC, webhook configs
```

Do not manually edit `zz_generated.deepcopy.go` or the files under `config/crd/bases/`.

## Adding a New API Type

1. Run `operator-sdk create api --group pg --version v1alpha1 --kind MyType`
2. Edit `api/v1alpha1/mytype_types.go` with your spec and status fields
3. Run `make generate && make manifests`
4. Implement the reconciler in `internal/controller/`
5. Register the reconciler in `cmd/main.go`
6. Write integration tests in `internal/controller/`

## Debugging

Enable verbose controller logging by passing `--zap-log-level=debug` to the manager:

```bash
go run ./cmd/main.go --zap-log-level=debug
```

To inspect the state of a specific reconcile loop, add structured log entries using
`log.FromContext(ctx).Info(...)` or `log.FromContext(ctx).V(1).Info(...)`.

## Releasing

Releases are triggered by pushing a semver tag:

```bash
git tag v0.2.0
git push origin v0.2.0
```

The release workflow builds and pushes the container image to
`ghcr.io/kitio-tek/athos:<version>` and packages the Helm chart as a GitHub
release artifact.
