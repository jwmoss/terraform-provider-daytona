action "daytona_create_sandbox_snapshot" "example" {
  config {
    sandbox_id_or_name = "sandbox-id-or-name"
    name               = "snapshot-name"
    include_memory     = false
  }
}
