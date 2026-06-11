data "daytona_sandbox" "example" {
  sandbox_id_or_name = "sandbox-id-or-name"
}

output "daytona_sandbox_state" {
  value = data.daytona_sandbox.example.state
}
