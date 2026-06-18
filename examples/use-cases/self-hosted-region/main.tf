terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
  }
}

provider "daytona" {}

locals {
  # Flatten regions × runners into one map so each runner gets a stable
  # address like "us-east/runner-1" that survives adding or removing peers.
  runners = merge([
    for region_key, region in var.regions : {
      for runner_name, runner in region.runners :
      "${region_key}/${runner_name}" => {
        region_key = region_key
        name       = runner_name
        tags       = runner.tags
        draining   = runner.draining
      }
    }
  ]...)
}

# Register each region's control-plane endpoints with Daytona. The returned
# proxy, SSH gateway, and snapshot manager credentials are generated once and
# stored as sensitive state.
resource "daytona_region" "this" {
  for_each = var.regions

  name                 = each.value.name
  proxy_url            = each.value.proxy_url
  ssh_gateway_url      = each.value.ssh_gateway_url
  snapshot_manager_url = each.value.snapshot_manager_url

  # Bump var.credential_rotation_id (any new string, for example a date) to
  # regenerate all three credentials in place. Each rotation is persisted to
  # state as soon as Daytona returns it.
  proxy_api_key_rotation_id                = var.credential_rotation_id
  ssh_gateway_api_key_rotation_id          = var.credential_rotation_id
  snapshot_manager_credentials_rotation_id = var.credential_rotation_id
}

# Register one runner per host. The one-time api_key feeds the runner daemon
# configuration (cloud-init, user data, or a config management tool) on the
# machine you provision with your cloud provider.
resource "daytona_runner" "this" {
  for_each = local.runners

  region_id = daytona_region.this[each.value.region_key].id
  name      = each.value.name
  tags      = each.value.tags

  # Set draining = true on a runner before removing it from var.regions so
  # in-flight sandboxes migrate off the host first.
  draining = each.value.draining

  lifecycle {
    precondition {
      condition     = length(each.value.tags) > 0
      error_message = "Every runner needs at least one tag so schedulers can target it."
    }
  }
}

# Cloud-init payload per runner, ready to hand to an aws_instance user_data,
# a google_compute_instance metadata block, or the EC2 module in
# github.com/daytonaio/terraform-modules.
locals {
  runner_cloud_init = {
    for key, runner in daytona_runner.this :
    key => templatefile("${path.module}/templates/runner-env.tpl", {
      runner_id   = runner.id
      runner_name = runner.name
      api_key     = runner.api_key
      region      = runner.region
    })
  }
}
