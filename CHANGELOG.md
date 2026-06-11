## 0.2.0 (2026-06-11)

FEATURES:

- Added user API-key revocation, sandbox runtime state actions, and organization region quota management.
- Added admin APIs for organization region quotas, default Docker registries, snapshot general status, snapshot image cleanup checks, global audit logs, webhook status/message attempts/send/initialize, user read/list/create/key-pair regeneration, sandbox recovery, and runner management.
- Added the `daytona_admin_runner` resource plus `daytona_admin_runner` and `daytona_admin_runners` data sources for admin runner create/read/list/delete/scheduling coverage.
- Updated runner and auth documentation for Daytona API-key, OAuth/JWT, health-check, organization-infrastructure, and admin-fixture requirements.

VALIDATION:

- Re-ran local generation, formatting, tests, linting, release checks, release snapshot packaging, hooks, and secret-pattern scans across the post-`v0.1.0` updates.
- GitHub Actions `Tests` passed for every pushed post-`v0.1.0` provider commit.

## 0.1.0 (2026-06-11)

FEATURES:

- Initial Terraform Plugin Framework provider for Daytona, configured with `api_key` or `access_token`, `api_url`, and optional `organization_id`.
- Resources for Daytona API keys, Docker registries, organizations, organization invitations, member access, organization roles, organization OTEL config, regions, region credential rotation, runners, runner scheduling/draining controls, sandboxes, snapshots, and volumes.
- Terraform 1.14 provider-defined actions for snapshots, organization invitations/lifecycle, user account linking/SMS MFA, sandbox lifecycle/access operations, and webhook initialization/refresh.
- Data sources for Daytona configuration, current user/API key, authenticated-user organization invitations, organizations, organization governance, OpenTelemetry configuration, regions, shared regions, runners and runner relationships, authenticated runner sandbox assignment, sandboxes, sandbox query/filtering, sandbox access/relationships/observability/validation, snapshots, jobs, registries, volumes by ID/name, object-storage push access, webhooks, and health checks.
- Generated provider documentation, GitHub-source module examples, repository security guidance, and GoReleaser-based GitHub release packaging.
- Live verification for API-key-backed current API-key lookup, volume create/read/delete, and the public GitHub module example provisioning a sandbox plus companion volume.

BUG FIXES:

- Handles Daytona sandbox server defaults for user, target, CPU, GPU, memory, disk, and auto-archive/delete intervals without Terraform provider inconsistent-result errors.
