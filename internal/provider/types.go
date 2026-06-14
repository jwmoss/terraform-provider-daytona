package provider

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// nullUnknownModelValues replaces unknown attribute values in a resource model with nulls so a
// partially-populated model can be persisted to state before follow-up API calls run. Saving
// state as soon as the remote object exists prevents orphaning it when a later call fails.
func nullUnknownModelValues(ctx context.Context, model any) {
	v := reflect.ValueOf(model).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() || !field.CanSet() {
			continue
		}
		value, ok := field.Interface().(attr.Value)
		if !ok || !value.IsUnknown() {
			continue
		}
		typ := value.Type(ctx)
		nullValue, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
		if err != nil {
			continue
		}
		field.Set(reflect.ValueOf(nullValue))
	}
}

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	v := value.ValueString()
	return &v
}

func optionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

func optionalInt32(value types.Int64) *int32 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := int32(value.ValueInt64())
	return &v
}

func stringMap(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	result := map[string]string{}
	if value.IsNull() || value.IsUnknown() {
		return result, nil
	}

	diags := value.ElementsAs(ctx, &result, false)
	return result, diags
}

func stringList(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	result := []string{}
	if value.IsNull() || value.IsUnknown() {
		return result, nil
	}

	diags := value.ElementsAs(ctx, &result, false)
	return result, diags
}

func setStringValue(ctx context.Context, values []string) types.Set {
	result, diags := types.SetValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.SetNull(types.StringType)
	}
	return result
}

func listStringValue(ctx context.Context, values []string) types.List {
	result, diags := types.ListValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return result
}

func stringMapValue(ctx context.Context, values map[string]string) types.Map {
	if values == nil {
		return types.MapNull(types.StringType)
	}
	result, diags := types.MapValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return result
}
