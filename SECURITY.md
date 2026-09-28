# Security Policy

## Supported versions

Security fixes are applied to the current main development line and the latest published release when practical.

## Reporting a vulnerability

Please do not disclose security vulnerabilities in public GitHub issues.

Use GitHub's private security reporting/advisory mechanism for this repository when available. If private reporting is unavailable, contact the maintainer through a private repository channel before public disclosure.

Include:

- affected version/commit
- impact
- reproduction steps or proof of concept
- relevant logs without secrets
- suggested mitigation, if known

Please allow reasonable time for investigation and coordinated disclosure.

## Secrets

Never include API keys, passwords, JWTs, private certificates or production database URLs in issues, pull requests or commits.

SentinelFlow is designed to receive production secrets from external secret management rather than generating production credentials implicitly.
