terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
  }
}

provider "daytona" {}

# Custom roles defined once, referenced by name from members and invitations
# below so adding a person never requires copying permission lists around.
resource "daytona_organization_role" "this" {
  for_each = var.roles

  organization_id = var.organization_id
  name            = each.key
  description     = each.value.description
  permissions     = each.value.permissions
}

# Existing members: organization role plus any custom roles, resolved from
# the role resources above by name.
resource "daytona_organization_member_access" "this" {
  for_each = var.members

  organization_id = var.organization_id
  user_id         = each.key
  role            = each.value.role
  assigned_role_ids = [
    for role_name in each.value.custom_roles : daytona_organization_role.this[role_name].id
  ]

  lifecycle {
    precondition {
      condition     = alltrue([for r in each.value.custom_roles : contains(keys(var.roles), r)])
      error_message = "Member ${each.key} references a custom role that is not defined in var.roles."
    }
  }
}

# Pending invitations follow the same shape, so a person moves from
# var.invitations to var.members with the same roles once they accept.
resource "daytona_organization_invitation" "this" {
  for_each = var.invitations

  organization_id = var.organization_id
  email           = each.key
  role            = each.value.role
  expires_at      = each.value.expires_at
  assigned_role_ids = [
    for role_name in each.value.custom_roles : daytona_organization_role.this[role_name].id
  ]
}

# Per-sandbox-class compute ceilings for each region the organization uses.
resource "daytona_organization_region_quota" "this" {
  for_each = var.region_quotas

  organization_id    = var.organization_id
  region_id          = each.value.region_id
  sandbox_class      = each.value.sandbox_class
  total_cpu_quota    = each.value.total_cpu
  total_memory_quota = each.value.total_memory
  total_disk_quota   = each.value.total_disk
  total_gpu_quota    = each.value.total_gpu
}

# Ship sandbox telemetry to the team's OpenTelemetry collector.
resource "daytona_organization_otel_config" "this" {
  count = var.otel_endpoint != null ? 1 : 0

  organization_id = var.organization_id
  endpoint        = var.otel_endpoint
  headers         = var.otel_headers
}

# Live membership, for auditing people who exist in Daytona but are not yet
# managed here.
data "daytona_organization_members" "current" {
  organization_id = var.organization_id
}
