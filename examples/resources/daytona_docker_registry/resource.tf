variable "registry_password" {
  type      = string
  sensitive = true
}

resource "daytona_docker_registry" "example" {
  name     = "example-registry"
  url      = "registry.example.com"
  username = "terraform"
  password = var.registry_password
}
