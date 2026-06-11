data "daytona_snapshot" "example" {
  id = "snapshot-id"
}

output "daytona_snapshot_state" {
  value = data.daytona_snapshot.example.state
}
