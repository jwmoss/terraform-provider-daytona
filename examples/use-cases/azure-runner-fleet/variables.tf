variable "address_space" {
  description = "Address space for the Azure virtual network."
  type        = list(string)
  default     = ["10.42.0.0/16"]
}

variable "admin_ssh_public_key" {
  description = "SSH public key for the admin user on runner VMs."
  type        = string
}

variable "admin_username" {
  description = "Admin username for runner VMs."
  type        = string
  default     = "azureuser"
}

variable "azure_location" {
  description = "Azure region for the runner infrastructure."
  type        = string
  default     = "eastus"
}

variable "azure_subscription_id" {
  description = "Azure subscription ID. Leave null to use ARM_SUBSCRIPTION_ID or provider default authentication."
  type        = string
  default     = null
}

variable "cloud_init_package_update" {
  description = "Whether cloud-init runs package index updates before installing the runner."
  type        = bool
  default     = true
}

variable "cloud_init_package_upgrade" {
  description = "Whether cloud-init upgrades installed packages before installing the runner."
  type        = bool
  default     = false
}

variable "create_region" {
  description = "Create a Daytona region in this apply. Set false to join an existing region_id."
  type        = bool
  default     = true
}

variable "credential_rotation_id" {
  description = "Change this value to rotate region proxy, SSH gateway, and snapshot manager credentials."
  type        = string
  default     = "initial"
}

variable "daytona_api_url" {
  description = "Daytona API URL the provider uses and the runner daemon reports to."
  type        = string
  default     = "https://app.daytona.io/api"
}

variable "enable_ssh" {
  description = "Open SSH to the runner VMs from ssh_ingress_sources."
  type        = bool
  default     = false
}

variable "name_prefix" {
  description = "Prefix used for Azure resource names."
  type        = string
  default     = "daytona-azure"
}

variable "poll_limit" {
  description = "Runner job polling limit."
  type        = number
  default     = 10
}

variable "poll_timeout" {
  description = "Runner job polling timeout."
  type        = string
  default     = "30s"
}

variable "proxy_url" {
  description = "Optional Daytona proxy URL to store on the created region."
  type        = string
  default     = null
}

variable "region_id" {
  description = "Existing Daytona region ID to join when create_region is false."
  type        = string
  default     = null
}

variable "region_name" {
  description = "Name for the Daytona region created by this example."
  type        = string
  default     = null
}

variable "resource_group_name" {
  description = "Azure resource group name for the runner infrastructure."
  type        = string
  default     = "rg-daytona-azure-runner-fleet"
}

variable "runner_ingress_sources" {
  description = "Named source prefixes allowed to reach the runner API port."
  type        = map(string)
  default = {
    vnet = "VirtualNetwork"
  }
}

variable "runner_port" {
  description = "Runner API port allowed by the network security group."
  type        = number
  default     = 8080
}

variable "runner_subnet_address_prefixes" {
  description = "Address prefixes for the runner subnet."
  type        = list(string)
  default     = ["10.42.1.0/24"]
}

variable "runner_version" {
  description = "Daytona runner package version to install from GitHub releases."
  type        = string
  default     = "0.187.0"
}

variable "runners" {
  description = "Runner VMs to launch, keyed by runner name."
  type = map(object({
    tags                         = list(string)
    draining                     = optional(bool, false)
    os_disk_size_gb              = optional(number, 80)
    os_disk_storage_account_type = optional(string, "Premium_LRS")
    unschedulable                = optional(bool, false)
    vm_size                      = optional(string, "Standard_D4s_v5")
  }))

  default = {
    runner-1 = {
      tags = ["terraform", "azure", "general"]
    }
  }
}

variable "snapshot_manager_url" {
  description = "Optional Daytona snapshot manager URL to store on the created region."
  type        = string
  default     = null
}

variable "ssh_gateway_url" {
  description = "Optional Daytona SSH gateway URL to store on the created region."
  type        = string
  default     = null
}

variable "ssh_ingress_sources" {
  description = "Named source prefixes allowed to SSH when enable_ssh is true."
  type        = map(string)
  default     = {}
}

variable "tags" {
  description = "Additional Azure tags."
  type        = map(string)
  default     = {}
}
