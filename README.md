# Terraform Provider for Daytona

This repository contains a Terraform Plugin Framework provider for [Daytona](https://github.com/daytonaio/daytona). It lets teams manage Daytona sandboxes and supporting Daytona infrastructure with the same Terraform workflows they use for AWS, Azure, GCP, and other enterprise platform dependencies.

## Features

- Provider configuration through `DAYTONA_API_KEY`, `DAYTONA_API_URL`, and `DAYTONA_ORGANIZATION_ID`
- Daytona managed-service default API URL: `https://app.daytona.io/api`
- Resources:
  - `daytona_api_key`
  - `daytona_docker_registry`
  - `daytona_region`
  - `daytona_runner`
  - `daytona_sandbox`
  - `daytona_snapshot`
  - `daytona_volume`
- Data sources:
  - `daytona_current_api_key`
  - `daytona_docker_registries`
  - `daytona_regions`
  - `daytona_runners`
  - `daytona_sandboxes`
  - `daytona_snapshots`
  - `daytona_volumes`

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
