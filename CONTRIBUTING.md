# Contributing to Athos

Thank you for your interest in contributing to Athos. The following guidelines
explain how to submit issues and pull requests effectively.

## Code of Conduct

All contributors are expected to treat each other with respect.

## Reporting Issues

Before opening an issue, search the existing issues to see if the problem has
already been reported. When filing a new issue, include:

- The Athos version (operator image tag or git SHA)
- The Kubernetes version and platform
- A minimal reproduction case (ideally a PostgresCluster YAML that demonstrates
  the problem)
- The full output of `kubectl describe` and relevant operator logs

## Development Workflow

1. Fork the repository on GitHub.
2. Clone your fork and create a feature branch from `main`:

   ```bash
   git checkout -b feat/my-feature
   ```

3. Make your changes. Ensure:
   - All new public functions have doc comments
   - No emoji anywhere in code, comments, or commit messages
   - Code passes `go vet ./...` and `golangci-lint run`

4. Add or update tests. Pull requests that reduce test coverage will not be merged.

5. Run the full test suite before pushing:

   ```bash
   make test
   ```

6. Commit your changes using descriptive messages:

   ```
   controller: handle StatefulSet update conflicts with retry
   ```

   Commit messages must:
   - Use the imperative mood ("add feature", not "added feature")
   - Be 72 characters or fewer on the first line
   - Reference a GitHub issue number where applicable (e.g. "Fixes #42")

7. Push your branch and open a pull request against `main`. Fill in the pull
   request template.

## Pull Request Guidelines

- Keep pull requests focused. One logical change per PR.
- Rebase on `main` before requesting review.
- All CI checks must pass.
- At least one maintainer approval is required before merging.
- Squash-merge is preferred for small changes. For larger changes, a merge commit
  preserving the individual commits may be preferred at maintainer discretion.

## Commit Style

Follow the conventional commit style used throughout the repository:

```
<scope>: <short description>

<optional longer description>

Fixes #<issue-number>
```

Scopes include: `api`, `controller`, `internal`, `webhook`, `test`, `helm`,
`kuttl`, `ci`, `docs`, `make`.

## End-to-End Tests

Athos ships two parallel e2e suites. Both run on every CI build; new
scenarios should target Chainsaw first.

Test harness upstream references:

- [kyverno/chainsaw](https://github.com/kyverno/chainsaw) - declarative
  Kubernetes-native test framework (replaces KUTTL).
- [kudobuilder/kuttl](https://github.com/kudobuilder/kuttl) - KUbernetes
  Test TooL, the original framework Athos was scaffolded on.

- **Chainsaw (default).** `tests/e2e/chainsaw/` - each test is a
  `apiVersion: chainsaw.kyverno.io/v1alpha1` `Test` resource with declarative
  `apply` / `assert` / `error` / `cleanup` steps and per-step timeouts.
  Run locally with:

  ```bash
  make e2e-test
  # or
  chainsaw test --config tests/e2e/chainsaw/.chainsaw.yaml tests/e2e/chainsaw/tests/
  ```

- **KUTTL (legacy parity).** `tests/e2e/kuttl/` - each test is a numbered
  directory with `NN-<name>.yaml` apply files paired with `NN-assert.yaml`
  expected-state files. Kept for parity with the historical Athos test
  harness while we migrate every scenario into Chainsaw. Run locally with:

  ```bash
  make e2e-test-kuttl
  ```

<details>
<summary><b>Why Chainsaw over KUTTL</b></summary>

Athos was originally scaffolded on KUTTL. KUTTL is enough for the basic
"apply this CR, expect a StatefulSet with this name" loop, but every
slightly richer assertion drifts into bash inside a `commands:` block.
Chainsaw, maintained by the Kyverno team, fixes the specific gaps Athos
kept hitting (see the upstream rationale in
[kyverno/chainsaw#254](https://github.com/kyverno/chainsaw/discussions/254)):

| Concern | KUTTL | Chainsaw |
|---|---|---|
| Test resource model | Numbered `NN-step.yaml` / `NN-assert.yaml` files paired by leading digit. | A single `chainsaw-test.yaml` `Test` resource with named `steps[]`, each with `try` / `catch` / `cleanup`. |
| Array assertions | Positional only - no way to say "this list is unordered". | Per-field directives, so `env:` can be unordered while `initContainers:` stays ordered. |
| Conditional / comparative assertions | No `>`, `<`, `contains`, partial-object matching. | First-class JMESPath assertions (e.g. `status.(readyReplicas > '0'): true`). |
| Command / CLI output | Plain `commands:` block; you write bash to check output. | `script:` and `exec:` actions with `check:` over stdout/stderr/exit code. |
| Timeouts | One global `timeout`. | Per-stage budgets (`apply` / `assert` / `error` / `delete` / `cleanup` / `exec`) at suite, test, and step level. |
| Negative tests | Workarounds via `errors.yaml`. | First-class `error:` step asserting a resource is absent or rejected. |
| Debugging | Sparse logs; you re-run with `--verbose` and read raw events. | Structured per-step logs with `BEGIN` / `END` markers and a richer failure dump. |


When adding a new e2e scenario, port it to Chainsaw and drop the equivalent
KUTTL case in the same PR if the coverage now lives only in Chainsaw.

</details>

## Documentation

Update the relevant documentation when changing public APIs or operator behaviour:

- API field changes belong in `docs/api-reference.md`
- Architecture and workflow changes belong in `DEVELOPER.md`
- User-facing feature additions belong in `README.md`



## License

By contributing to Athos, you agree that your contributions will be licensed
under the Apache License, Version 2.0.
