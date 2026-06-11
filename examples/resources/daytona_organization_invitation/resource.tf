resource "daytona_organization_invitation" "example" {
  organization_id   = "organization-id"
  email             = "user@example.com"
  role              = "member"
  assigned_role_ids = []
}
