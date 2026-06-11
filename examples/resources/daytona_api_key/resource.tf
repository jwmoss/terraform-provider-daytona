resource "daytona_api_key" "example" {
  name = "terraform-example"

  permissions = [
    "write:sandboxes",
    "delete:sandboxes",
    "read:volumes",
  ]
}
