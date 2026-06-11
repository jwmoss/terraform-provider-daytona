resource "daytona_admin_runner" "example" {
  region_id   = "region-id"
  name        = "example-admin-runner"
  api_key     = "runner-api-key"
  api_version = "2"

  domain    = "runner.example.com"
  api_url   = "https://api.runner.example.com"
  proxy_url = "https://proxy.runner.example.com"

  cpu        = 8
  memory_gib = 16
  disk_gib   = 100

  tags = ["terraform"]
}
