output "resource_group_name" {
  description = "Azure resource group containing the runner fleet."
  value       = azurerm_resource_group.main.name
}

output "runner_ids" {
  description = "Runner name to Daytona runner ID."
  value       = { for name, runner in daytona_runner.this : name => runner.id }
}

output "runner_private_ips" {
  description = "Runner name to Azure private IP address."
  value       = { for name, nic in azurerm_network_interface.runner : name => nic.private_ip_address }
}

output "runner_public_ips" {
  description = "Runner name to Azure public IP address."
  value       = { for name, ip in azurerm_public_ip.runner : name => ip.ip_address }
}

output "runner_vm_ids" {
  description = "Runner name to Azure VM ID."
  value       = { for name, vm in azurerm_linux_virtual_machine.runner : name => vm.id }
}

output "ssh_commands" {
  description = "SSH commands for runner VMs when enable_ssh is true and a public IP is assigned."
  value = {
    for name, ip in azurerm_public_ip.runner :
    name => "ssh ${var.admin_username}@${ip.ip_address}"
  }
}

output "region_id" {
  description = "Daytona region ID used by the runner fleet."
  value       = local.effective_region_id
}
