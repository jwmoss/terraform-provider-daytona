resource "daytona_organization_otel_config" "example" {
  organization_id = "organization-id"
  endpoint        = "https://otel.example.com/v1/traces"

  headers = {
    Authorization = "Bearer token"
  }
}
