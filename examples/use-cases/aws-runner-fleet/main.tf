terraform {
  required_providers {
    daytona = {
      source = "536tech/daytona"
    }
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "daytona" {}

provider "aws" {
  region = var.aws_region
}

# Latest Canonical Ubuntu 22.04 image for the runner hosts.
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# Register each host as a Daytona runner FIRST. The one-time api_key Daytona
# returns is what the EC2 instance below boots with, so the machine that comes
# up is already the runner Daytona expects — no manual join step.
#
# region_id comes from the self-hosted-region example or the Daytona dashboard.
resource "daytona_runner" "this" {
  for_each = var.runners

  region_id = var.region_id
  name      = each.key
  tags      = each.value.tags

  # Set draining = true before removing a runner from var.runners so in-flight
  # sandboxes migrate off the host before Terraform destroys the instance.
  draining = each.value.draining
}

# Render each runner's bootstrap from the credentials Daytona just generated.
locals {
  user_data = {
    for name, runner in daytona_runner.this :
    name => templatefile("${path.module}/templates/runner-init.sh.tpl", {
      runner_id   = runner.id
      runner_name = runner.name
      api_key     = runner.api_key
      region      = runner.region
      api_url     = var.daytona_api_url
    })
  }
}

# One EC2 host per runner. user_data carries the runner identity and key; it
# lands in instance metadata, so restrict metadata access with a security group
# and IMDSv2, or swap to a Secrets Manager pull in the template for stricter
# environments.
resource "aws_instance" "runner" {
  for_each = var.runners

  ami                    = data.aws_ami.ubuntu.id
  instance_type          = each.value.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = var.security_group_ids
  user_data              = local.user_data[each.key]

  metadata_options {
    http_tokens = "required" # IMDSv2 only
  }

  tags = {
    Name            = "daytona-runner-${each.key}"
    DaytonaRunnerId = daytona_runner.this[each.key].id
    DaytonaRegion   = var.region_id
  }
}
