package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func configureDaytonaClient(providerData any, terraformKind string, diags *diag.Diagnostics) *daytonaClient {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*daytonaClient)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Unexpected %s Configure Type", terraformKind),
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

func configureResourceDaytonaClient(providerData any, diags *diag.Diagnostics) *daytonaClient {
	return configureDaytonaClient(providerData, "Resource", diags)
}

func configureDataSourceClient(providerData any, diags *diag.Diagnostics) *daytonaClient {
	return configureDaytonaClient(providerData, "Data Source", diags)
}
