resource "daytona_admin_organization_region_quota" "example" {
  organization_id    = "organization-id"
  region_id          = "region-id"
  sandbox_class      = "container"
  total_cpu_quota    = 32
  total_memory_quota = 64
  total_disk_quota   = 1024
  total_gpu_quota    = 0
}
