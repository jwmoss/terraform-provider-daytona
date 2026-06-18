# Azure runner fleet

Launches Azure VM hosts and registers each as a Daytona runner in the same
apply. This is the Azure equivalent of [aws-runner-fleet](../aws-runner-fleet):
the Daytona provider owns the region and runner registrations, and `azurerm`
owns the Azure resource group, network, public IPs, NICs, and Linux VMs.

## What this tests

This example is meant for local apply/destroy smoke testing of the provider's
self-hosted surfaces:

1. `daytona_region` creates a customer region when `create_region = true`.
2. `daytona_runner` registers each runner and returns its one-time `api_key`.
3. `azurerm_linux_virtual_machine` boots with cloud-init and installs the
   Daytona runner package.
4. `terraform destroy` removes the VMs, public IPs, network, runner
   registrations, and optionally the region registration.

The managed Daytona cloud currently does not serve the custom region/runner
management routes for ordinary accounts. Use this against a self-hosted Daytona
API where those routes are enabled, or set `create_region = false` and pass an
existing region ID from that environment.

## Prerequisites

- Terraform >= 1.14
- Azure CLI authenticated with `az login`
- `ARM_SUBSCRIPTION_ID` exported, or pass `azure_subscription_id`
- A self-hosted Daytona API token with custom region and runner permissions
- An SSH public key for the Azure VM admin user

For local provider development, build this provider and use a Terraform CLI
dev override before running the example:

```hcl
provider_installation {
  dev_overrides {
    "jwmoss/daytona" = "/Users/jwmoss/github/terraform-provider-daytona"
  }

  direct {}
}
```

## Usage

```sh
cd examples/use-cases/azure-runner-fleet

export DAYTONA_API_KEY="dtn_..."
export ARM_SUBSCRIPTION_ID="$(az account show --query id -o tsv)"

terraform init
terraform apply \
  -var 'daytona_api_url=https://daytona.example.com/api' \
  -var "admin_ssh_public_key=$(cat ~/.ssh/id_ed25519.pub)"
```

To join an existing region instead of creating one:

```sh
terraform apply \
  -var 'create_region=false' \
  -var 'region_id=region_...' \
  -var 'daytona_api_url=https://daytona.example.com/api' \
  -var "admin_ssh_public_key=$(cat ~/.ssh/id_ed25519.pub)"
```

Destroy everything created by the example:

```sh
terraform destroy \
  -var 'daytona_api_url=https://daytona.example.com/api' \
  -var "admin_ssh_public_key=$(cat ~/.ssh/id_ed25519.pub)"
```

## Runner access

SSH is closed by default. To open it from your current public IP:

```sh
terraform apply \
  -var 'enable_ssh=true' \
  -var "ssh_ingress_sources={home=\"$(curl -fsS https://ifconfig.me)/32\"}" \
  -var 'daytona_api_url=https://daytona.example.com/api' \
  -var "admin_ssh_public_key=$(cat ~/.ssh/id_ed25519.pub)"
```

Then use the `ssh_commands` output. On the VM, useful checks are:

```sh
sudo cloud-init status --long
sudo journalctl -u daytona-runner -n 100 --no-pager
sudo systemctl status daytona-runner --no-pager
```

## Security note

The runner `api_key` is rendered into Azure VM `custom_data`, so it is present
in Terraform state. This is acceptable for a local smoke fixture, but a
production module should put the key in Key Vault and have the VM retrieve it
with managed identity during boot.
