data "daytona_volume" "example" {
  id = "volume-id"
}

output "daytona_volume_state" {
  value = data.daytona_volume.example.state
}
