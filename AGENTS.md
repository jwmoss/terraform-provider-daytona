# AGENTS.md

Guidance for AI agents and contributors working on the Daytona Terraform provider.

## What this is

A Terraform provider for [Daytona](https://www.daytona.io), built with the
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
and the generated `github.com/daytonaio/daytona/libs/api-client-go` SDK. All
provider code lives in `internal/provider/`.

## Commands

```bash
make build      # go build ./...
make test       # unit tests (go test -cover); no credentials needed
make lint       # golangci-lint run
make generate   # regenerate docs/ from schemas (tfplugindocs) — run after schema changes
make fmt        # gofmt -s -w
make testacc    # acceptance tests (TF_ACC=1); creates real resources — see Testing
```

Always run `make lint` and `make generate` before committing; CI fails on lint
issues and on uncommitted doc changes.

## Layout

| Path | Purpose |
|---|---|
| `internal/provider/*_resource.go` | Resources |
| `internal/provider/*_data_source*.go` | Data sources |
| `internal/provider/*_test.go` | Mocked unit tests (`httptest`) |
| `internal/provider/*_acc_test.go` | Acceptance tests (`TestAcc`, live API) |
| `internal/provider/provider.go` | Provider config + resource/data-source registration |
| `internal/provider/client.go` | Shared client, `addAPIError`, `isNotFound` |
| `internal/provider/types.go` | Conversion helpers (`optionalString`, `stringMapValue`, …) |
| `docs/` | Generated docs — never hand-edit; run `make generate` |
| `templates/`, `tools/` | tfplugindocs templates and the generate entrypoint |
| `ci/` | Local developer helpers (e.g. minting an access token) |

## Auth model (read this first)

Daytona has two auth modes against the same REST API:

- **API key** (`DAYTONA_API_KEY`): long-lived, for workload resources —
  `sandbox`, `volume`, `snapshot`, `docker_registry`.
- **Access token / JWT** (`DAYTONA_ACCESS_TOKEN` + `DAYTONA_ORGANIZATION_ID`):
  from `daytona login`, expires ~24h, required for organization-management
  resources — `organization`, `api_key`, `organization_role`,
  `organization_invitation`, `organization_member_access`,
  `organization_otel_config`. API-key auth returns `401` on these routes.

The provider also reads `DAYTONA_API_URL` (default `https://app.daytona.io/api`).
Some endpoints (`region`, `runner`, `admin_*`, quota writes) are only served by
self-hosted Daytona or platform admins; the managed cloud returns `404`/`401`.

## Adding a new resource

1. Create `internal/provider/<name>_resource.go` with a `New<Name>Resource`
   constructor and a struct implementing `Metadata`, `Schema`, `Configure`,
   `Create`, `Read`, `Update`, `Delete`, and (usually) `ImportState`. Copy the
   closest existing resource (e.g. `volume_resource.go`) and follow its shape.
2. Define a `<name>ResourceModel` struct with `tfsdk:"..."` tags.
3. Register the constructor in `Resources()` in `provider.go` (keep it alphabetical).
4. Build the API request in a small `expand…`/inline helper; map the API response
   in a `flatten…` helper. Keep CRUD methods thin.
5. Use the shared schema helpers (`requiredStringAttribute`,
   `computedStringAttribute`, `optionalComputedReplace*Attribute`, …) and error
   helpers (`addAPIError`, `isNotFound`) — match the surrounding code.
6. Add unit tests `<name>_resource_test.go` driving `httptest.NewServer` (no creds).
7. Add an acceptance test `<name>_resource_acc_test.go` (see Testing).
8. Run `make generate` to produce `docs/resources/<name>.md`, then `make lint` and
   `make test`.

Data sources follow the same pattern; register them in `DataSources()`.

### Conventions

- If the server assigns a default for an optional input (the response echoes a
  value the user did not set), make the attribute **Optional + Computed** —
  otherwise apply fails with "inconsistent result after apply".
- Set every computed attribute to a known value after create; never leave one
  unknown. If the create response omits it, set it null and let `Read` populate it.
- Absolute imports only; keep functions small; comments explain *why*, not *what*.
- Never hand-edit `docs/`; regenerate.

## Testing

### Unit tests (always, no credentials)

```bash
make test
```

Mocked `httptest` tests cover request shape and state mapping for every resource.
Add these for any new resource and any bug fix.

### Acceptance tests (live API)

Acceptance tests create and destroy real resources. They `t.Skip()` unless
`TF_ACC=1` and the right credential is set, so they are safe to leave in place.

**API-key resources** (sandbox, volume, snapshot, docker_registry):

```bash
export DAYTONA_API_KEY=...
TF_ACC=1 go test -run TestAcc ./internal/provider/
```

**Organization-API resources** (require an access token): log in once, then mint a
token with the local helper:

```bash
daytona login
export DAYTONA_ACCESS_TOKEN="$(ci/daytona-access-token.sh)"
export DAYTONA_ORGANIZATION_ID=...   # target org
TF_ACC=1 go test -run TestAcc ./internal/provider/
```

`ci/daytona-access-token.sh` exchanges the refresh token from `daytona login` for a
fresh access token. It is a local convenience only — CI does not use it.

**Opt-in gates** keep slow, costly, or environment-specific tests off the default
run; set the flag to enable:

| Flag | Enables |
|---|---|
| `DAYTONA_ACC_REGISTRY=1` | Docker registry (needs `write:registries`) |
| `DAYTONA_ACC_SNAPSHOT_BUILD=1` | Snapshot build (slow, builds an image) |
| `DAYTONA_ACC_INVITE=1` | Organization invitation (sends a real email) |
| `DAYTONA_ACC_SELF_HOSTED=1` / `_ADMIN=1` | region/runner/admin_runner (self-hosted only) |
| `DAYTONA_ACC_ORG_ADMIN=1` | region quota (platform-admin only) |
| `DAYTONA_ACC_MEMBER_ORG_ID` + `_USER_ID` | member access (needs an existing member) |

CI runs the unit tests on every push and the API-key acceptance suite on `main`.
The access-token acceptance tests are run manually before a release.

## Before opening a PR

- `make fmt lint test generate` all clean, with no uncommitted doc changes.
- New/changed behavior covered by a unit test; live behavior verified with an
  acceptance test when feasible.
- PR description states only what is in the diff.
