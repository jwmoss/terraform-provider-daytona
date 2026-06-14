# ECR snapshot pipeline

Manages the ECR repositories that hold golden images, wires Daytona to pull from
them, and defines the snapshots agents fork from — all in one apply. This is the
AWS-native form of [golden-snapshot-pipeline](../golden-snapshot-pipeline), which
uses a generic registry.

## The chain

```
aws_ecr_repository  →  daytona_docker_registry  →  daytona_snapshot
   (the image home)      (pull credential)           (what sandboxes fork from)
```

- One ECR repository per image family, with `IMMUTABLE` tags so a published
  version can never change under a running snapshot.
- A single `daytona_docker_registry` covers every repository, since they share
  one registry host per account and region.
- Each family version expands into its own snapshot, pinned to the matching ECR
  tag. Roll forward by adding a version; the new snapshot is created alongside
  the old one, so rollback is a one-line revert.

## Images are pushed by CI, not Terraform

Terraform owns the repository, the pull wiring, and the snapshot definition. The
image itself is built and pushed by your pipeline before you apply:

```sh
aws ecr get-login-password | docker login --username AWS --password-stdin "$REGISTRY_HOST"
docker build -t "$REPO_URL:1.5.0" .
docker push "$REPO_URL:1.5.0"
```

The snapshot build pulls that image, so push the tag before applying the
snapshot that references it.

## ECR credentials expire — keep them live

`aws_ecr_authorization_token` is valid for **12 hours**. This config refreshes it
on every apply, but the credential stored in Daytona goes stale between runs.
Keep it live one of two ways:

- Run this apply on a schedule (the same pipeline that pushes images), so the
  registry credential is refreshed well inside the 12-hour window.
- Or use an ECR pull-through cache / a longer-lived pull path if your setup
  needs Daytona to pull without a recent apply.

Treat the stored credential as a short-lived cache, not a permanent secret.

## Usage

```sh
export DAYTONA_API_KEY="dtn_..."
export DAYTONA_ORGANIZATION_ID="..."

terraform init
terraform apply -var 'aws_region=us-east-1'
```

Uses the AWS provider alongside Daytona, but only resources and data sources —
works with OpenTofu.
