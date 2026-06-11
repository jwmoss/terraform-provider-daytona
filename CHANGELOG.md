## 0.1.0 (Unreleased)

FEATURES:

- Initial Terraform Plugin Framework provider for Daytona, configured with `api_key` or `access_token`, `api_url`, and optional `organization_id`.
- Resources for Daytona API keys, Docker registries, organizations, organization invitations, member access, organization roles, organization OTEL config, regions, region credential rotation, runners, runner scheduling/draining controls, sandboxes, snapshots, and volumes.
- Actions for activating/deactivating Daytona snapshots, starting sandbox backups/snapshots/forks/recovery, expiring/revoking sandbox access tokens, updating sandbox last activity, and initializing/refreshing Daytona webhooks with Terraform 1.14 provider-defined actions.
- Data sources for Daytona configuration, current user/API key, authenticated-user organization invitations, organizations, organization governance, regions, shared regions, runners and runner relationships, authenticated runner sandbox assignment, sandboxes, sandbox query/filtering, sandbox access/relationships/observability, snapshots, jobs, registries, volumes by ID/name, object-storage push access, webhooks, and health checks.
- Generated provider documentation, examples, repository security guidance, and GoReleaser-based release packaging.
