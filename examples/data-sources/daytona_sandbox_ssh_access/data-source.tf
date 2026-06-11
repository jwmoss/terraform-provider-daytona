data "daytona_sandbox_ssh_access" "example" {
  sandbox_id_or_name = "my-sandbox"
  expires_in_minutes = 30
}
