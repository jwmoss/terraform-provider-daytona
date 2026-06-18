terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "daytona" {}

provider "aws" {
  region = var.aws_region
}

# One ECR repository per golden image family, with immutable tags so a
# published version can never be overwritten under a running snapshot.
resource "aws_ecr_repository" "golden" {
  for_each = var.snapshots

  name                 = "${var.repository_prefix}/${each.key}"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

# Registry-wide pull credentials. ECR tokens last 12 hours, so this data source
# re-fetches a fresh credential on every apply (see README for keeping it live).
data "aws_ecr_authorization_token" "this" {}

# Point Daytona at the account's ECR registry once. Every repository above
# lives under the same host, so a single credential covers them all.
resource "daytona_docker_registry" "ecr" {
  name     = "${var.repository_prefix}-ecr"
  url      = replace(data.aws_ecr_authorization_token.this.proxy_endpoint, "https://", "")
  username = data.aws_ecr_authorization_token.this.user_name
  password = data.aws_ecr_authorization_token.this.password
}

locals {
  # Expand each family's versions into individual snapshots named
  # "<family>-<version>", each pinned to the matching immutable ECR tag.
  snapshot_versions = merge([
    for family_key, family in var.snapshots : {
      for version in family.versions :
      "${family_key}-${version}" => {
        image      = "${aws_ecr_repository.golden[family_key].repository_url}:${version}"
        cpu        = family.cpu
        memory     = family.memory
        disk       = family.disk
        entrypoint = family.entrypoint
      }
    }
  ]...)
}

# The golden snapshots SDK-created sandboxes fork from. Daytona pulls each image
# through the registry credential above; the image must already be pushed to ECR
# (CI's job) before the snapshot can build.
resource "daytona_snapshot" "golden" {
  for_each = local.snapshot_versions

  name       = each.key
  image_name = each.value.image
  cpu        = each.value.cpu
  memory     = each.value.memory
  disk       = each.value.disk
  entrypoint = each.value.entrypoint

  depends_on = [daytona_docker_registry.ecr]
}
