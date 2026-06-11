# Terraform Provider for Daytona

This repository contains a Terraform Plugin Framework provider for [Daytona](https://github.com/daytonaio/daytona). It lets teams manage Daytona sandboxes and supporting Daytona infrastructure with the same Terraform workflows they use for AWS, Azure, GCP, and other enterprise platform dependencies.

## Features

- Provider configuration through `DAYTONA_API_KEY`, `DAYTONA_ACCESS_TOKEN`, `DAYTONA_API_URL`, and `DAYTONA_ORGANIZATION_ID`
- Daytona managed-service default API URL: `https://app.daytona.io/api`
- Resources:
  - `daytona_api_key`
  - `daytona_docker_registry`
  - `daytona_organization`
  - `daytona_organization_invitation`
  - `daytona_organization_member_access`
  - `daytona_organization_otel_config`
  - `daytona_organization_role`
  - `daytona_region`
  - `daytona_runner`
  - `daytona_sandbox`
  - `daytona_snapshot`
  - `daytona_volume`
- Actions:
  - `daytona_accept_organization_invitation`
  - `daytona_activate_snapshot`
  - `daytona_create_sandbox_backup`
  - `daytona_create_sandbox_snapshot`
  - `daytona_deactivate_snapshot`
  - `daytona_decline_organization_invitation`
  - `daytona_expire_sandbox_signed_port_preview_url`
  - `daytona_fork_sandbox`
  - `daytona_initialize_webhooks`
  - `daytona_leave_organization`
  - `daytona_recover_sandbox`
  - `daytona_refresh_webhook_endpoints`
  - `daytona_revoke_sandbox_ssh_access`
  - `daytona_suspend_organization`
  - `daytona_unsuspend_organization`
  - `daytona_update_sandbox_last_activity`
- Data sources:
  - `daytona_account_providers`
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
  go test ./internal/provider -run 'TestAcc(CurrentAPIKeyDataSource|VolumeResource)_basic' -v
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

Acceptance tests create real Daytona resources. Volume create/delete was verified live after adding lifecycle polling for Daytona's asynchronous volume states. The full org/user suite requires an OAuth access token because the current Daytona API rejects normal API keys on JWT-only routes.

Generate provider documentation:

```shell
make generate
```

Validate release packaging:

```shell
make release-check
make release-snapshot
```

## Repository Status

This provider was scaffolded from `hashicorp/terraform-provider-scaffolding-framework` and then converted to a Daytona-specific provider module at `github.com/jwmoss/terraform-provider-daytona`.
