# Self-hosted region and runners

Registers Daytona regions and runners natively, replacing the `terracurl`
HTTP calls used in [daytonaio/terraform-modules](https://github.com/daytonaio/terraform-modules)
(which predate this provider) with real Terraform lifecycle management:

- **Real destroy**: removing a runner from `var.regions` deletes its
  registration instead of leaking it (`terracurl` needed `skip_destroy`).
- **Drift detection**: out-of-band changes to a runner or region show up in
  `terraform plan` instead of being invisible behind `ignore_changes`.
- **Credential rotation**: bump `var.credential_rotation_id` to regenerate the
  proxy API key, SSH gateway API key, and snapshot manager credentials in one
  apply. Each rotation persists to state the moment Daytona returns it, so a
  partial failure never loses a fresh credential.
- **Safe decommissioning**: set `draining = true` on a runner and apply,
  let workloads migrate, then remove the runner block entirely.

## Pairing with cloud infrastructure

This example owns the **Daytona API side**: region registration, runner
registration, and the one-time credentials. The compute itself (VPC, AMI, EC2
instances) still comes from your cloud provider — for AWS, the `runner` and
`region` modules in daytonaio/terraform-modules work as-is if you replace
their `terracurl_request` resources with the `daytona_region`/`daytona_runner`
resources here and feed `local.runner_cloud_init[...]` into the instance user
data:

```hcl
resource "aws_instance" "runner" {
  for_each = daytona_runner.this

  ami           = data.aws_ami.daytona_runner.id
  instance_type = "t3.large"
  user_data     = local.runner_cloud_init[each.key]
  # ...
}
```

## Usage

```sh
export DAYTONA_API_KEY="dtn_..."
terraform init && terraform apply
```

Works with OpenTofu (`tofu init && tofu apply`) using only resources and data
sources.
