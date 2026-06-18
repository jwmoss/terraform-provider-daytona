locals {
  common_tags = merge(var.tags, {
    Example   = "azure-runner-fleet"
    ManagedBy = "Terraform"
    Project   = "daytona"
  })

  effective_region_id = var.create_region ? daytona_region.this[0].id : coalesce(var.region_id, "")
  region_name         = coalesce(var.region_name, "${var.name_prefix}-${var.azure_location}")

  runner_cloud_init = {
    for name, runner in daytona_runner.this :
    name => templatefile("${path.module}/templates/runner-cloud-init.yaml.tpl", {
      api_key         = runner.api_key
      api_url         = var.daytona_api_url
      package_update  = var.cloud_init_package_update
      package_upgrade = var.cloud_init_package_upgrade
      poll_limit      = var.poll_limit
      poll_timeout    = var.poll_timeout
      runner_id       = runner.id
      runner_name     = runner.name
      runner_region   = runner.region
      runner_version  = var.runner_version
    })
  }
}
