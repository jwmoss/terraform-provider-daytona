terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
  }
}

# Authenticate with DAYTONA_API_KEY (and DAYTONA_ORGANIZATION_ID for the
# organization-scoped resources below). Override api_url for self-hosted.
provider "daytona" {
  api_url = var.daytona_api_url
}

# This is the "stage" an agent platform runs on. A team applies it once, then
# points its agent runtime (the Daytona SDK) at the outputs to start spawning
# ephemeral sandboxes. Terraform owns the durable control plane here; the
# per-request sandboxes stay in the SDK at runtime, where they belong.
#
# Each slice below is the minimal form of a deeper example:
#   - registry + snapshot -> golden-snapshot-pipeline
#   - region quota        -> organization-governance
#   - otel config         -> organization-governance
#   - agent api key       -> ci-service-api-keys
# Reach for those when a slice grows past one of something.

# 1. Private registry so Daytona can pull the golden image. Skipped entirely
#    when the image is public (registry = null).
resource "daytona_docker_registry" "golden" {
  count = var.registry != null ? 1 : 0

  name     = "${var.platform_name}-golden"
  url      = var.registry.url
  username = var.registry.username
  password = var.registry.password
  project  = var.registry.project
}

# 2. The golden snapshot every agent sandbox forks from. The image tag is in
#    the snapshot name, so publishing a new image is an additive change and
#    rollback is a one-line edit to var.golden_snapshot.version.
resource "daytona_snapshot" "golden" {
  name       = "${var.platform_name}-${var.golden_snapshot.version}"
  image_name = "${var.golden_snapshot.image}:${var.golden_snapshot.version}"
  cpu        = var.golden_snapshot.cpu
  memory     = var.golden_snapshot.memory
  disk       = var.golden_snapshot.disk
  entrypoint = var.golden_snapshot.entrypoint

  depends_on = [daytona_docker_registry.golden]
}

# 3. The blast-radius control. Agent loops, RL jobs, and evals can request
#    thousands of sandboxes in parallel; this caps total compute per region so
#    a runaway run cannot exhaust the account or the budget.
resource "daytona_organization_region_quota" "agents" {
  organization_id    = var.organization_id
  region_id          = var.region_id
  sandbox_class      = var.quota.sandbox_class
  total_cpu_quota    = var.quota.total_cpu
  total_memory_quota = var.quota.total_memory
  total_disk_quota   = var.quota.total_disk
  total_gpu_quota    = var.quota.total_gpu
}

# 4. Observability from the first sandbox. Wiring OTel at the org level means
#    every sandbox the runtime spawns is traceable without per-agent setup.
resource "daytona_organization_otel_config" "agents" {
  count = var.otel_endpoint != null ? 1 : 0

  organization_id = var.organization_id
  endpoint        = var.otel_endpoint
  headers         = var.otel_headers
}

# 5. The scoped key the agent runtime authenticates with. It can manage
#    sandboxes and read the snapshot it forks from, but nothing else, so a
#    leaked runtime key cannot touch governance, registries, or other orgs.
resource "daytona_api_key" "agent_runtime" {
  name        = "${var.platform_name}-agent-runtime"
  permissions = ["write:sandboxes", "delete:sandboxes", "read:snapshots", "read:volumes"]
  expires_at  = var.agent_key_expires_at

  lifecycle {
    precondition {
      condition     = var.agent_key_expires_at != null
      error_message = "Set agent_key_expires_at so the runtime key rotates; long-lived keys are a standing risk."
    }
  }
}
