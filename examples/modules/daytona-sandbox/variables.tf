variable "name" {
  type        = string
  description = "Daytona sandbox name."
}

variable "snapshot" {
  type        = string
  description = "Daytona snapshot ID or name used to create the sandbox."
  default     = "daytonaio/sandbox:0.6.0"
}

variable "desired_state" {
  type        = string
  description = "Desired sandbox lifecycle state: started, stopped, or archived."
  default     = "started"
}

variable "target" {
  type        = string
  description = "Optional target region for the sandbox."
  default     = null
}

variable "user" {
  type        = string
  description = "Optional user associated with the sandbox project."
  default     = null
}

variable "labels" {
  type        = map(string)
  description = "Labels to apply to the sandbox."
  default = {
    managed-by = "terraform"
  }
}

variable "env" {
  type        = map(string)
  description = "Environment variables for the sandbox. Values are stored as sensitive Terraform state."
  default     = {}
  sensitive   = true
}

variable "cpu" {
  type        = number
  description = "Optional CPU cores allocated to the sandbox."
  default     = null
}

variable "memory" {
  type        = number
  description = "Optional memory allocated to the sandbox in GB."
  default     = null
}

variable "disk" {
  type        = number
  description = "Optional disk allocated to the sandbox in GB."
  default     = null
}

variable "gpu" {
  type        = number
  description = "Optional GPU units allocated to the sandbox."
  default     = null
}

variable "public" {
  type        = bool
  description = "Whether HTTP previews are publicly accessible."
  default     = null
}

variable "auto_stop_interval" {
  type        = number
  description = "Optional auto-stop interval in minutes. Use 0 to disable."
  default     = null
}

variable "auto_archive_interval" {
  type        = number
  description = "Optional auto-archive interval in minutes."
  default     = null
}

variable "auto_delete_interval" {
  type        = number
  description = "Optional auto-delete interval in minutes. Negative values disable auto-delete."
  default     = null
}

variable "network_block_all" {
  type        = bool
  description = "Whether to block all sandbox network access."
  default     = null
}

variable "network_allow_list" {
  type        = string
  description = "Optional comma-separated list of allowed CIDR network addresses."
  default     = null
}

variable "linked_sandbox" {
  type        = string
  description = "Optional existing sandbox ID or name to link the new sandbox to."
  default     = null
}

variable "create_volume" {
  type        = bool
  description = "Whether to create a companion Daytona persistent volume."
  default     = false
}

variable "volume_name" {
  type        = string
  description = "Optional name for the companion Daytona volume. Defaults to '<name>-workspace'."
  default     = null
}
