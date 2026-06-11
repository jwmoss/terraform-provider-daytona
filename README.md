# Terraform Provider for Daytona

This repository contains a Terraform Plugin Framework provider for [Daytona](https://github.com/daytonaio/daytona). It lets teams manage Daytona sandboxes and supporting Daytona infrastructure with the same Terraform workflows they use for AWS, Azure, GCP, and other enterprise platform dependencies.

## Features

- Provider configuration through `DAYTONA_API_KEY`, `DAYTONA_API_URL`, and `DAYTONA_ORGANIZATION_ID`
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
- Data sources:
  - `daytona_account_providers`
  - `daytona_api_key`
  - `daytona_api_keys`
  - `daytona_config`
  - `daytona_current_api_key`
  - `daytona_current_user`
  - `daytona_docker_registries`
  - `daytona_docker_registry`
  - `daytona_docker_registry_push_access`
  - `daytona_job`
  - `daytona_jobs`
  - `daytona_object_storage_push_access`
  - `daytona_organization_audit_logs`
  - `daytona_organization_invitation`
  - `daytona_organization_invitations`
  - `daytona_organization_member`
  - `daytona_organization_members`
  - `daytona_organization_otel_config`
  - `daytona_organization_role`
  - `daytona_organization_roles`
  - `daytona_organization_usage`
  - `daytona_organization`
  - `daytona_organizations`
  - `daytona_region`
  - `daytona_regions`
  - `daytona_runner`
  - `daytona_runners`
  - `daytona_sandbox_ancestors`
  - `daytona_sandbox_build_logs_url`
  - `daytona_sandbox_forks`
  - `daytona_sandbox_organization`
  - `daytona_sandbox_parent`
  - `daytona_sandbox_port_preview_url`
  - `daytona_sandbox_region_quota`
  - `daytona_sandbox_ssh_access`
  - `daytona_sandbox`
  - `daytona_sandboxes`
  - `daytona_sandbox_toolbox_proxy_url`
  - `daytona_shared_regions`
  - `daytona_snapshot_build_logs_url`
  - `daytona_snapshot`
  - `daytona_snapshots`
  - `daytona_volume`
  - `daytona_volumes`
  - `daytona_webhook_app_portal_access`
  - `daytona_webhook_initialization_status`

The provider is backed by Daytona's generated Go OpenAPI client: `github.com/daytonaio/daytona/libs/api-client-go`.

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
export DAYTONA_API_URL="https://app.daytona.io/api"
```

## Development

Requirements:

- Go 1.25 or newer
- Terraform 1.0 or newer

Run the local test suite:

```shell
go test ./...
```

Run read-only live acceptance tests:

```shell
TF_ACC=1 DAYTONA_API_KEY="dtn_..." \
  go test ./internal/provider -run 'TestAcc(CurrentAPIKeyDataSource_basic|CollectionDataSources_basic)' -v
```

Run the full acceptance suite:

```shell
TF_ACC=1 DAYTONA_API_KEY="dtn_..." go test ./internal/provider -v
```

Acceptance tests create real Daytona resources. The current live test key can read Daytona metadata, but create/delete resource tests are blocked until the Daytona organization has a payment method; the API currently returns `Organization is suspended: Payment method required` for volume creation.

Generate provider documentation:

```shell
make generate
```

## Repository Status

This provider was scaffolded from `hashicorp/terraform-provider-scaffolding-framework` and then converted to a Daytona-specific provider module at `github.com/jwmoss/terraform-provider-daytona`.
