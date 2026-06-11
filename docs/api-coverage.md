# Daytona API Coverage

This provider is audited against Daytona's generated OpenAPI client in `/Users/jwmoss/github/daytona/libs/api-client-go` at Daytona source commit `3a6dbc150` and client release `v0.187.0`.

The Terraform surface focuses on durable SaaS infrastructure and read-only discovery. Runtime actions, deprecated toolbox proxy operations, admin-only internals, and endpoints that only validate ephemeral tokens are intentionally excluded unless they map cleanly to Terraform state.

## Covered API Groups

| Daytona API area | Terraform coverage |
| --- | --- |
| API keys | `daytona_api_key` resource, `daytona_api_key`, `daytona_api_keys`, and `daytona_current_api_key` data sources cover create, read, list, and delete for normal API keys. |
| Config and users | `daytona_config`, `daytona_current_user`, and `daytona_account_providers` cover managed-service configuration and authenticated-user/account-provider discovery. |
| Docker registries | `daytona_docker_registry` resource plus `daytona_docker_registry`, `daytona_docker_registries`, and `daytona_docker_registry_push_access` data sources cover registry CRUD and temporary push credentials. |
| Health | `daytona_health` covers Daytona liveness and readiness checks, including structured unhealthy readiness responses. |
| Jobs | `daytona_job` and `daytona_jobs` cover job read/list. Job status mutation is runtime-owned and not exposed. |
| Object storage | `daytona_object_storage_push_access` covers temporary object-storage push credentials. |
| Organizations | `daytona_organization` resource and `daytona_organization`, `daytona_organizations`, `daytona_organization_usage`, and `daytona_organization_audit_logs` data sources cover organization CRUD, default region, quotas/rate limits, experimental config, sandbox egress default, usage, and audit logs. |
| Organization invitations | `daytona_organization_invitation` resource plus `daytona_organization_invitation` and `daytona_organization_invitations` data sources cover invitation create/update/cancel and read/list. Accept/decline invitations are user actions and not modeled as Terraform-managed state. |
| Organization members | `daytona_organization_member_access`, `daytona_organization_member`, and `daytona_organization_members` cover member access management and read/list. |
| Organization OTEL config | `daytona_organization_otel_config` resource and data source cover get/update/delete of organization OpenTelemetry export settings. |
| Organization roles | `daytona_organization_role` resource plus `daytona_organization_role` and `daytona_organization_roles` data sources cover role create/update/delete and read/list. |
| Regions | `daytona_region`, `daytona_region`, `daytona_regions`, and `daytona_shared_regions` cover customer region CRUD and region discovery. Region credential regeneration endpoints are action-style secret rotation and not exposed yet. |
| Runners | `daytona_runner`, `daytona_runner`, and `daytona_runners` cover runner registration, read/list, and deletion. Managed-service `/api/runners` currently returned `404 Cannot GET /api/runners` during live verification and needs Daytona-side route/account confirmation. |
| Sandboxes | `daytona_sandbox`, `daytona_sandbox`, and `daytona_sandboxes` cover sandbox create/read/list/delete, desired started/stopped/archived state, public status, labels, CPU/memory/disk resize, auto-stop/archive/delete intervals, and network settings supported by the current resource model. |
| Sandbox access and relationships | `daytona_sandbox_ssh_access`, `daytona_sandbox_build_logs_url`, `daytona_sandbox_port_preview_url`, `daytona_sandbox_toolbox_proxy_url`, `daytona_sandbox_organization`, `daytona_sandbox_region_quota`, `daytona_sandbox_parent`, `daytona_sandbox_ancestors`, and `daytona_sandbox_forks` cover Terraform-readable sandbox access URLs and topology/quota lookups. |
| Sandbox observability | `daytona_sandbox_logs`, `daytona_sandbox_traces`, `daytona_sandbox_trace_spans`, and `daytona_sandbox_metrics` cover bounded OpenTelemetry log, trace, span, and metric reads for a sandbox. |
| Snapshots | `daytona_snapshot`, `daytona_snapshot`, `daytona_snapshots`, and `daytona_snapshot_build_logs_url` cover snapshot create/read/list/delete and build-log URL discovery. Snapshot activate/deactivate endpoints are runtime actions and not modeled as resources. |
| Volumes | `daytona_volume`, `daytona_volume`, and `daytona_volumes` cover volume create/read/list/delete. |
| Webhooks | `daytona_webhook_initialization_status` and `daytona_webhook_app_portal_access` cover webhook status and Svix app portal access. Initialization and endpoint refresh are operational actions and not exposed as Terraform state. |

## Intentionally Excluded

| Endpoint family | Reason |
| --- | --- |
| Admin APIs | Daytona admin-only operations manage global users, runners, region quotas, webhook internals, default registries, snapshot global flags, and global audit logs. These are not normal organization-scoped Terraform provider surfaces. |
| Deprecated toolbox APIs | The generated client marks the toolbox file, git, process, session, PTY, LSP, computer-use, mouse, keyboard, and screenshot endpoints as deprecated. These are runtime control-plane actions, not durable Terraform state. |
| Preview token validation APIs | `isSandboxPublic`, `isValidAuthToken`, `hasSandboxAccess`, and `getSandboxIdFromSignedPreviewUrlToken` validate request-time access/token state and do not provision infrastructure. |
| Signed preview URL expiration | `expireSignedPortPreviewUrl` invalidates an ephemeral token and has no stable desired state. |
| SSH access validation and revocation | Temporary SSH access creation is exposed as a data source; validation/revocation are request-time operational actions. |
| Sandbox runtime derivation and maintenance actions | `recoverSandbox`, `createBackup`, `createSandboxSnapshot`, `forkSandbox`, and `updateLastActivity` are imperative runtime actions or derived-object creation flows rather than stable desired-state attributes of the sandbox resource. |
| Runner self-service and job queue APIs | Authenticated-runner info, runner sandbox assignment, runner healthcheck, job polling, and job status updates are runner-agent protocol endpoints, not user-managed Terraform infrastructure. |
| Runner scheduling/draining updates | Daytona's OpenAPI describes body fields for scheduling/draining, but the generated Go client request types do not currently expose body setters for those endpoints. |
| Region credential regeneration | Proxy, SSH gateway, and snapshot-manager credential regeneration endpoints rotate secrets as one-off actions. Region create stores initial generated credentials as sensitive Terraform state. |
| Organization invitation accept/decline/leave/suspend | These are user/account lifecycle actions or admin/suspension actions, not desired-state infrastructure resources. |
| User account linking and MFA actions | Account link/unlink and SMS MFA enrollment are interactive user-security flows, not organization infrastructure state. |
| Sandbox-auth-token OTEL lookup | `getOrganizationOtelConfigBySandboxAuthToken` is a token-scoped runtime lookup for sandbox agents; the organization-scoped OTEL resource and data source cover Terraform-managed export configuration. |

## Live Verification Gaps

- `DAYTONA_API_KEY` is not set in the current shell, so read-only live acceptance tests could not be rerun for the latest data sources.
- Full create/delete acceptance remains blocked by the live organization state: Daytona returns `Organization is suspended: Payment method required` for resource creation.
- Runner endpoints remain implemented from OpenAPI but require Daytona-side verification because the managed route returned `404 Cannot GET /api/runners` for the current account/key.
