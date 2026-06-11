data "daytona_runner" "example" {
  id = "runner-id"
}

output "daytona_runner_state" {
  value = data.daytona_runner.example.state
}
