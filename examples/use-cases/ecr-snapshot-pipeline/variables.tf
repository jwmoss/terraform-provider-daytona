variable "aws_region" {
  description = "AWS region the ECR repositories live in."
  type        = string
  default     = "us-east-1"
}

variable "repository_prefix" {
  description = "Prefix for the ECR repository names and the Daytona registry name."
  type        = string
  default     = "daytona-golden"
}

variable "snapshots" {
  description = "Golden image families. Each version becomes its own immutable ECR tag and Daytona snapshot named <family>-<version>."
  type = map(object({
    versions   = list(string)
    cpu        = optional(number)
    memory     = optional(number)
    disk       = optional(number)
    entrypoint = optional(list(string))
  }))

  default = {
    python-agent = {
      versions = ["1.4.0", "1.5.0"]
      cpu      = 2
      memory   = 4
      disk     = 10
    }
    node-ci = {
      versions   = ["20.11.0"]
      cpu        = 4
      memory     = 8
      disk       = 20
      entrypoint = ["sleep", "infinity"]
    }
  }
}
