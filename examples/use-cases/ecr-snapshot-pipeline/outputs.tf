output "repository_urls" {
  description = "Image family to ECR repository URL. Push your golden images here before applying snapshots."
  value       = { for name, repo in aws_ecr_repository.golden : name => repo.repository_url }
}

output "registry_host" {
  description = "ECR registry host Daytona pulls from."
  value       = daytona_docker_registry.ecr.url
}

output "snapshot_names" {
  description = "Snapshot names ready to pass to the SDK's CreateSandboxFromSnapshotParams."
  value       = sort(keys(daytona_snapshot.golden))
}

output "snapshot_states" {
  description = "Build state per snapshot; sandboxes can launch once a snapshot is active."
  value       = { for name, snapshot in daytona_snapshot.golden : name => snapshot.state }
}
