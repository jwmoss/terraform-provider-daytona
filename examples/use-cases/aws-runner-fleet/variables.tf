variable "aws_region" {
  description = "AWS region the runner hosts are launched in."
  type        = string
  default     = "us-east-1"
}

variable "daytona_api_url" {
  description = "Daytona API URL the runner daemon reports to."
  type        = string
  default     = "https://app.daytona.io/api"
}

variable "region_id" {
  description = "Daytona region the runners join (from the self-hosted-region example or the dashboard)."
  type        = string
}

variable "subnet_id" {
  description = "Subnet the runner instances launch in."
  type        = string
}

variable "security_group_ids" {
  description = "Security groups for the runner instances."
  type        = list(string)
}

variable "runners" {
  description = "Runner hosts to launch, keyed by runner name. Each becomes one Daytona runner and one EC2 instance."
  type = map(object({
    tags          = list(string)
    instance_type = optional(string, "t3.large")
    draining      = optional(bool, false)
  }))

  default = {
    runner-1 = {
      tags          = ["terraform", "general"]
      instance_type = "t3.large"
    }
    runner-2 = {
      tags          = ["terraform", "gpu"]
      instance_type = "g4dn.xlarge"
    }
  }
}
