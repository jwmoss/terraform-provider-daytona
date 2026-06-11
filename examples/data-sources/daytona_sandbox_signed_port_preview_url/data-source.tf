data "daytona_sandbox_signed_port_preview_url" "example" {
  sandbox_id_or_name = "my-sandbox"
  port               = 3000
  expires_in_seconds = 300
}
