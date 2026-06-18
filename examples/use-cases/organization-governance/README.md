# Organization governance as code

Manages the people-and-policy layer of a Daytona organization: custom roles,
member access, invitations, per-region compute quotas, and OpenTelemetry
export. This is configuration that otherwise lives in the dashboard, drifts
silently, and has no review trail.

The role/member/invitation shapes are designed to flow into each other:

1. Define permission sets once in `var.roles`.
2. Invite someone via `var.invitations` with role names — the configuration
   resolves names to role IDs.
3. When they accept, move the same entry to `var.members` keyed by their user
   ID. Same roles, no permission copying, reviewable in the diff.

The `unmanaged_members` output lists user IDs that exist in the organization
but not in `var.members` — useful as a CI check that nobody was added out of
band.

## Usage

```hcl
module "governance" {
  source = "github.com/536tech/terraform-provider-daytona//examples/use-cases/organization-governance"

  organization_id = "your-org-id"

  members = {
    "user-id-1" = { role = "owner" }
    "user-id-2" = { role = "member", custom_roles = ["sandbox-operator"] }
  }

  invitations = {
    "new-hire@example.com" = {
      role         = "member"
      custom_roles = ["sandbox-operator", "read-only"]
      expires_at   = "2026-07-01T00:00:00Z"
    }
  }

  region_quotas = {
    us-containers = {
      region_id     = "region-id"
      sandbox_class = "container"
      total_cpu     = 64
      total_memory  = 256
      total_disk    = 2048
    }
  }

  otel_endpoint = "https://otel.example.com/v1/traces"
}
```

Works with OpenTofu — only resources and data sources, no provider-defined
actions.
