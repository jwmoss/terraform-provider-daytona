package provider

import (
	"context"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sandboxVolumeModel struct {
	VolumeID  types.String `tfsdk:"volume_id"`
	MountPath types.String `tfsdk:"mount_path"`
	Subpath   types.String `tfsdk:"subpath"`
}

func sandboxVolumesAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Persistent volumes mounted into the sandbox at create time.",
		PlanModifiers: []planmodifier.List{
			listplanmodifier.RequiresReplace(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"volume_id": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Daytona volume ID or name to mount.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"mount_path": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Absolute path where the volume is mounted in the sandbox.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"subpath": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Optional subpath within the volume to mount.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
			},
		},
	}
}

func sandboxVolumeAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"volume_id":  types.StringType,
		"mount_path": types.StringType,
		"subpath":    types.StringType,
	}
}

func expandSandboxVolumes(ctx context.Context, value types.List) ([]apiclient.SandboxVolume, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var data []sandboxVolumeModel
	diags.Append(value.ElementsAs(ctx, &data, false)...)
	if diags.HasError() {
		return nil, diags
	}

	volumes := make([]apiclient.SandboxVolume, 0, len(data))
	for _, volumeData := range data {
		volume := apiclient.NewSandboxVolume(volumeData.VolumeID.ValueString(), volumeData.MountPath.ValueString())
		if subpath := optionalString(volumeData.Subpath); subpath != nil {
			volume.SetSubpath(*subpath)
		}
		volumes = append(volumes, *volume)
	}

	return volumes, diags
}

func flattenSandboxVolumes(volumes []apiclient.SandboxVolume, prior types.List) types.List {
	objectType := types.ObjectType{AttrTypes: sandboxVolumeAttributeTypes()}
	if len(volumes) == 0 {
		if prior.IsUnknown() {
			return types.ListNull(objectType)
		}
		return prior
	}

	values := make([]attr.Value, 0, len(volumes))
	for _, volume := range volumes {
		objectValue, diags := types.ObjectValue(sandboxVolumeAttributeTypes(), map[string]attr.Value{
			"volume_id":  types.StringValue(volume.VolumeId),
			"mount_path": types.StringValue(volume.MountPath),
			"subpath":    pointerStringValue(volume.Subpath),
		})
		if diags.HasError() {
			return types.ListNull(objectType)
		}
		values = append(values, objectValue)
	}

	listValue, diags := types.ListValue(objectType, values)
	if diags.HasError() {
		return types.ListNull(objectType)
	}

	return listValue
}
