package provider

import (
	"context"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func expandGPUTypes(ctx context.Context, value types.List) ([]apiclient.GpuType, diag.Diagnostics) {
	var diags diag.Diagnostics
	values, listDiags := stringList(ctx, value)
	diags.Append(listDiags...)
	if diags.HasError() || len(values) == 0 {
		return nil, diags
	}

	allowed := map[string]struct{}{}
	for _, value := range gpuTypeValues() {
		allowed[value] = struct{}{}
	}

	gpuTypes := make([]apiclient.GpuType, 0, len(values))
	for i, value := range values {
		if _, ok := allowed[value]; !ok {
			diags.AddAttributeError(
				path.Root("gpu_types").AtListIndex(i),
				"Invalid GPU type",
				fmt.Sprintf("Unsupported GPU type %q. Supported values are: %s.", value, strings.Join(gpuTypeValues(), ", ")),
			)
			continue
		}
		gpuTypes = append(gpuTypes, apiclient.GpuType(value))
	}

	return gpuTypes, diags
}
