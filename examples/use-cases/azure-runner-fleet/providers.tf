provider "azurerm" {
  subscription_id = var.azure_subscription_id

  features {}
}

provider "daytona" {
  api_url = var.daytona_api_url
}
