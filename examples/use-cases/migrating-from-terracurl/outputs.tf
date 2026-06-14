# The runner's one-time API key as a typed attribute — this replaces the old
# jsondecode(terracurl_request.runner.response).apiKey. Feed it into the runner
# host's user_data the same way (see the aws-runner-fleet example).
output "runner_api_key" {
  description = "One-time runner API key (replaces jsondecode(...).apiKey)."
  sensitive   = true
  value       = daytona_runner.this.api_key
}

output "region_id" {
  description = "Daytona region ID, for the runner and for state imports."
  value       = daytona_region.this.id
}

output "region_credentials" {
  description = "Control-plane credentials as typed attributes, instead of hand-parsed from a curl response."
  sensitive   = true
  value = {
    proxy_api_key             = daytona_region.this.proxy_api_key
    ssh_gateway_api_key       = daytona_region.this.ssh_gateway_api_key
    snapshot_manager_username = daytona_region.this.snapshot_manager_username
    snapshot_manager_password = daytona_region.this.snapshot_manager_password
  }
}
