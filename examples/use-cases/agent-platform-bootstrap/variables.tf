variable "daytona_api_url" {
  description = "Daytona API URL. Defaults to the managed service; override for self-hosted."
  type        = string
  default     = "https://app.daytona.io/api"
}

variable "organization_id" {
  description = "Organization the quota and OpenTelemetry config apply to."
  type        = string
}

variable "region_id" {
  description = "Region the agent fleet's sandboxes run in, and that the quota caps."
  type        = string
}

variable "platform_name" {
  description = "Short prefix for the snapshot, registry, and API key this stack creates."
  type        = string
  default     = "agents"
}

variable "registry" {
  description = "Private registry for the golden image. Set to null for a public image."
  type = object({
    url      = string
    username = string
    password = string
    project  = optional(string)
  })
  default   = null
  sensitive = true
}

variable "golden_snapshot" {
  description = "The base image agent sandboxes fork from."
  type = object({
    image      = string
    version    = string
    cpu        = optional(number)
    memory     = optional(number)
    disk       = optional(number)
    entrypoint = optional(list(string))
  })

  default = {
    image   = "ghcr.io/example/python-agent"
    version = "1.5.0"
    cpu     = 2
    memory  = 4
    disk    = 10
  }
}

variable "quota" {
  description = "Per-region compute ceiling for the agent fleet. sandbox_class is one of linux-vm, container, android, windows."
  type = object({
    sandbox_class = optional(string, "linux-vm")
    total_cpu     = number
    total_memory  = number
    total_disk    = number
    total_gpu     = optional(number, 0)
  })

  default = {
    total_cpu    = 256
    total_memory = 1024
    total_disk   = 2048
  }
}

variable "otel_endpoint" {
  description = "OpenTelemetry collector endpoint for sandbox telemetry. Null disables export."
  type        = string
  default     = null
}

variable "otel_headers" {
  description = "Optional headers sent to the OpenTelemetry collector."
  type        = map(string)
  default     = null
  sensitive   = true
}

variable "agent_key_expires_at" {
  description = "RFC3339 expiry for the agent-runtime API key. Required so the key rotates."
  type        = string
}
