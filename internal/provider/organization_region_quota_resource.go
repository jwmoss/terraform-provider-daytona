// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationRegionQuotaResource{}
var _ resource.ResourceWithImportState = &OrganizationRegionQuotaResource{}

func NewOrganizationRegionQuotaResource() resource.Resource {
	return &OrganizationRegionQuotaResource{}
}

type OrganizationRegionQuotaResource struct {
	client *daytonaClient
}

type organizationRegionQuotaResourceModel struct {
	ID                            types.String  `tfsdk:"id"`
	OrganizationID                types.String  `tfsdk:"organization_id"`
	RegionID                      types.String  `tfsdk:"region_id"`
	SandboxClass                  types.String  `tfsdk:"sandbox_class"`
	TotalCPUQuota                 types.Float64 `tfsdk:"total_cpu_quota"`
	TotalMemoryQuota              types.Float64 `tfsdk:"total_memory_quota"`
	TotalDiskQuota                types.Float64 `tfsdk:"total_disk_quota"`
	TotalGPUQuota                 types.Float64 `tfsdk:"total_gpu_quota"`
	AllowedGPUTypes               types.List    `tfsdk:"allowed_gpu_types"`
	MaxCPUPerSandbox              types.Float64 `tfsdk:"max_cpu_per_sandbox"`
	MaxMemoryPerSandbox           types.Float64 `tfsdk:"max_memory_per_sandbox"`
	MaxDiskPerSandbox             types.Float64 `tfsdk:"max_disk_per_sandbox"`
	MaxDiskPerNonEphemeralSandbox types.Float64 `tfsdk:"max_disk_per_non_ephemeral_sandbox"`
	MaxCPUPerGPUSandbox           types.Float64 `tfsdk:"max_cpu_per_gpu_sandbox"`
	MaxMemoryPerGPUSandbox        types.Float64 `tfsdk:"max_memory_per_gpu_sandbox"`
	MaxDiskPerGPUSandbox          types.Float64 `tfsdk:"max_disk_per_gpu_sandbox"`
}

func (r *OrganizationRegionQuotaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_region_quota"
}

func (r *OrganizationRegionQuotaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an existing Daytona organization region quota. Daytona's organization API exposes update and usage readback for these quotas, but not create or delete; destroying this resource only removes Terraform state and does not delete or reset the quota.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Quota identifier in `organization_id:region_id:sandbox_class` format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona region ID where the quota applies.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sandbox_class": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("Sandbox class for this quota. Supported values are: %s.", strings.Join(sandboxClassValues(), ", ")),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"total_cpu_quota":    requiredFloat64Attribute("Total CPU quota."),
			"total_memory_quota": requiredFloat64Attribute("Total memory quota."),
			"total_disk_quota":   requiredFloat64Attribute("Total disk quota."),
			"total_gpu_quota":    requiredFloat64Attribute("Total GPU quota."),
			"allowed_gpu_types": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: fmt.Sprintf("Allowed GPU types for this quota. Supported values are: %s.", strings.Join(gpuTypeValues(), ", ")),
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"max_cpu_per_sandbox":                optionalComputedFloat64Attribute("Maximum CPU per sandbox."),
			"max_memory_per_sandbox":             optionalComputedFloat64Attribute("Maximum memory per sandbox."),
			"max_disk_per_sandbox":               optionalComputedFloat64Attribute("Maximum disk per sandbox."),
			"max_disk_per_non_ephemeral_sandbox": optionalComputedFloat64Attribute("Maximum disk per non-ephemeral sandbox."),
			"max_cpu_per_gpu_sandbox":            optionalComputedFloat64Attribute("Maximum CPU per GPU sandbox."),
			"max_memory_per_gpu_sandbox":         optionalComputedFloat64Attribute("Maximum memory per GPU sandbox."),
			"max_disk_per_gpu_sandbox":           optionalComputedFloat64Attribute("Maximum disk per GPU sandbox."),
		},
	}
}

func requiredFloat64Attribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Required:            true,
		MarkdownDescription: description,
	}
}

