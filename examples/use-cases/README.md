# Use-case examples

Runnable configurations for the platform-infrastructure side of Daytona, where
Terraform's lifecycle management is the right tool. Ephemeral sandboxes are
usually created at runtime with the Daytona SDK; the durable objects sandboxes
depend on — regions, runners, golden snapshots, registries, volumes, roles,
quotas, and API keys — are long-lived, drift-prone, and team-owned, which is
exactly what Terraform is for.

Every example uses only resources and data sources (no Terraform 1.14
provider-defined actions), so they all work with OpenTofu as well.

| Example | What it manages |
|---|---|
| [self-hosted-region](./self-hosted-region) | Region and runner registration for bring-your-own-compute, replacing `terracurl` calls in [daytonaio/terraform-modules](https://github.com/daytonaio/terraform-modules) |
| [organization-governance](./organization-governance) | Custom roles, member access, invitations, region quotas, and OpenTelemetry export as code |
| [golden-snapshot-pipeline](./golden-snapshot-pipeline) | Private registry credentials, versioned golden snapshots, and shared volumes that SDK-created sandboxes consume |
| [ci-service-api-keys](./ci-service-api-keys) | Scoped, expiring API keys for CI systems and service accounts |
| [agent-platform-bootstrap](./agent-platform-bootstrap) | Composition example: the full durable substrate (registry, golden snapshot, region quota, OpenTelemetry, runtime key) in one apply, with outputs the agent runtime consumes |

Each example authenticates the provider through the standard environment
variables:

```sh
export DAYTONA_API_KEY="dtn_..."
export DAYTONA_ORGANIZATION_ID="..."   # for organization-scoped examples
```
