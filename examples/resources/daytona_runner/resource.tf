resource "daytona_runner" "example" {
  region_id = daytona_region.example.id
  name      = "example-runner"

  tags = ["terraform"]
}
