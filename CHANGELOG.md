## 0.1.0 (Unreleased)

FEATURES:

- Initial Terraform Plugin Framework provider for Daytona, configured with `api_key`, `api_url`, and optional `organization_id`.
- Resources for Daytona API keys, Docker registries, organizations, organization invitations, member access, organization roles, organization OTEL config, regions, runners, sandboxes, snapshots, and volumes.
- Actions for activating and deactivating Daytona snapshots with Terraform 1.14 provider-defined actions.
- Data sources for Daytona configuration, current user/API key, organizations, organization governance, regions, shared regions, runners, sandboxes, sandbox access/relationships/observability, snapshots, jobs, registries, object-storage push access, webhooks, and health checks.
- Generated provider documentation, examples, repository security guidance, and GoReleaser-based release packaging.
