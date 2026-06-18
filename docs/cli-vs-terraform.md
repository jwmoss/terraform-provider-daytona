# Daytona CLI/SDK vs. this Terraform provider

A note on where this provider adds value versus where Daytona's CLI/SDK is the
better tool. Written to revisit when deciding where to invest maintenance effort.

## The core tension

Daytona today is a **sandbox-as-infrastructure platform for AI agents** — the
pitch is sub-100ms ephemeral sandboxes created programmatically at runtime, used
to run code, then torn down. That primary workflow is **imperative and dynamic**:
an agent or app calls the SDK (Python/TS) or CLI to spin a sandbox up on demand.

Terraform's declarative "here is my desired steady state, reconcile to it" model
fights that. A sandbox created by an AI agent at runtime should not show up as
drift in `terraform plan`. So the instinct that "the compute side is CLI/SDK
driven and Terraform isn't useful there" is **correct** — but it does not apply
uniformly across what this provider covers.

## Two buckets

### Bucket A — ephemeral compute (CLI/SDK territory, weak Terraform fit)

- `daytona_sandbox`, `daytona_snapshot`, `daytona_volume`
- Most actions: `start_sandbox`, `stop_sandbox`, `archive_sandbox`,
  `fork_sandbox`, `create_sandbox_backup`, `recover_sandbox`, etc.

Lifecycle operations on short-lived objects. Managing them through Terraform
state is awkward — TF doing what the SDK does natively inside application code.
The actions in particular are one-shot imperative RPCs; Terraform's action
feature can express them, but they map far more naturally to `daytona ...` CLI
calls or an SDK call. Reimplementing `accept_organization_invitation` or
`send_webhook` as a TF action is mostly novelty.

### Bucket B — persistent control-plane / account config (genuine Terraform territory)

- `daytona_organization`, `daytona_organization_role`,
  `daytona_organization_member_access`, `daytona_organization_region_quota`
- `daytona_api_key`, `daytona_docker_registry`,
  `daytona_organization_otel_config`
- `daytona_runner`, `daytona_admin_runner`, `daytona_region`

This is where the provider earns its keep. Configure-once, reproducible state:
org structure, RBAC, registry credentials, quotas, telemetry config, runner
registration. Exactly what Terraform is good at and the CLI is bad at — versioned,
reviewed in PRs, drift-detectable, reproducible across environments, and tied
into the same TF workflow that provisions the underlying AWS/Azure resources.
The CLI can do these too, but imperatively and without state/drift/review.

## Recommendation

The provider is valuable as a **control-plane / platform-team tool**, not a
runtime sandbox tool. Its audience is the platform engineer setting up a Daytona
org for a company — RBAC, registries, quotas, runners, telemetry — alongside the
rest of their IaC. It is **not** for the AI app developer spinning sandboxes up
and down; that person should use the SDK.

Where to invest:

1. **Lead with Bucket B** in positioning — "manage your Daytona org and platform
   config as code," not "manage sandboxes with Terraform."
2. **Keep but de-emphasize Bucket A.** A `daytona_sandbox` resource is legitimate
   for long-lived *pet* sandboxes (shared dev box, persistent build environment) —
   a real niche, just not Daytona's headline use case. Document it as such.
3. **Reconsider the action sprawl.** ~30 actions is a lot of surface for one-shot
   RPCs better served by the CLI/SDK. Question whether `enroll_sms_mfa`,
   `link_account`, `send_webhook` belong in a Terraform provider at all.

## Open follow-ups

- Verify `daytona_sandbox` handles ephemerality gracefully (does it cause
  perpetual drift?).
- Consider a repositioned README intro reflecting the Bucket B framing.
