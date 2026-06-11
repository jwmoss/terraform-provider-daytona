output "sandbox_id" {
  description = "Daytona sandbox ID."
  value       = daytona_sandbox.this.id
}

output "sandbox_name" {
  description = "Daytona sandbox name."
  value       = daytona_sandbox.this.name
}

output "sandbox_state" {
  description = "Current Daytona sandbox state."
  value       = daytona_sandbox.this.state
}

output "sandbox_toolbox_proxy_url" {
  description = "Toolbox proxy URL for the sandbox."
  value       = daytona_sandbox.this.toolbox_proxy_url
}

output "volume_id" {
  description = "Companion Daytona volume ID, when create_volume is true."
  value       = try(daytona_volume.workspace[0].id, null)
}

output "volume_name" {
  description = "Companion Daytona volume name, when create_volume is true."
  value       = try(daytona_volume.workspace[0].name, null)
}
