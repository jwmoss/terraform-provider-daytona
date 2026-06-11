data "daytona_sandbox_id_from_signed_preview_token" "example" {
  signed_preview_token = "signed-preview-token"
  port                 = 3000
}
