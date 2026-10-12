# Contributing to SentinelFlow

**Contributions are welcome!** You do not need to be a maintainer to help. Bug fixes, tests, documentation, reproducible bug reports, issue triage, and focused feature work are all valuable.

[![Contribute](https://img.shields.io/badge/Contributions-welcome-2ea44f?style=for-the-badge&logo=github)](https://github.com/lecodev-26/sentinelflow/issues)

[Browse open issues](https://github.com/lecodev-26/sentinelflow/issues) · [Good first issues](https://github.com/lecodev-26/sentinelflow/labels/good%20first%20issue) · [Report a bug](https://github.com/lecodev-26/sentinelflow/issues/new?template=bug_report.md)

## Contribution workflow

1. **Find or propose a task.** Search [open issues](https://github.com/lecodev-26/sentinelflow/issues) first. Comment that you would like to work on an issue so effort is not duplicated. For large or architectural changes, open an issue and agree on the direction before coding.
2. **Choose the right target.** Stable fixes intended for the current V5 release should target main only when the maintainers confirm that target. V6 work belongs on v6. If contributing to an existing pull request or a task already in progress, coordinate with its author before creating overlapping changes. Do not assume main is the V6 development branch.
3. **Fork and branch.** Fork the repository, clone your fork, add the upstream remote, and create a focused branch from the correct target branch:
   ~~~bash
   git clone https://github.com/<your-user>/sentinelflow.git
   cd sentinelflow
   git remote add upstream https://github.com/lecodev-26/sentinelflow.git
   git fetch upstream
   git switch -c feat/short-description upstream/v6
   ~~~
   Use fix/short-description for bug fixes and docs/short-description for documentation-only changes. Replace upstream/v6 with the explicitly agreed target when working on another release line.
4. **Make a focused change.** Follow existing patterns, keep compatibility in mind, add/update tests, and document public API/configuration behavior. Never commit credentials, .env files, databases, backups, generated binaries, or real user data.
5. **Run relevant checks.** From the repository root:
   ~~~bash
   go test ./...
   go vet ./...
   go build ./cmd/gateway ./cmd/controlplane ./cmd/worker ./cmd/cli ./cmd/migrator
   git diff --check
   ~~~
   Run gofmt on Go files you changed. Integration tests may require PostgreSQL/Redis; clearly report unavailable infrastructure instead of describing those tests as passed.
6. **Push your branch and open a pull request.** Push to your fork, then open a PR against the agreed upstream base branch. Fill in the PR template: describe the problem, the change, validation, security/tenant impact, migration/rollback impact, and docs. Link the relevant issue (for example, Closes #123 only if the PR truly completes that issue).
7. **Respond to review.** Keep the discussion constructive, make requested changes in follow-up commits, and do not merge your own PR unless repository policy explicitly grants that responsibility.

## What can I contribute?

- Reproducible bug reports and minimal test cases.
- Tests, documentation improvements, examples, and SDK fixes.
- Focused fixes for issues labelled good first issue.
- V6 runtime, security, governance, reliability, and observability work that matches the [V6 architecture](docs/v6/ARCHITECTURE.md) and [architecture principles](docs/v6/PRINCIPLES.md).
- Performance measurements and clear reports of deployment or compatibility problems.

If an issue is already assigned or someone has announced they are working on it, ask before starting overlapping work. If you are unsure, ask a question in the issue first.

## Development requirements and checks

Go 1.27+ is required. PostgreSQL and Redis are needed for some integration paths.

~~~bash
go test ./...
go vet ./...
go build ./cmd/gateway ./cmd/controlplane ./cmd/worker ./cmd/cli ./cmd/migrator
git diff --check
~~~

Format changed Go files with gofmt. Add tests for behavior changes, especially failure, retry, authorization, and tenant-isolation cases. If a check cannot run in your environment, state which one and why in the PR.

## Pull-request expectations

- Keep the PR focused and reasonably small.
- Link the issue and explain the user-visible or operational impact.
- Include tests for behavioral changes and update documentation for public API/configuration changes.
- Preserve tenant boundaries, authorization and secret-handling guarantees.
- Describe migration, compatibility, rollback, and retry implications.
- Do not claim checks passed unless they actually ran successfully.
- Mark incomplete work as draft and clearly describe what remains.

## Security vulnerabilities

**Do not report security vulnerabilities in public issues or pull requests.** Follow [SECURITY.md](SECURITY.md) for responsible, private disclosure.

## Code of conduct and license

Be respectful and constructive; see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). By contributing, you agree that your contributions are provided under the repository's MIT License.
