# Contributing to Athos

Thank you for your interest in contributing to Athos. The following guidelines
explain how to submit issues and pull requests effectively.

## Code of Conduct

This project follows the CNCF Code of Conduct. All contributors are expected to
treat each other with respect.

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

## Documentation

Update the relevant documentation when changing public APIs or operator behaviour:

- API field changes belong in `docs/api-reference.md`
- Architecture and workflow changes belong in `DEVELOPER.md`
- User-facing feature additions belong in `README.md`

## License

By contributing to Athos, you agree that your contributions will be licensed
under the Apache License, Version 2.0.
