# AWS runner fleet

Launches the EC2 hosts for a Daytona region and registers each as a runner in
the same apply. This is the cloud half of [self-hosted-region](../self-hosted-region):
that example renders the runner bootstrap and stops; here the rendered bootstrap
becomes a real `aws_instance` user_data, so compute and its Daytona registration
land together in one state file.

## The join

The order matters and Terraform gets it right automatically:

1. `daytona_runner` registers the host and returns a one-time `api_key`.
2. That `api_key` is rendered into `templates/runner-init.sh.tpl`.
3. `aws_instance` boots with that script as `user_data`.

So the machine that comes up is already the runner Daytona expects — there is no
manual join step, and destroying the instance and deregistering the runner are a
single `terraform destroy`.

## Scaling and draining

- Add or remove hosts by editing `var.runners`; each key is one runner and one
  instance.
- Before removing a runner, set `draining = true` and apply first, so in-flight
  sandboxes migrate off the host before the instance is destroyed.
- `tags` is how schedulers target a runner (for example `gpu`); set
  `instance_type` to match (`g4dn.xlarge` for the GPU runner in the default).

## A note on the runner key

`user_data` carries the runner's `api_key` and lands in instance metadata. The
config enforces IMDSv2 (`http_tokens = required`); for stricter environments,
change the template to pull the key from Secrets Manager at boot instead of
baking it in.

## Usage

```sh
export DAYTONA_API_KEY="dtn_..."

terraform init
terraform apply \
  -var 'region_id=...' \
  -var 'subnet_id=subnet-...' \
  -var 'security_group_ids=["sg-..."]'
```

Provide an existing region (`region_id`), subnet, and security groups — this
example launches compute into your VPC rather than creating the network. The
runner install command in the template is a placeholder; point it at your
Daytona runner deployment docs.

Uses the AWS provider alongside Daytona, but only resources and data sources —
works with OpenTofu.
