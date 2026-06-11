action "daytona_admin_send_webhook" "example" {
  config {
    organization_id = "organization-id"
    event_type      = "sandbox.created"
    payload_json    = jsonencode({ sandboxId = "sandbox-id" })
  }
}
