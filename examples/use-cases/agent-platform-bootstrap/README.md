# Agent platform bootstrap

Stands up the whole durable substrate an agent platform runs on in one apply,
then hands the agent runtime exactly what it needs to start spawning
sandboxes. Terraform builds the stage; the Daytona SDK performs on it at
runtime.

This is the composition example. Each of the other use-cases owns one slice in
depth — this one wires the minimal form of all of them together so you can see
the control-plane / data-plane split end to end:

| Slice | Resource | Deeper example |
|---|---|---|
| Private registry | `daytona_docker_registry` | [golden-snapshot-pipeline](../golden-snapshot-pipeline) |
| Golden image agents fork from | `daytona_snapshot` | [golden-snapshot-pipeline](../golden-snapshot-pipeline) |
| Fleet compute ceiling | `daytona_organization_region_quota` | [organization-governance](../organization-governance) |
| Sandbox telemetry export | `daytona_organization_otel_config` | [organization-governance](../organization-governance) |
| Scoped runtime key | `daytona_api_key` | [ci-service-api-keys](../ci-service-api-keys) |

## Why Terraform here, SDK at runtime

The per-request sandbox an agent spawns, runs code in, forks, and discards is
created with the SDK — it is too short-lived and too high-volume for a plan and
apply cycle. What every one of those sandboxes depends on (the golden image,
the region quota that bounds them, the telemetry pipeline, the key the runtime
authenticates with) is long-lived, drift-prone, and team-owned. That is the
control plane, and that is what this stack manages.

## The handoff

After `terraform apply`, the outputs are the runtime's configuration:

```sh
export DAYTONA_API_URL=$(terraform output -raw daytona_api_url)
export DAYTONA_API_KEY=$(terraform output -raw agent_runtime_api_key)
export DAYTONA_SNAPSHOT=$(terraform output -raw snapshot_name)
export DAYTONA_REGION=$(terraform output -raw region_id)
```

The runtime then forks sandboxes from `$DAYTONA_SNAPSHOT` in `$DAYTONA_REGION`,
bounded by the `fleet_quota`, with telemetry flowing if `otel_enabled` is true.

## Usage

```sh
export DAYTONA_API_KEY="dtn_..."
export DAYTONA_ORGANIZATION_ID="..."

terraform init
terraform apply \
  -var 'organization_id=...' \
  -var 'region_id=...' \
  -var 'agent_key_expires_at=2026-12-31T23:59:59Z'
```

Set `registry` for a private golden image, and `otel_endpoint` to export
sandbox telemetry. Roll the image forward by bumping `golden_snapshot.version`;
the new snapshot is created alongside the old one, so rollback is a one-line
revert.

Works with OpenTofu — only resources and data sources, no provider-defined
actions.
