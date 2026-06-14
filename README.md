# Terraform Provider for Daytona

[![Tests](https://github.com/jwmoss/terraform-provider-daytona/actions/workflows/test.yml/badge.svg)](https://github.com/jwmoss/terraform-provider-daytona/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jwmoss/terraform-provider-daytona)](https://goreportcard.com/report/github.com/jwmoss/terraform-provider-daytona)
[![Latest Release](https://img.shields.io/github/v/release/jwmoss/terraform-provider-daytona)](https://github.com/jwmoss/terraform-provider-daytona/releases)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](LICENSE)

This repository contains a Terraform Plugin Framework provider for [Daytona](https://github.com/daytonaio/daytona). It lets teams manage Daytona sandboxes and supporting Daytona infrastructure with the same Terraform workflows they use for AWS, Azure, GCP, and other enterprise platform dependencies.

## Features

- Provider configuration through `DAYTONA_API_KEY`, `DAYTONA_ACCESS_TOKEN`, `DAYTONA_API_URL`, and `DAYTONA_ORGANIZATION_ID`
- Daytona managed-service default API URL: `https://app.daytona.io/api`
- Resources:
  - `daytona_admin_runner`
  - `daytona_admin_organization_region_quota`
  - `daytona_api_key`
  - `daytona_docker_registry`
  - `daytona_organization`
  - `daytona_organization_invitation`
  - `daytona_organization_member_access`
  - `daytona_organization_otel_config`
  - `daytona_organization_region_quota`
  - `daytona_organization_role`
  - `daytona_region`
  - `daytona_runner`
  - `daytona_sandbox`
  - `daytona_snapshot`
  - `daytona_volume`
- Actions:
  - `daytona_accept_organization_invitation`
  - `daytona_activate_snapshot`
  - `daytona_admin_create_user`
  - `daytona_admin_initialize_webhooks`
  - `daytona_admin_regenerate_user_key_pair`
  - `daytona_admin_recover_sandbox`
  - `daytona_admin_send_webhook`
  - `daytona_admin_set_default_docker_registry`
  - `daytona_admin_set_snapshot_general_status`
  - `daytona_archive_sandbox`
  - `daytona_create_sandbox_backup`
  - `daytona_create_sandbox_snapshot`
  - `daytona_deactivate_snapshot`
  - `daytona_decline_organization_invitation`
  - `daytona_enroll_sms_mfa`
  - `daytona_expire_sandbox_signed_port_preview_url`
  - `daytona_fork_sandbox`
  - `daytona_initialize_webhooks`
  - `daytona_leave_organization`
  - `daytona_link_account`
  - `daytona_recover_sandbox`
  - `daytona_refresh_webhook_endpoints`
  - `daytona_revoke_api_key_for_user`
  - `daytona_revoke_sandbox_ssh_access`
  - `daytona_start_sandbox`
  - `daytona_stop_sandbox`
  - `daytona_suspend_organization`
  - `daytona_unlink_account`
  - `daytona_unsuspend_organization`
  - `daytona_update_sandbox_last_activity`
- Data sources:
  - `daytona_account_providers`
  - `daytona_admin_audit_logs`
  - `daytona_admin_runner`
  - `daytona_admin_runners`
  - `daytona_admin_snapshot_image_cleanup_status`
  - `daytona_admin_user`
  - `daytona_admin_users`
  - `daytona_admin_webhook_message_attempts`
  - `daytona_admin_webhook_status`
  - `daytona_api_key`
  - `daytona_api_keys`
  - `daytona_authenticated_runner_sandboxes`
  - `daytona_config`
  - `daytona_current_api_key`
  - `daytona_current_user`
  - `daytona_current_user_organization_invitations`
  - `daytona_docker_registries`
  - `daytona_docker_registry`
  - `daytona_docker_registry_push_access`
  - `daytona_health`
  - `daytona_job`
  - `daytona_jobs`
  - `daytona_object_storage_push_access`
  - `daytona_organization_audit_logs`
  - `daytona_organization_invitation`
  - `daytona_organization_invitations`
  - `daytona_organization_member`
  - `daytona_organization_members`
  - `daytona_organization_otel_config`
  - `daytona_organization_otel_config_by_sandbox_auth_token`
  - `daytona_organization_role`
  - `daytona_organization_roles`
  - `daytona_organization_usage`
  - `daytona_organization`
  - `daytona_organizations`
  - `daytona_region`
  - `daytona_regions`
  - `daytona_runner`
  - `daytona_runner_for_sandbox`
  - `daytona_runner_full`
  - `daytona_runners`
  - `daytona_runners_by_snapshot_ref`
  - `daytona_sandbox_access`
  - `daytona_sandbox_ancestors`
  - `daytona_sandbox_auth_token_validation`
  - `daytona_sandbox_build_logs_url`
  - `daytona_sandbox_forks`
  - `daytona_sandbox_id_from_signed_preview_token`
  - `daytona_sandbox_logs`
  - `daytona_sandbox_metrics`
  - `daytona_sandbox_organization`
  - `daytona_sandbox_parent`
  - `daytona_sandbox_port_preview_url`
  - `daytona_sandbox_public_status`
  - `daytona_sandbox_query`
  - `daytona_sandbox_region_quota`
  - `daytona_sandbox_signed_port_preview_url`
  - `daytona_sandbox_ssh_access`
  - `daytona_sandbox_ssh_access_validation`
  - `daytona_sandbox`
  - `daytona_sandboxes`
  - `daytona_sandbox_trace_spans`
  - `daytona_sandbox_traces`
  - `daytona_sandbox_toolbox_proxy_url`
  - `daytona_shared_regions`
  - `daytona_snapshot_build_logs_url`
  - `daytona_snapshot`
  - `daytona_snapshots`
  - `daytona_volume`
  - `daytona_volume_by_name`
  - `daytona_volumes`
  - `daytona_webhook_app_portal_access`
  - `daytona_webhook_initialization_status`

The provider is backed by Daytona's generated Go OpenAPI client: `github.com/daytonaio/daytona/libs/api-client-go`.

See [docs/api-coverage.md](docs/api-coverage.md) for the current Daytona API coverage matrix and intentionally excluded endpoints.

## Example

```terraform
terraform {
  required_providers {
    daytona = {
      source = "jwmoss/daytona"
    }
  }
}

provider "daytona" {}

resource "daytona_volume" "workspace" {
  name = "workspace-cache"
}

resource "daytona_sandbox" "agent" {
  name          = "agent-runtime"
  snapshot      = "daytonaio/sandbox:0.6.0"
  desired_state = "started"

  labels = {
    managed-by = "terraform"
  }
}

data "daytona_current_api_key" "current" {}

data "daytona_organizations" "available" {}
```

Set credentials with environment variables:

```shell
export DAYTONA_API_KEY="dtn_..."
export DAYTONA_ACCESS_TOKEN="eyJ..."
export DAYTONA_API_URL="https://app.daytona.io/api"
```

Daytona API keys work for API-key-enabled routes such as current API-key lookup and volume management. Daytona org/user provisioning and discovery routes are JWT-only in the current Daytona API; set `DAYTONA_ACCESS_TOKEN` and `DAYTONA_ORGANIZATION_ID` for those routes. When both token types are set, `DAYTONA_ACCESS_TOKEN` takes precedence.

## Use From GitHub Source

Terraform provider installation uses provider addresses rather than module-style GitHub sources. Until this provider is published to Terraform Registry, install the provider binary from GitHub and point Terraform at the local build:

```shell
go install github.com/jwmoss/terraform-provider-daytona@v0.4.1

cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "jwmoss/daytona" = "$HOME/go/bin"
  }
  direct {}
}
EOF
```

Terraform configurations and modules can then use the same provider address shown above:

```terraform
terraform {
  required_providers {
    daytona = {
      source = "jwmoss/daytona"
    }
  }
}
```

This repository also includes a reusable module example that can be consumed directly from GitHub:

```terraform
module "daytona_sandbox" {
  source = "github.com/jwmoss/terraform-provider-daytona//examples/modules/daytona-sandbox?ref=v0.4.1"

  name          = "agent-runtime"
  snapshot      = "daytonaio/sandbox:0.6.0"
  desired_state = "started"
}
```

## Use Cases

Ephemeral sandboxes are usually created at runtime with the Daytona SDK; this
provider's sweet spot is the durable platform layer those sandboxes depend on.
[examples/use-cases](./examples/use-cases) contains complete configurations
for:

- **[self-hosted-region](./examples/use-cases/self-hosted-region)** — register
  bring-your-own-compute regions and runners natively, with real destroy,
  drift detection, and credential rotation (replaces the `terracurl` calls in
  [daytonaio/terraform-modules](https://github.com/daytonaio/terraform-modules)).
- **[organization-governance](./examples/use-cases/organization-governance)** —
  custom roles, member access, invitations, region quotas, and OpenTelemetry
  export as reviewable code.
- **[golden-snapshot-pipeline](./examples/use-cases/golden-snapshot-pipeline)** —
  registry credentials, versioned golden snapshots, and shared volumes that
  SDK-created sandboxes consume.
- **[ci-service-api-keys](./examples/use-cases/ci-service-api-keys)** —
  least-privilege, expiring API keys for CI systems and service accounts.

The use-case examples rely only on resources and data sources, so they work
with both Terraform and OpenTofu. Provider-defined actions are an optional
extra that require Terraform 1.14+ and are not supported by OpenTofu; where an
action has a resource equivalent (for example sandbox start/stop via the
`daytona_sandbox.desired_state` attribute), prefer the resource.

## Development

Requirements:

- Go 1.25 or newer
- Terraform 1.0 or newer; provider-defined actions require Terraform 1.14 or newer

Run the local test suite:

```shell
go test ./...
```

Install and run local hooks:

```shell
prek install
prek run --all-files
```

Run API-key live acceptance tests:

```shell
TF_ACC=1 DAYTONA_API_KEY="dtn_..." \
  go test ./internal/provider -run 'TestAcc(CurrentAPIKeyDataSource|VolumeResource|SandboxResource)_basic' -v
```

Run the opt-in resource acceptance tests. The Docker registry test needs an API key with the `WRITE_REGISTRIES` permission, and the snapshot test builds a real image:

```shell
TF_ACC=1 DAYTONA_API_KEY="dtn_..." DAYTONA_ACC_REGISTRY=1 \
  go test ./internal/provider -run TestAccDockerRegistryResource_basic -v

TF_ACC=1 DAYTONA_API_KEY="dtn_..." DAYTONA_ACC_SNAPSHOT_BUILD=1 \
  go test ./internal/provider -run TestAccSnapshotResource_basic -v
```

Run JWT-only org/user acceptance tests:

```shell
TF_ACC=1 DAYTONA_ACCESS_TOKEN="eyJ..." DAYTONA_ORGANIZATION_ID="org-..." \
  go test ./internal/provider -run 'TestAcc(CollectionDataSources|LookupDataSources|OperationalDataSources)_basic' -v
```

Run the Daytona readiness acceptance test with a health-check API key:

```shell
TF_ACC=1 DAYTONA_HEALTH_CHECK_API_KEY="dtn_..." \
  go test ./internal/provider -run TestAccHealthDataSource_basic -v
```

Acceptance tests create real Daytona resources. Volume create/delete was verified live after adding lifecycle polling for Daytona's asynchronous volume states. The full org/user suite requires an OAuth access token because the current Daytona API rejects normal API keys on JWT-only routes; Daytona CLI API-key login has the same organization-command limitation and is not a substitute for browser/OAuth authentication.

If an acceptance run is interrupted and leaves resources behind, clean up everything created with the `tf-acc-` name prefix using the sweepers:

```shell
DAYTONA_API_KEY="dtn_..." go test ./internal/provider/ -sweep=all
```

Generate provider documentation:

```shell
make generate
```

Validate release packaging:

```shell
make release-check
make release-snapshot
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build, test, documentation, and pull
request guidelines.

## Repository Status

This provider was scaffolded from `hashicorp/terraform-provider-scaffolding-framework` and then converted to a Daytona-specific provider module at `github.com/jwmoss/terraform-provider-daytona`.

## License

This provider is distributed under the [Mozilla Public License 2.0](LICENSE).