func (r *OrganizationRegionQuotaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*daytonaClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *OrganizationRegionQuotaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.applyOrganizationRegionQuota(ctx, data, &resp.Diagnostics) {
		return
	}

	updated, found, httpResp, err := r.readOrganizationRegionQuota(ctx, data)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization region quota", "read organization region quota", httpResp, err)
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"Daytona organization region quota not found",
			fmt.Sprintf("Daytona did not return a region quota for organization %q, region %q, and sandbox class %q after update.", data.OrganizationID.ValueString(), data.RegionID.ValueString(), data.SandboxClass.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *OrganizationRegionQuotaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, found, httpResp, err := r.readOrganizationRegionQuota(ctx, data)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization region quota", "read organization region quota", httpResp, err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *OrganizationRegionQuotaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.applyOrganizationRegionQuota(ctx, data, &resp.Diagnostics) {
		return
	}

	updated, found, httpResp, err := r.readOrganizationRegionQuota(ctx, data)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization region quota", "read organization region quota", httpResp, err)
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"Daytona organization region quota not found",
			fmt.Sprintf("Daytona did not return a region quota for organization %q, region %q, and sandbox class %q after update.", data.OrganizationID.ValueString(), data.RegionID.ValueString(), data.SandboxClass.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *OrganizationRegionQuotaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
}

func (r *OrganizationRegionQuotaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Daytona organization region quota import ID",
			"Use organization_id:region_id:sandbox_class, for example org-123:region-123:container.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sandbox_class"), parts[2])...)
}

func (r *OrganizationRegionQuotaResource) applyOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) bool {
	payload, ok := organizationRegionQuotaPayload(ctx, data, diags)
	if !ok {
		return false
	}

	httpResp, err := r.client.patchJSON(ctx, fmt.Sprintf(
		"/organizations/%s/quota/%s",
		url.PathEscape(data.OrganizationID.ValueString()),
		url.PathEscape(data.RegionID.ValueString()),
	), payload, nil)
	if err != nil {
		addAPIError(diags, "Unable to update Daytona organization region quota", "update organization region quota", httpResp, err)
		return false
	}

	return true
}

func organizationRegionQuotaPayload(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) (map[string]any, bool) {
	sandboxClass := apiclient.SandboxClass(data.SandboxClass.ValueString())
	if !sandboxClass.IsValid() || sandboxClass == apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
		diags.AddAttributeError(path.Root("sandbox_class"), "Invalid sandbox class", fmt.Sprintf("Unsupported sandbox class %q. Supported values are: %s.", data.SandboxClass.ValueString(), strings.Join(sandboxClassValues(), ", ")))
		return nil, false
	}

	payload := map[string]any{
		"sandboxClass":     data.SandboxClass.ValueString(),
		"totalCpuQuota":    data.TotalCPUQuota.ValueFloat64(),
		"totalMemoryQuota": data.TotalMemoryQuota.ValueFloat64(),
		"totalDiskQuota":   data.TotalDiskQuota.ValueFloat64(),
		"totalGpuQuota":    data.TotalGPUQuota.ValueFloat64(),
	}

	if !data.AllowedGPUTypes.IsNull() && !data.AllowedGPUTypes.IsUnknown() {
		values, listDiags := stringList(ctx, data.AllowedGPUTypes)
		diags.Append(listDiags...)
		if listDiags.HasError() {
			return nil, false
		}
		for _, value := range values {
			gpuType := apiclient.GpuType(value)
			if !gpuType.IsValid() || gpuType == apiclient.GPUTYPE_UNKNOWN_DEFAULT_OPEN_API {
				diags.AddAttributeError(path.Root("allowed_gpu_types"), "Invalid GPU type", fmt.Sprintf("Unsupported GPU type %q. Supported values are: %s.", value, strings.Join(gpuTypeValues(), ", ")))
				return nil, false
			}
		}
		payload["allowedGpuTypes"] = values
	}

	addOptionalFloat64(payload, "maxCpuPerSandbox", data.MaxCPUPerSandbox)
	addOptionalFloat64(payload, "maxMemoryPerSandbox", data.MaxMemoryPerSandbox)
	addOptionalFloat64(payload, "maxDiskPerSandbox", data.MaxDiskPerSandbox)
	addOptionalFloat64(payload, "maxDiskPerNonEphemeralSandbox", data.MaxDiskPerNonEphemeralSandbox)
	addOptionalFloat64(payload, "maxCpuPerGpuSandbox", data.MaxCPUPerGPUSandbox)
	addOptionalFloat64(payload, "maxMemoryPerGpuSandbox", data.MaxMemoryPerGPUSandbox)
	addOptionalFloat64(payload, "maxDiskPerGpuSandbox", data.MaxDiskPerGPUSandbox)

	return payload, true
}

