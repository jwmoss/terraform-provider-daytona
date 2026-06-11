## 0.1.0 (2026-06-11)

FEATURES:

- Initial Terraform Plugin Framework provider for Daytona, configured with `api_key` or `access_token`, `api_url`, and optional `organization_id`.
- Resources for Daytona API keys, Docker registries, organizations, organization invitations, member access, organization roles, organization OTEL config, regions, region credential rotation, runners, runner scheduling/draining controls, sandboxes, snapshots, and volumes.
- Actions for activating/deactivating Daytona snapshots, accepting/declining organization invitations, revoking user API keys, linking/unlinking secondary user accounts, starting SMS MFA enrollment, leaving/suspending/unsuspending organizations, starting sandbox backups/snapshots/forks/recovery, expiring/revoking sandbox access tokens, updating sandbox last activity, and initializing/refreshing Daytona webhooks with Terraform 1.14 provider-defined actions.
- Data sources for Daytona configuration, current user/API key, authenticated-user organization invitations, organizations, organization governance, organization OpenTelemetry configuration, regions, shared regions, runners and runner relationships, authenticated runner sandbox assignment, sandboxes, sandbox query/filtering, sandbox access/relationships/observability/validation, snapshots, jobs, registries, volumes by ID/name, object-storage push access, webhooks, and health checks.
- Generated provider documentation, GitHub-source module examples, repository security guidance, and GoReleaser-based GitHub release packaging.
- Live verification for API-key-backed current API-key lookup, volume create/read/delete, and the public GitHub module example provisioning a sandbox plus companion volume.

BUG FIXES:

- Handles Daytona sandbox server defaults for user, target, CPU, GPU, memory, disk, and auto-archive/delete intervals without Terraform provider inconsistent-result errors.
