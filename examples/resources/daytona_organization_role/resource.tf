resource "daytona_organization_role" "example" {
  organization_id = "organization-id"
  name            = "sandbox-operator"
  description     = "Can manage Daytona sandboxes and snapshots."

  permissions = [
    "write:sandboxes",
    "write:snapshots",
  ]
}
