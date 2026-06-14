## 0.3.0 (2026-06-11)

ENHANCEMENTS:

- Transient Daytona API failures (HTTP 429, 5xx, and connection errors) are now retried with capped exponential backoff that honors any `Retry-After` header, so a single blip no longer fails an entire `terraform apply`.

BUG FIXES:

- Reads of `daytona_organization_member_access`, `daytona_organization_role`, and `daytona_organization_invitation` no longer remove the resource from state when the API returns a transient error; only a genuine not-found does.
- `daytona_runner`, `daytona_admin_runner`, `daytona_organization`, `daytona_sandbox`, and `daytona_volume` now persist state as soon as the remote object exists, so a failed follow-up call during create can no longer orphan the object or lose the one-time runner API key.
- `daytona_region` persists state after each credential rotation, so a partial rotation failure can no longer lose a freshly regenerated credential or silently skip a pending rotation.
- `daytona_sandbox` no longer refreshes `env` from the API (preventing inconsistent-result errors and unwanted replacement), keeps unconfigured `labels` and empty `network_allow_list` null, clears `network_allow_list` server-side when removed from configuration, and reconciles `desired_state` with the actual sandbox state on refresh so out-of-band stops surface as a plan diff.
- `daytona_runner.draining` is now tracked in state instead of write-only; previously, changing only `draining` produced an empty plan and the drain request was never sent.
- `daytona_api_key` no longer stores the masked key value from the list endpoint after import; `value` stays null when the real key is unavailable.
- All API calls now have a 5-minute per-attempt timeout instead of hanging indefinitely on a stalled connection.
- The `daytona_sandboxes` and `daytona_snapshots` data sources paginate through all results instead of silently truncating at 100 items.
- `daytona_organization_region_quota` destroy now warns that the quota remains active, since Daytona's organization API has no quota delete endpoint.
- The provider warns when `DAYTONA_ACCESS_TOKEN` overrides an explicitly configured `api_key`.

NOTES:

- Release checksums are now GPG-signed, as required for Terraform Registry publication.
- Added unit test coverage for the previously untested resources.
- Added acceptance-test sweepers (`go test ./internal/provider/ -sweep=all`) for volumes, sandboxes, snapshots, and Docker registries that delete leaked `tf-acc-`-prefixed resources after a failed run.
- Added live acceptance coverage for `daytona_sandbox` and `daytona_docker_registry`; `daytona_snapshot` acceptance is opt-in via `DAYTONA_ACC_SNAPSHOT_BUILD=1` because it builds a real image.

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
