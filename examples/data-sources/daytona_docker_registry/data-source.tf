data "daytona_docker_registry" "example" {
  id = "registry-id"
}

output "daytona_docker_registry_url" {
  value = data.daytona_docker_registry.example.url
}
