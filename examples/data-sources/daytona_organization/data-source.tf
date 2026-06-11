data "daytona_organization" "example" {
  id = "organization-id"
}

output "daytona_organization_name" {
  value = data.daytona_organization.example.name
}
