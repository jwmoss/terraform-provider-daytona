resource "azurerm_resource_group" "main" {
  name     = var.resource_group_name
  location = var.azure_location
  tags     = local.common_tags
}

resource "azurerm_virtual_network" "main" {
  name                = "${var.name_prefix}-vnet"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  address_space       = var.address_space
  tags                = local.common_tags
}

resource "azurerm_subnet" "runner" {
  name                 = "runner"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = var.runner_subnet_address_prefixes
}

resource "azurerm_network_security_group" "runner" {
  name                = "${var.name_prefix}-runner-nsg"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.common_tags
}

resource "azurerm_network_security_rule" "runner_api" {
  for_each = var.runner_ingress_sources

  name                        = "runner-api-${replace(each.key, "_", "-")}"
  priority                    = 1100 + index(sort(keys(var.runner_ingress_sources)), each.key)
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = tostring(var.runner_port)
  source_address_prefix       = each.value
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.runner.name
}

resource "azurerm_network_security_rule" "ssh" {
  for_each = var.enable_ssh ? var.ssh_ingress_sources : {}

  name                        = "ssh-${replace(each.key, "_", "-")}"
  priority                    = 1200 + index(sort(keys(var.ssh_ingress_sources)), each.key)
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "22"
  source_address_prefix       = each.value
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.runner.name
}

resource "azurerm_subnet_network_security_group_association" "runner" {
  subnet_id                 = azurerm_subnet.runner.id
  network_security_group_id = azurerm_network_security_group.runner.id
}

resource "daytona_region" "this" {
  count = var.create_region ? 1 : 0

  name                 = local.region_name
  proxy_url            = var.proxy_url
  ssh_gateway_url      = var.ssh_gateway_url
  snapshot_manager_url = var.snapshot_manager_url

  proxy_api_key_rotation_id                = var.credential_rotation_id
  ssh_gateway_api_key_rotation_id          = var.credential_rotation_id
  snapshot_manager_credentials_rotation_id = var.credential_rotation_id
}

resource "daytona_runner" "this" {
  for_each = var.runners

  region_id     = local.effective_region_id
  name          = each.key
  tags          = each.value.tags
  draining      = each.value.draining
  unschedulable = each.value.unschedulable

  lifecycle {
    precondition {
      condition     = local.effective_region_id != ""
      error_message = "Set create_region = true or provide region_id for an existing Daytona region."
    }
  }
}

resource "azurerm_public_ip" "runner" {
  for_each = var.runners

  name                = "${var.name_prefix}-${each.key}-pip"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.common_tags
}

resource "azurerm_network_interface" "runner" {
  for_each = var.runners

  name                = "${var.name_prefix}-${each.key}-nic"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.common_tags

  ip_configuration {
    name                          = "primary"
    subnet_id                     = azurerm_subnet.runner.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.runner[each.key].id
  }
}

resource "azurerm_linux_virtual_machine" "runner" {
  for_each = var.runners

  name                  = "${var.name_prefix}-${each.key}"
  resource_group_name   = azurerm_resource_group.main.name
  location              = azurerm_resource_group.main.location
  size                  = each.value.vm_size
  admin_username        = var.admin_username
  network_interface_ids = [azurerm_network_interface.runner[each.key].id]
  custom_data           = base64encode(local.runner_cloud_init[each.key])
  tags = merge(local.common_tags, {
    DaytonaRegionId = local.effective_region_id
    DaytonaRunnerId = daytona_runner.this[each.key].id
    DaytonaRunner   = each.key
  })

  admin_ssh_key {
    username   = var.admin_username
    public_key = var.admin_ssh_public_key
  }

  os_disk {
    caching              = "ReadWrite"
    disk_size_gb         = each.value.os_disk_size_gb
    storage_account_type = each.value.os_disk_storage_account_type
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  boot_diagnostics {}
}
