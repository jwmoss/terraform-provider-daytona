# Migrating from terracurl to the Daytona provider

Daytona's own [daytonaio/terraform-modules](https://github.com/daytonaio/terraform-modules)
register regions and runners with the `devops-rob/terracurl` provider — raw HTTP
`POST`s to the Daytona API — because no Terraform provider existed when they were
written. This example shows the `daytona_region` / `daytona_runner` replacement
and, more importantly, how to migrate an **already-registered** region or runner
without deregistering it.

Scope: this replaces the API **registration** only. The proxy, SSH gateway, and
snapshot-manager **services** are still deployed by the AWS ECS/ALB/S3 resources
in those modules — keep them as they are.

## Before / after

**Region** — `region/main.tf` in daytonaio/terraform-modules:

```hcl
# BEFORE
resource "terracurl_request" "region" {
  url    = "${var.daytona_api_url}/regions"
  method = "POST"
  # ...request_body assembled by hand...
}

locals {
  region_response = jsondecode(terracurl_request.region.response)
  proxy_url_regex = regex("^(https?)://([^:/]+):?([0-9]*)$", var.proxy_url)
  # ...more manual parsing to recover IDs and credentials...
}
```

```hcl
# AFTER (main.tf)
resource "daytona_region" "this" {
  name                 = var.region_name
  proxy_url            = var.proxy_url
  ssh_gateway_url      = var.ssh_gateway_url
  snapshot_manager_url = var.snapshot_manager_url
}
```

**Runner** — `runner/main.tf`:

```hcl
# BEFORE
resource "terracurl_request" "runner" {
  url          = "${var.api_url}/runners"
  method       = "POST"
  skip_destroy = true                       # destroy leaves the runner registered
  lifecycle {
    ignore_changes = [headers, request_body, destroy_headers]
  }
}
# ...jsondecode(terracurl_request.runner.response).apiKey -> instance user_data...
```

```hcl
# AFTER (main.tf)
resource "daytona_runner" "this" {
  region_id = daytona_region.this.id
  name      = var.runner_name
  tags      = var.runner_tags
}
# daytona_runner.this.api_key -> instance user_data (see aws-runner-fleet)
```

## What you gain

| terracurl in the module | The Daytona provider |
|---|---|
| `skip_destroy = true` — destroying the runner leaves it registered (orphan) | real DELETE — destroy deregisters the runner |
| `ignore_changes = [request_body, …]` — no update path, no drift detection | typed attributes with full CRUD and drift detection |
| `jsondecode(response).apiKey` | typed `daytona_runner.api_key` |
| `regex(...)` to parse `proxy_url` into the request body | typed `proxy_url` / `ssh_gateway_url` / `snapshot_manager_url` inputs |
| re-`POST` to recover credentials | `*_rotation_id` attributes rotate them in place |

## Migrating live state (no re-registration)

Your region and runner are already registered in Daytona by terracurl. You want
Terraform to **adopt** the existing objects, not create new ones — re-POSTing
would orphan in-flight sandboxes and reissue runner keys. Both resources support
`terraform import`, so the move is import-then-drop:

```sh
# 1. Find the existing IDs (from the old terracurl state)...
terraform state show terracurl_request.region | grep -i '"id"'
terraform state show terracurl_request.runner | grep -i '"id"'
#    ...or from the Daytona dashboard / GET /regions, GET /runners.

# 2. Adopt the live objects into the new resources.
terraform import daytona_region.this <region-id>
terraform import daytona_runner.this <runner-id>

# 3. Stop terracurl from managing them (does not call the API).
terraform state rm terracurl_request.region
terraform state rm terracurl_request.runner

# 4. Confirm no changes are planned against the live objects.
terraform plan
```

After step 4, `plan` should show the region and runner as managed with no diff
(adjust the `var.*` values to match what is already registered if it does not).

### Caveats, verified against the provider

- **Runner `api_key` is one-time** (set only at creation), so it is **not in
  state after import**. That is fine: the host already baked the key into its
  user_data at first boot. You only need a fresh key when you replace the host,
  which recreates the resource and issues a new key anyway.
- **Region credentials** (`proxy_api_key`, `ssh_gateway_api_key`, snapshot-manager
  user/pass) are produced by the `*_rotation_id` attributes. After import they
  are not in state until you intentionally bump a rotation id — so do **not**
  bump rotation during migration unless you mean to roll those credentials.

## Usage

```sh
export DAYTONA_API_KEY="dtn_..."

terraform init
terraform apply        # for a fresh region+runner; for migration, follow the import steps above
```

Uses only the Daytona provider — works with OpenTofu.
