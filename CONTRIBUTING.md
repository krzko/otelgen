# Contributing to trazr-gen

Thank you for your interest in contributing! Please follow these guidelines to help us maintain code quality and streamline the development process.

## Development Workflow

1. **Fork and Clone**  
   Fork the repository and clone it to your local machine.

2. **Branching**  
   Create a feature or fix branch from `main`:
   ```sh
   git checkout -b my-feature
   ```

3. **Pre-commit Checklist**
   - Run all tests:
     ```sh
     go test ./...
     ```
   - Run the linter:
     ```sh
     make lint
     ```
     or
     ```sh
     golangci-lint run
     ```
   - Ensure code is formatted:
     ```sh
     gofmt -s -w .
     goimports -w .
     ```

## Linting

We use [golangci-lint](https://golangci-lint.run/) with a strict configuration (`.golangci.yml`).  
**Key rules:**
- Use `math/rand/v2` (not `math/rand`).
- Avoid denied dependencies (see `.golangci.yml` for details).
- Follow idiomatic Go error handling and naming conventions.
- Security checks are enforced via `gosec`.

Run locally with:
```sh
golangci-lint run
```

## Continuous Integration (GitHub Actions)

All pull requests and pushes trigger GitHub Actions workflows:
- **Lint:** Ensures code style and static analysis.
- **Build:** Verifies the project builds and runs tests.
- **CodeQL:** Runs security/code scanning.
- **Release:** (on tag) Publishes releases using GoReleaser.
- **Docker:** (on release) Builds and pushes Docker images.

You can view workflow files in `.github/workflows/`.

## Releases

We use [GoReleaser](https://goreleaser.com/) for automated releases:
- Releases are triggered by pushing a tag (e.g., `v1.2.3`).
- Artifacts and Docker images are published automatically.

**To create a release:**
1. Bump the version and tag your commit:
   ```sh
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```
2. GoReleaser and GitHub Actions will handle the rest.

## Code Style

- Follow idiomatic Go practices.
- Keep functions short and focused.
- Write table-driven and parallelizable tests.
- Document all exported functions and types.

## Questions?

Open an issue or discussion if you have questions or need help!

---

Thank you for contributing! 