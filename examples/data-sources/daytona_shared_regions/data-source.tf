data "daytona_shared_regions" "available" {}

output "daytona_shared_region_ids" {
  value = [for region in data.daytona_shared_regions.available.items : region.id]
}
