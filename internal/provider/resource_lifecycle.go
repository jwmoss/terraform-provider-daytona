package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type resourceStateSetter func(context.Context, any) diag.Diagnostics

func persistCreatedResourceState(ctx context.Context, setState resourceStateSetter, data any, diags *diag.Diagnostics) bool {
	nullUnknownModelValues(ctx, data)
	diags.Append(setState(ctx, data)...)
	return !diags.HasError()
}
