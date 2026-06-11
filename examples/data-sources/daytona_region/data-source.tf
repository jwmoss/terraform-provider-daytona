data "daytona_region" "example" {
  id = "region-id"
}

output "daytona_region_name" {
  value = data.daytona_region.example.name
}
