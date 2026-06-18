terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
  }
}

provider "daytona" {}

# Registry credentials so Daytona can pull private golden images.
resource "daytona_docker_registry" "this" {
  for_each = var.registries

  name     = each.key
  url      = each.value.url
  username = each.value.username
  password = var.registry_passwords[each.key]
  project  = each.value.project
}

locals {
  # Snapshot names carry the image tag (for example "python-agent-3.12.1"),
  # so publishing a new version creates a new snapshot alongside the old one
  # and rollback is a one-line change in the SDK call.
  snapshot_versions = merge([
    for snapshot_key, snapshot in var.snapshots : {
      for version in snapshot.versions :
      "${snapshot_key}-${version}" => {
        image      = "${snapshot.image}:${version}"
        cpu        = snapshot.cpu
        memory     = snapshot.memory
        disk       = snapshot.disk
        entrypoint = snapshot.entrypoint
      }
    }
  ]...)
}

resource "daytona_snapshot" "this" {
  for_each = local.snapshot_versions

  name       = each.key
  image_name = each.value.image
  cpu        = each.value.cpu
  memory     = each.value.memory
  disk       = each.value.disk
  entrypoint = each.value.entrypoint

  depends_on = [daytona_docker_registry.this]
}

# Shared volumes that SDK-created sandboxes mount for datasets, caches, or
# model weights that outlive any single sandbox.
resource "daytona_volume" "shared" {
  for_each = var.shared_volumes

  name = each.key
}

# Full snapshot inventory, including snapshots published outside Terraform.
data "daytona_snapshots" "all" {}
