# Developer Guide

This guide covers setting up a development environment, building the operator,
running tests, and debugging locally.

## Prerequisites

- Go 1.25 or later
- Docker or Podman
- kubectl
- Helm 3.x
- kind (for local cluster testing)
- operator-sdk v1.38.0
- controller-gen v0.18.0 (installed via `make controller-gen`)
- golangci-lint v2.x (installed automatically by the CI action; locally
  install with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`)

## Repository Structure

```
athos-kubernetes/
  api/v1alpha1/                Custom resource type definitions
  cmd/main.go                  Operator entry point
  cmd/plugin/                  kubectl-athos plugin
  config/                      Kustomize manifests
  charts/athos-kubernetes/     Helm chart shipped with releases
  internal/
    controller/                Reconciler implementations
    postgres/                  PostgreSQL-specific helpers (naming, config, resources)
    ...                        Other reusable libraries (cronexpr, sqlescape, etc.)
  tests/e2e/chainsaw/          Chainsaw (Kyverno) e2e tests — default
  tests/e2e/kuttl/             KUTTL e2e tests — legacy parity
  test/e2e/                    Go-based end-to-end tests
  .github/workflows/           CI/CD pipelines
```

## Setting Up the Development Environment

Clone the repository and download dependencies:

```bash
git clone git@github.com:Kitio-Tek/athos-kubernetes.git
cd athos-kubernetes
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
make docker-build IMG=ghcr.io/kitio-tek/athos-kubernetes:dev
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

### End-to-End Tests (Chainsaw + KUTTL)

E2E tests require a running Kubernetes cluster and the operator deployed inside it.
Using kind:

```bash
make kind-create
make docker-build IMG=ghcr.io/kitio-tek/athos-kubernetes:dev
make kind-load    IMG=ghcr.io/kitio-tek/athos-kubernetes:dev
make helm-install IMG=ghcr.io/kitio-tek/athos-kubernetes:dev

# Default Chainsaw suite (Kyverno chainsaw.kyverno.io/v1alpha1 Tests):
make e2e-test

# Legacy KUTTL parity suite (kudobuilder/kuttl):
make e2e-test-kuttl
```

Install Chainsaw from <https://github.com/kyverno/chainsaw/releases> and
`kubectl-kuttl` from <https://github.com/kudobuilder/kuttl/releases> before
running these targets locally.

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

Releases are produced manually for each semver tag. The flow is:

1. Update `charts/athos-kubernetes/Chart.yaml` to the new version.
2. Commit, push, and tag:

   ```bash
   git tag -a v0.7.0 -m "v0.7.0 - <one-line summary>"
   git push origin main
   git push origin v0.7.0
   ```

3. Package the helm chart and create the GitHub release:

   ```bash
   helm package charts/athos-kubernetes/ -d /tmp
   gh release create v0.7.0 /tmp/athos-kubernetes-0.7.0.tgz \
     --title "v0.7.0" --notes-file release-notes.md
   ```

4. Optionally trigger the `Release` workflow from the Actions tab to also
   publish the container image to `ghcr.io/kitio-tek/athos-kubernetes:<version>`.
   The workflow accepts the tag as a `workflow_dispatch` input, so it will
   never fire automatically and never collide with the manual `gh release
   create` step above.
