terraform {
  required_providers {
    daytona = {
      source = "jwmoss/daytona"
    }
  }
}

provider "daytona" {}

module "daytona_sandbox" {
  source = "github.com/jwmoss/terraform-provider-daytona//examples/modules/daytona-sandbox?ref=v0.4.2"

  name          = "agent-runtime"
  snapshot      = "daytonaio/sandbox:0.6.0"
  desired_state = "started"
  create_volume = true

  labels = {
    managed-by = "terraform"
    workload   = "agent"
  }
}

output "sandbox_id" {
  value = module.daytona_sandbox.sandbox_id
}

output "volume_id" {
  value = module.daytona_sandbox.volume_id
}
