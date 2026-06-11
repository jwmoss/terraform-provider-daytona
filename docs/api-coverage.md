# Daytona API Coverage

This provider is audited against Daytona's generated OpenAPI client in `/Users/jwmoss/github/daytona/libs/api-client-go` at Daytona source commit `7c66e95c8` and client release `v0.187.0`.

The Terraform surface focuses on durable SaaS infrastructure, read-only discovery, and provider-defined actions where Daytona exposes an explicit operational action. Runtime operations, deprecated toolbox proxy operations, admin-only internals, and endpoints that only validate ephemeral tokens are intentionally excluded unless they map cleanly to Terraform state or Terraform's action model.

The provider accepts either `api_key`/`DAYTONA_API_KEY` or `access_token`/`DAYTONA_ACCESS_TOKEN` as its bearer token. Daytona API keys currently work for API-key-enabled routes such as current API-key lookup and volume management; Daytona org/user provisioning and discovery routes are JWT-only and require an OAuth access token plus `organization_id`/`DAYTONA_ORGANIZATION_ID`.

## Covered API Groups

| Daytona API area | Terraform coverage |
| --- | --- |
| API keys | `daytona_api_key` resource, `daytona_api_key`, `daytona_api_keys`, and `daytona_current_api_key` data sources cover create, read, list, and delete for normal API keys. |
| Config and users | `daytona_config`, `daytona_current_user`, `daytona_current_user_organization_invitations`, and `daytona_account_providers` cover managed-service configuration, authenticated-user/account-provider discovery, and pending authenticated-user organization invitations. |
| Docker registries | `daytona_docker_registry` resource plus `daytona_docker_registry`, `daytona_docker_registries`, and `daytona_docker_registry_push_access` data sources cover registry CRUD and temporary push credentials. |
| Health | `daytona_health` covers Daytona liveness and readiness checks, including structured unhealthy readiness responses. |
| Jobs | `daytona_job` and `daytona_jobs` cover job read/list. Job status mutation is runtime-owned and not exposed. |
| Object storage | `daytona_object_storage_push_access` covers temporary object-storage push credentials. |
| Organizations | `daytona_organization` resource and `daytona_organization`, `daytona_organizations`, `daytona_organization_usage`, and `daytona_organization_audit_logs` data sources cover organization CRUD, default region, quotas/rate limits, experimental config, sandbox egress default, usage, and audit logs. |
| Organization invitations | `daytona_organization_invitation` resource plus `daytona_organization_invitation`, `daytona_organization_invitations`, and `daytona_current_user_organization_invitations` data sources cover invitation create/update/cancel, organization-scoped read/list, and authenticated-user list/count. Accept/decline invitations are user actions and not modeled as Terraform-managed state. |
| Organization members | `daytona_organization_member_access`, `daytona_organization_member`, and `daytona_organization_members` cover member access management and read/list. |
| Organization OTEL config | `daytona_organization_otel_config` resource and data source cover get/update/delete of organization OpenTelemetry export settings. |
| Organization roles | `daytona_organization_role` resource plus `daytona_organization_role` and `daytona_organization_roles` data sources cover role create/update/delete and read/list. |
| Regions | `daytona_region`, `daytona_region`, `daytona_regions`, and `daytona_shared_regions` cover customer region CRUD and region discovery. Region credential regeneration endpoints are action-style secret rotation and not exposed yet. |
| Runners | `daytona_runner`, `daytona_runner`, `daytona_runner_full`, `daytona_runner_for_sandbox`, `daytona_authenticated_runner_sandboxes`, `daytona_runners`, and `daytona_runners_by_snapshot_ref` cover runner registration, read/list, deletion, scheduling status, write-only draining status, full runner details, sandbox-to-runner lookup, authenticated-runner sandbox assignment, and snapshot-ref runner mappings. Managed-service `/api/runners` currently returned `404 Cannot GET /api/runners` during live verification and needs Daytona-side route/account confirmation. |
| Sandboxes | `daytona_sandbox`, `daytona_sandbox`, `daytona_sandboxes`, and `daytona_sandbox_query` cover sandbox create/read/list/delete, server-side list filtering/sorting/cursor pagination, desired started/stopped/archived state, public status, labels, CPU/memory/disk resize, auto-stop/archive/delete intervals, and network settings supported by the current resource model. |
| Sandbox access and relationships | `daytona_sandbox_ssh_access`, `daytona_sandbox_build_logs_url`, `daytona_sandbox_port_preview_url`, `daytona_sandbox_signed_port_preview_url`, `daytona_sandbox_toolbox_proxy_url`, `daytona_sandbox_organization`, `daytona_sandbox_region_quota`, `daytona_sandbox_parent`, `daytona_sandbox_ancestors`, and `daytona_sandbox_forks` cover Terraform-readable sandbox access URLs and topology/quota lookups. `daytona_expire_sandbox_signed_port_preview_url` and `daytona_revoke_sandbox_ssh_access` expose access invalidation as Terraform 1.14 provider-defined actions. |
| Sandbox observability | `daytona_sandbox_logs`, `daytona_sandbox_traces`, `daytona_sandbox_trace_spans`, and `daytona_sandbox_metrics` cover bounded OpenTelemetry log, trace, span, and metric reads for a sandbox. |
| Snapshots | `daytona_snapshot`, `daytona_snapshot`, `daytona_snapshots`, and `daytona_snapshot_build_logs_url` cover snapshot create/read/list/delete and build-log URL discovery. `daytona_activate_snapshot` and `daytona_deactivate_snapshot` expose snapshot activation as Terraform 1.14 provider-defined actions. |
| Volumes | `daytona_volume`, `daytona_volume`, `daytona_volume_by_name`, and `daytona_volumes` cover volume create/read-by-ID, read-by-name, list, and delete. |
| Webhooks | `daytona_webhook_initialization_status` and `daytona_webhook_app_portal_access` cover webhook status and Svix app portal access. `daytona_initialize_webhooks` and `daytona_refresh_webhook_endpoints` expose webhook initialization and endpoint refresh as Terraform 1.14 provider-defined actions. |

