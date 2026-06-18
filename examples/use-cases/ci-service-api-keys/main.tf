terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
  }
}

provider "daytona" {}

# One scoped key per consuming system. Keys are immutable in Daytona, so any
# change to permissions or expiry replaces the key — rotation is "change the
# expires_at, apply, update the consumer's secret store".
resource "daytona_api_key" "service" {
  for_each = var.service_keys

  name        = each.key
  permissions = each.value.permissions
  expires_at  = each.value.expires_at

  lifecycle {
    precondition {
      condition     = each.value.expires_at != null || each.value.allow_non_expiring
      error_message = "Key ${each.key} must set expires_at, or explicitly opt out with allow_non_expiring = true."
    }
  }
}

# Every key in the organization, for auditing keys created outside Terraform
# and keys approaching expiry.
data "daytona_api_keys" "all" {}
