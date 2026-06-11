resource "daytona_sandbox" "example" {
  name          = "example-sandbox"
  snapshot      = "daytonaio/sandbox:0.6.0"
  desired_state = "started"

  labels = {
    managed-by = "terraform"
  }
}