## Intentionally Excluded

| Endpoint family | Reason |
| --- | --- |
| Admin APIs | Daytona admin-only operations manage global users, runners, region quotas, webhook internals, default registries, snapshot global flags, and global audit logs. These are not normal organization-scoped Terraform provider surfaces. |
| Deprecated toolbox APIs | The generated client marks the toolbox file, git, process, session, PTY, LSP, computer-use, mouse, keyboard, and screenshot endpoints as deprecated. These are runtime control-plane actions, not durable Terraform state. |
| Preview token validation APIs | `isSandboxPublic`, `isValidAuthToken`, `hasSandboxAccess`, and `getSandboxIdFromSignedPreviewUrlToken` validate request-time access/token state and do not provision infrastructure. |
| SSH access validation | Temporary SSH access creation is exposed as a data source, and revocation is exposed as an action. Validation is a request-time token check. |
| Sandbox runtime derivation and maintenance actions | `recoverSandbox`, `createBackup`, `createSandboxSnapshot`, `forkSandbox`, and `updateLastActivity` are imperative runtime actions or derived-object creation flows rather than stable desired-state attributes of the sandbox resource. |
| Runner self-service and job queue APIs | Authenticated-runner info, runner healthcheck, job polling, and job status updates are runner-agent protocol endpoints, not user-managed Terraform infrastructure. Authenticated-runner sandbox assignment is covered by `daytona_authenticated_runner_sandboxes`; user-facing full runner details, scheduling/draining controls, and sandbox/snapshot-ref runner lookups are covered. |
| Region credential regeneration | Proxy, SSH gateway, and snapshot-manager credential regeneration endpoints rotate secrets as one-off actions. Region create stores initial generated credentials as sensitive Terraform state. |
| Organization invitation accept/decline/leave/suspend | These are user/account lifecycle actions or admin/suspension actions, not desired-state infrastructure resources. |
| User account linking and MFA actions | Account link/unlink and SMS MFA enrollment are interactive user-security flows, not organization infrastructure state. |
| Sandbox-auth-token OTEL lookup | `getOrganizationOtelConfigBySandboxAuthToken` is a token-scoped runtime lookup for sandbox agents; the organization-scoped OTEL resource and data source cover Terraform-managed export configuration. |

## Live Verification Gaps

- Live API-key acceptance passed for `daytona_current_api_key` and `daytona_volume` create/read/delete after adding polling for Daytona's asynchronous volume states.
- Org/user discovery and provisioning acceptance tests require `DAYTONA_ACCESS_TOKEN` and `DAYTONA_ORGANIZATION_ID`; the live Daytona API returned `401` for those JWT-only routes when called with a normal Daytona API key.
- Daytona readiness acceptance requires a `DAYTONA_HEALTH_CHECK_API_KEY`; the live Daytona API returned `403` for `/api/health/ready` when called with a normal user API key.
- Runner endpoints remain implemented from OpenAPI but require Daytona-side verification because the managed route returned `404 Cannot GET /api/runners` for the current account/key.