func addOptionalFloat64(payload map[string]any, key string, value types.Float64) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	payload[key] = value.ValueFloat64()
}

func (r *OrganizationRegionQuotaResource) readOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel) (organizationRegionQuotaResourceModel, bool, *http.Response, error) {
	usage, httpResp, err := r.client.api.OrganizationsAPI.GetOrganizationUsageOverview(ctx, data.OrganizationID.ValueString()).Execute()
	if err != nil {
		return data, false, httpResp, err
	}
	if usage == nil {
		return data, false, httpResp, nil
	}

	for _, item := range usage.RegionUsage {
		if item.RegionId == data.RegionID.ValueString() && string(item.SandboxClass) == data.SandboxClass.ValueString() {
			return flattenOrganizationRegionQuota(ctx, &item, data), true, httpResp, nil
		}
	}

	return data, false, httpResp, nil
}

func flattenOrganizationRegionQuota(ctx context.Context, usage *apiclient.RegionUsageOverview, prior organizationRegionQuotaResourceModel) organizationRegionQuotaResourceModel {
	allowedGPUTypes := make([]string, 0, len(usage.AllowedGpuTypes))
	for _, gpuType := range usage.AllowedGpuTypes {
		allowedGPUTypes = append(allowedGPUTypes, string(gpuType))
	}

	prior.ID = types.StringValue(prior.OrganizationID.ValueString() + ":" + usage.RegionId + ":" + string(usage.SandboxClass))
	prior.RegionID = types.StringValue(usage.RegionId)
	prior.SandboxClass = types.StringValue(string(usage.SandboxClass))
	prior.TotalCPUQuota = float64Value(usage.TotalCpuQuota)
	prior.TotalMemoryQuota = float64Value(usage.TotalMemoryQuota)
	prior.TotalDiskQuota = float64Value(usage.TotalDiskQuota)
	prior.TotalGPUQuota = float64Value(usage.TotalGpuQuota)
	prior.AllowedGPUTypes = listStringValue(ctx, allowedGPUTypes)
	prior.MaxCPUPerSandbox = nullableFloat32Pointer(usage.MaxCpuPerSandbox.Get())
	prior.MaxMemoryPerSandbox = nullableFloat32Pointer(usage.MaxMemoryPerSandbox.Get())
	prior.MaxDiskPerSandbox = nullableFloat32Pointer(usage.MaxDiskPerSandbox.Get())
	prior.MaxDiskPerNonEphemeralSandbox = nullableFloat32Pointer(usage.MaxDiskPerNonEphemeralSandbox.Get())
	prior.MaxCPUPerGPUSandbox = nullableFloat32Pointer(usage.MaxCpuPerGpuSandbox.Get())
	prior.MaxMemoryPerGPUSandbox = nullableFloat32Pointer(usage.MaxMemoryPerGpuSandbox.Get())
	prior.MaxDiskPerGPUSandbox = nullableFloat32Pointer(usage.MaxDiskPerGpuSandbox.Get())

	return prior
}

func gpuTypeValues() []string {
	values := make([]string, 0, len(apiclient.AllowedGpuTypeEnumValues)-1)
	for _, value := range apiclient.AllowedGpuTypeEnumValues {
		if value != apiclient.GPUTYPE_UNKNOWN_DEFAULT_OPEN_API {
			values = append(values, string(value))
		}
	}
	return values
}
