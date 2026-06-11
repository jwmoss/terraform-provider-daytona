# Security Policy

## Reporting a Vulnerability

Please report suspected security issues privately through GitHub's security advisory flow for this repository when available. If that is unavailable, open a minimal public issue that does not include exploit details or secrets, and request a private contact path.

Do not include Daytona API keys, Terraform state, generated preview URLs, SSH access tokens, or other credentials in public issues, pull requests, logs, or examples.

## Sensitive Terraform State

This provider marks Daytona API keys, temporary access tokens, generated push credentials, signed preview URLs, SSH commands, and telemetry fields that may contain workload data as sensitive. Terraform state can still contain sensitive values, so store state in an encrypted backend with restricted access.
