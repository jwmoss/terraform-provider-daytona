terraform {
  required_providers {
    daytona = {
      source = "jwmoss/daytona"
    }
  }
}

provider "daytona" {}

# AFTER: the region registration that replaces `terracurl_request "region"` in
# daytonaio/terraform-modules. The proxy, SSH gateway, and snapshot-manager
# services are still deployed by your existing infra (the AWS ECS/ALB/S3
# resources in that module); only the API registration moves here, where it
# gains real state, drift detection, and a clean destroy.
resource "daytona_region" "this" {
  name                 = var.region_name
  proxy_url            = var.proxy_url
  ssh_gateway_url      = var.ssh_gateway_url
  snapshot_manager_url = var.snapshot_manager_url

  # Rotate all three control-plane credentials by changing this value. The
  # terracurl version had no equivalent — it re-POSTed and hand-parsed the
  # response to get these back.
  proxy_api_key_rotation_id                = var.credential_rotation_id
  ssh_gateway_api_key_rotation_id          = var.credential_rotation_id
  snapshot_manager_credentials_rotation_id = var.credential_rotation_id
}

# AFTER: the runner registration that replaces `terracurl_request "runner"`.
# Unlike the terracurl version (skip_destroy = true), destroying this resource
# deregisters the runner from Daytona instead of orphaning it.
resource "daytona_runner" "this" {
  region_id = daytona_region.this.id
  name      = var.runner_name
  tags      = var.runner_tags
}
