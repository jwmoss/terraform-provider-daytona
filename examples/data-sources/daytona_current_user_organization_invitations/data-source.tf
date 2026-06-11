data "daytona_current_user_organization_invitations" "example" {}

output "daytona_current_user_organization_invitations_count" {
  value = data.daytona_current_user_organization_invitations.example.total_count
}
