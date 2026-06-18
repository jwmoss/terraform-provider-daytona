## 0.7.0 (2026-06-17)

NOTES:

- The provider has moved to the **536Tech** organization and now publishes as `536tech/daytona` on the Terraform Registry (source: `github.com/536tech/terraform-provider-daytona`). Update your `source` from `jwmoss/daytona` to `536tech/daytona`; the `jwmoss/daytona` namespace is deprecated.
- Release artifacts are now signed with the 536Tech provider signing key (fingerprint `82517EBC5706F5211B8F65F9F0FDD0EDF505625D`). No provider functionality changed in this release.
- Added the `azure-runner-fleet` use-case example: launch Azure VM hosts and register each as a Daytona runner in one apply, feeding the runner key into cloud-init.
- Added `docs/cli-vs-terraform.md` describing where this provider adds value versus the Daytona CLI/SDK.

## 0.6.0 (2026-06-16)

FEATURES:

- `daytona_sandbox`: added persistent volume mounts with optional subpaths.
- `daytona_sandbox`: added ordered GPU type preferences, assigned `gpu_type`, Dockerfile `build_info`, and `last_activity_at` state.
- `daytona_snapshot`: added ordered GPU type preferences, assigned `gpu_type`, and Dockerfile `build_info` state.

NOTES:

- Added unit and acceptance coverage for the new sandbox and snapshot fields. The sandbox GPU acceptance test is gated by `DAYTONA_ACC_GPU` because it requires live GPU quota.

## 0.5.0 (2026-06-15)

BUG FIXES:

- `daytona_docker_registry`: an unset `project` no longer fails with "inconsistent result after apply"; an empty value from the API is now mapped to null.
- `daytona_api_key`: `user_id` and `last_used_at` are no longer left unknown after create (the create response omits them; they are set to null and populated on read).
- `daytona_organization`: creating an organization no longer calls the admin-only quota endpoint when no quota is configured, so a non-admin organization create succeeds.
- `daytona_organization_invitation`: `expires_at` is now Optional+Computed so a Daytona-assigned default expiry no longer breaks apply.
- `daytona_snapshot`: `sandbox_class`, `cpu`, `gpu`, `memory`, and `disk` are now Optional+Computed so Daytona-assigned defaults no longer break apply.
- `daytona_organization_otel_config`: the configuration is now read from the organization object instead of the dedicated read endpoint, which returns HTTP 401 for an organization owner; configured header values are preserved (the organization object redacts them).

NOTES:

- Added functional acceptance tests for all resources. Documented the API-key vs access-token (JWT) auth split; `daytona_region`, `daytona_runner`, and `daytona_admin_runner` are marked experimental (their endpoints are served only by self-hosted Daytona) and their acceptance tests are gated behind `DAYTONA_ACC_*` flags.
- Added `ci/daytona-access-token.sh`, a local helper that mints an access token from a `daytona login` refresh token for running the organization-API acceptance tests.
- The acceptance workflow now sets `DAYTONA_ACC_REGISTRY=1` so the Docker registry resource is exercised in CI.

## 0.4.3 (2026-06-14)

NOTES:

- Removed the per-file MPL-2.0 license headers from the Go sources; license intent is carried by the `LICENSE` file, whose copyright now attributes the work to the author instead of the scaffolding template default. No functional provider changes.
- Added a `CONTRIBUTING.md`, GitHub issue forms (bug report, feature request, and a config routing security reports to `SECURITY.md`), and README status/license badges with `Contributing` and `License` sections.

## 0.4.2 (2026-06-14)

NOTES:

- Re-tagged release of the 0.4.1 commit with no functional changes.

## 0.4.1 (2026-06-14)

BUG FIXES:

- Automatic API retries no longer replay non-idempotent mutations. The v0.4.0 retry wrapper retried 5xx and connection errors for every method, so a create whose response was lost could be replayed and duplicate a billable resource. Retries are now gated: HTTP 429 is retried for any request (the server rejects it before processing), 5xx is retried only for idempotent methods (`GET`/`HEAD`/`PUT`/`DELETE`), and transport errors are not retried.

## 0.4.0 (2026-06-14)

ENHANCEMENTS:

- Transient Daytona API failures (HTTP 429, 5xx, and connection errors) are now retried with capped exponential backoff that honors any `Retry-After` header, so a single blip no longer fails an entire `terraform apply`. The 5-minute timeout now bounds each attempt.

NOTES:

- Added acceptance-test sweepers (`go test ./internal/provider/ -sweep=all`) for volumes, sandboxes, snapshots, and Docker registries that delete leaked `tf-acc-`-prefixed resources after a failed run.
- Added live acceptance coverage for `daytona_sandbox` under the API-key precheck. `daytona_docker_registry` acceptance is opt-in via `DAYTONA_ACC_REGISTRY=1` (needs the `WRITE_REGISTRIES` permission) and `daytona_snapshot` acceptance is opt-in via `DAYTONA_ACC_SNAPSHOT_BUILD=1` (builds a real image).
- Added platform use-case examples.

## 0.3.0 (2026-06-11)

BUG FIXES:

- Reads of `daytona_organization_member_access`, `daytona_organization_role`, and `daytona_organization_invitation` no longer remove the resource from state when the API returns a transient error; only a genuine not-found does.
- `daytona_runner`, `daytona_admin_runner`, `daytona_organization`, `daytona_sandbox`, and `daytona_volume` now persist state as soon as the remote object exists, so a failed follow-up call during create can no longer orphan the object or lose the one-time runner API key.
- `daytona_region` persists state after each credential rotation, so a partial rotation failure can no longer lose a freshly regenerated credential or silently skip a pending rotation.
- `daytona_sandbox` no longer refreshes `env` from the API (preventing inconsistent-result errors and unwanted replacement), keeps unconfigured `labels` and empty `network_allow_list` null, clears `network_allow_list` server-side when removed from configuration, and reconciles `desired_state` with the actual sandbox state on refresh so out-of-band stops surface as a plan diff.
- `daytona_runner.draining` is now tracked in state instead of write-only; previously, changing only `draining` produced an empty plan and the drain request was never sent.
- `daytona_api_key` no longer stores the masked key value from the list endpoint after import; `value` stays null when the real key is unavailable.
- All API calls now have a 5-minute timeout instead of hanging indefinitely on a stalled connection.
- The `daytona_sandboxes` and `daytona_snapshots` data sources paginate through all results instead of silently truncating at 100 items.
- `daytona_organization_region_quota` destroy now warns that the quota remains active, since Daytona's organization API has no quota delete endpoint.
- The provider warns when `DAYTONA_ACCESS_TOKEN` overrides an explicitly configured `api_key`.

NOTES:

- Release checksums are now GPG-signed, as required for Terraform Registry publication.
- Added unit test coverage for the previously untested resources.

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
