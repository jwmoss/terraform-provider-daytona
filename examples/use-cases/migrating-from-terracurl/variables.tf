variable "region_name" {
  description = "Unique region identifier (the terracurl module's regionName)."
  type        = string
  default     = "us-east"
}

variable "proxy_url" {
  description = "Full URL to the region's proxy service, deployed by your existing infra."
  type        = string
  default     = "https://proxy.us-east.example.com:8080"
}

variable "ssh_gateway_url" {
  description = "Region SSH gateway endpoint."
  type        = string
  default     = "ssh.us-east.example.com:22"
}

variable "snapshot_manager_url" {
  description = "Full URL to the region's snapshot-manager service."
  type        = string
  default     = "https://snapshots.us-east.example.com:5000"
}

variable "credential_rotation_id" {
  description = "Change this to rotate the proxy, SSH gateway, and snapshot-manager credentials in place."
  type        = string
  default     = "initial"
}

variable "runner_name" {
  description = "Runner name registered in the region."
  type        = string
  default     = "runner-1"
}

variable "runner_tags" {
  description = "Scheduler tags for the runner."
  type        = list(string)
  default     = ["terraform", "general"]
}
