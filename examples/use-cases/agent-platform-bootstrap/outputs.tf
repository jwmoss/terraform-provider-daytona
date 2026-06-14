# The handoff contract: everything the agent runtime needs to start spawning
# sandboxes. Pipe these into the runtime's environment or secret store.

output "agent_runtime_api_key" {
  description = "Scoped API key for the agent runtime. Returned once at create time; store it securely."
  sensitive   = true
  value       = daytona_api_key.agent_runtime.value
}

output "daytona_api_url" {
  description = "API URL the runtime points at (DAYTONA_API_URL)."
  value       = var.daytona_api_url
}

output "snapshot_name" {
  description = "Snapshot name to pass to the SDK's CreateSandboxFromSnapshotParams."
  value       = daytona_snapshot.golden.name
}

output "snapshot_state" {
  description = "Build state of the golden snapshot; sandboxes can launch once it is active."
  value       = daytona_snapshot.golden.state
}

output "region_id" {
  description = "Region the runtime should place sandboxes in, matching the quota below."
  value       = var.region_id
}

output "fleet_quota" {
  description = "The compute ceiling the agent fleet runs under."
  value = {
    sandbox_class = daytona_organization_region_quota.agents.sandbox_class
    cpu           = daytona_organization_region_quota.agents.total_cpu_quota
    memory        = daytona_organization_region_quota.agents.total_memory_quota
    disk          = daytona_organization_region_quota.agents.total_disk_quota
    gpu           = daytona_organization_region_quota.agents.total_gpu_quota
  }
}

output "otel_enabled" {
  description = "Whether sandbox telemetry is being exported to a collector."
  value       = length(daytona_organization_otel_config.agents) > 0
}
