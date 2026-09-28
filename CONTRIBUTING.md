# Contributing to SentinelFlow

Thanks for contributing.

## Before you start

- Read the root README and relevant architecture documentation.
- For security vulnerabilities, follow SECURITY.md instead of opening a public issue.
- For large architectural changes, open an issue first so the design can be discussed.

## Development

Requirements: Go 1.27+, PostgreSQL and Redis for integration paths.

    go test ./...
    go vet ./...
    gofmt -w .
    go build ./cmd/gateway ./cmd/controlplane ./cmd/worker ./cmd/cli ./cmd/migrator
    git diff --check

Some integration tests require PostgreSQL/Redis. If an infrastructure dependency is unavailable, report that separately from code failures.

## Pull requests

- Keep changes focused.
- Add or update tests.
- Update API/docs when public behavior changes.
- Do not commit credentials, .env files, databases, backups or generated binaries.
- Explain migrations and rollback considerations.
- Prefer small, reviewable commits.
- Do not silently change public API compatibility.

## Review checklist

- What behavior changed?
- Is the tenant/security boundary preserved?
- Are failures and retries defined?
- Is the change observable?
- Are migrations explicit and reversible?
- Are tests included?
- Does documentation match the implementation?

## License

By contributing, you agree that your contributions are provided under the repository's MIT License.
