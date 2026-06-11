# Daytona Sandbox Module

This example module provisions a Daytona sandbox and, optionally, a companion Daytona persistent volume.

Use it from the public GitHub repository:

```terraform
module "daytona_sandbox" {
  source = "github.com/jwmoss/terraform-provider-daytona//examples/modules/daytona-sandbox?ref=v0.1.0"

  name          = "agent-runtime"
  snapshot      = "daytonaio/sandbox:0.6.0"
  desired_state = "started"

  labels = {
    managed-by = "terraform"
    workload   = "agent"
  }
}
```

Until the provider is published to Terraform Registry, install the provider binary from GitHub and configure a Terraform development override for `jwmoss/daytona`; see the repository root README for the exact command.
