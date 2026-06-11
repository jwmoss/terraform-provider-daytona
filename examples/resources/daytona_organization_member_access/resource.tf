resource "daytona_organization_member_access" "example" {
  organization_id   = "organization-id"
  user_id           = "user-id"
  role              = "member"
  assigned_role_ids = []
}
