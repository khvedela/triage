# Security Policy

## Supported versions

Security fixes are applied to the latest minor release on `main`. Pre-1.0, we do not backport to older minor lines.

## Reporting a vulnerability

**Do not** open a public issue for security problems. Instead:

- Use GitHub's [private security advisories](https://github.com/OWNER/triage/security/advisories/new) (preferred), or
- Email the maintainers at `security@triage.example` (replace with actual contact before first release).

Please include:

- A clear description of the vulnerability
- Steps to reproduce
- Affected versions
- Any proof-of-concept code (as attachment, not inline)

We will acknowledge receipt within 3 business days and aim to publish a fix within 30 days for critical issues.

## Scope

`triage` is a read-only CLI — it does not create, mutate, or delete Kubernetes resources. The primary attack surface is:

1. **Malicious cluster responses.** A compromised API server could attempt to exploit parsing bugs. We rely on upstream `client-go` hardening.
2. **Local file handling.** `triage` reads kubeconfig and its own config file from local paths. It does not exec, fetch remote URLs, or phone home.
3. **Output rendering.** ANSI escape sequences in cluster-provided strings (container names, event messages) are sanitized before rendering.

Out of scope: attacks that require an attacker who already has cluster-admin or local shell access.
