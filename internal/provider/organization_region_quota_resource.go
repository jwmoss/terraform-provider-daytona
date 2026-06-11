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
var _ resource.Resource = &AdminOrganizationRegionQuotaResource{}
var _ resource.ResourceWithImportState = &AdminOrganizationRegionQuotaResource{}

func NewOrganizationRegionQuotaResource() resource.Resource {
	return &OrganizationRegionQuotaResource{}
}

func NewAdminOrganizationRegionQuotaResource() resource.Resource {
	return &AdminOrganizationRegionQuotaResource{}
}

type OrganizationRegionQuotaResource struct {
	client *daytonaClient
}

type AdminOrganizationRegionQuotaResource struct {
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

func (r *AdminOrganizationRegionQuotaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_organization_region_quota"
}

func (r *AdminOrganizationRegionQuotaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Daytona organization region quotas through Daytona admin APIs. This resource requires credentials with Daytona admin privileges.",
		Attributes:          organizationRegionQuotaResourceAttributes("Quota identifier in `organization_id:region_id:sandbox_class` format."),
	}
}

func (r *OrganizationRegionQuotaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_region_quota"
}

func (r *OrganizationRegionQuotaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an existing Daytona organization region quota. Daytona's organization API exposes update and usage readback for these quotas, but not create or delete; destroying this resource only removes Terraform state and does not delete or reset the quota.",
		Attributes:          organizationRegionQuotaResourceAttributes("Quota identifier in `organization_id:region_id:sandbox_class` format."),
	}
}

func organizationRegionQuotaResourceAttributes(idDescription string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: idDescription,
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
	}
}

func requiredFloat64Attribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Required:            true,
		MarkdownDescription: description,
	}
}

func (r *OrganizationRegionQuotaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (r *AdminOrganizationRegionQuotaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func configureResourceDaytonaClient(providerData any, diags *diag.Diagnostics) *daytonaClient {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*daytonaClient)
	if !ok {
		diags.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
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
	importOrganizationRegionQuotaState(ctx, req.ID, &resp.Diagnostics, resp.State.SetAttribute)
}

func (r *AdminOrganizationRegionQuotaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, ok := r.createAdminOrganizationRegionQuota(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &created)...)
}

func (r *AdminOrganizationRegionQuotaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, found, httpResp, err := r.readAdminOrganizationRegionQuota(ctx, data)
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin organization region quota", "read admin organization region quota", httpResp, err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *AdminOrganizationRegionQuotaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.updateAdminOrganizationRegionQuota(ctx, data, &resp.Diagnostics) {
		return
	}

	updated, found, httpResp, err := r.readAdminOrganizationRegionQuota(ctx, data)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin organization region quota", "read admin organization region quota", httpResp, err)
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"Daytona admin organization region quota not found",
			fmt.Sprintf("Daytona did not return a region quota for organization %q, region %q, and sandbox class %q after update.", data.OrganizationID.ValueString(), data.RegionID.ValueString(), data.SandboxClass.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *AdminOrganizationRegionQuotaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationRegionQuotaResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.deleteAdminOrganizationRegionQuota(ctx, data, &resp.Diagnostics) {
		return
	}
}

func (r *AdminOrganizationRegionQuotaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importOrganizationRegionQuotaState(ctx, req.ID, &resp.Diagnostics, resp.State.SetAttribute)
}

func importOrganizationRegionQuotaState(ctx context.Context, id string, diags *diag.Diagnostics, setAttribute func(context.Context, path.Path, any) diag.Diagnostics) {
	parts := strings.Split(id, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		diags.AddError(
			"Invalid Daytona organization region quota import ID",
			"Use organization_id:region_id:sandbox_class, for example org-123:region-123:container.",
		)
		return
	}

	diags.Append(setAttribute(ctx, path.Root("id"), id)...)
	diags.Append(setAttribute(ctx, path.Root("organization_id"), parts[0])...)
	diags.Append(setAttribute(ctx, path.Root("region_id"), parts[1])...)
	diags.Append(setAttribute(ctx, path.Root("sandbox_class"), parts[2])...)
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
	_, gpuTypes, ok := organizationRegionQuotaEnums(ctx, data, diags)
	if !ok {
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
		values := make([]string, 0, len(gpuTypes))
		for _, gpuType := range gpuTypes {
			values = append(values, string(gpuType))
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

func organizationRegionQuotaEnums(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) (apiclient.SandboxClass, []apiclient.GpuType, bool) {
	sandboxClass := apiclient.SandboxClass(data.SandboxClass.ValueString())
	if !sandboxClass.IsValid() || sandboxClass == apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
		diags.AddAttributeError(path.Root("sandbox_class"), "Invalid sandbox class", fmt.Sprintf("Unsupported sandbox class %q. Supported values are: %s.", data.SandboxClass.ValueString(), strings.Join(sandboxClassValues(), ", ")))
		return "", nil, false
	}

	gpuTypes := []apiclient.GpuType{}
	if !data.AllowedGPUTypes.IsNull() && !data.AllowedGPUTypes.IsUnknown() {
		values, listDiags := stringList(ctx, data.AllowedGPUTypes)
		diags.Append(listDiags...)
		if listDiags.HasError() {
			return "", nil, false
		}
		for _, value := range values {
			gpuType := apiclient.GpuType(value)
			if !gpuType.IsValid() || gpuType == apiclient.GPUTYPE_UNKNOWN_DEFAULT_OPEN_API {
				diags.AddAttributeError(path.Root("allowed_gpu_types"), "Invalid GPU type", fmt.Sprintf("Unsupported GPU type %q. Supported values are: %s.", value, strings.Join(gpuTypeValues(), ", ")))
				return "", nil, false
			}
			gpuTypes = append(gpuTypes, gpuType)
		}
	}

	return sandboxClass, gpuTypes, true
}

func createOrganizationRegionQuotaFromData(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) (apiclient.CreateOrganizationRegionQuota, bool) {
	sandboxClass, gpuTypes, ok := organizationRegionQuotaEnums(ctx, data, diags)
	if !ok {
		return apiclient.CreateOrganizationRegionQuota{}, false
	}

	payload := *apiclient.NewCreateOrganizationRegionQuota(
		sandboxClass,
		float32(data.TotalCPUQuota.ValueFloat64()),
		float32(data.TotalMemoryQuota.ValueFloat64()),
		float32(data.TotalDiskQuota.ValueFloat64()),
		float32(data.TotalGPUQuota.ValueFloat64()),
	)
	if !data.AllowedGPUTypes.IsNull() && !data.AllowedGPUTypes.IsUnknown() {
		payload.AllowedGpuTypes = gpuTypes
	}
	payload.MaxCpuPerSandbox = nullableFloat32FromFloat64(data.MaxCPUPerSandbox)
	payload.MaxMemoryPerSandbox = nullableFloat32FromFloat64(data.MaxMemoryPerSandbox)
	payload.MaxDiskPerSandbox = nullableFloat32FromFloat64(data.MaxDiskPerSandbox)
	payload.MaxDiskPerNonEphemeralSandbox = nullableFloat32FromFloat64(data.MaxDiskPerNonEphemeralSandbox)
	payload.MaxCpuPerGpuSandbox = nullableFloat32FromFloat64(data.MaxCPUPerGPUSandbox)
	payload.MaxMemoryPerGpuSandbox = nullableFloat32FromFloat64(data.MaxMemoryPerGPUSandbox)
	payload.MaxDiskPerGpuSandbox = nullableFloat32FromFloat64(data.MaxDiskPerGPUSandbox)

	return payload, true
}

func updateOrganizationRegionQuotaFromData(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) (apiclient.UpdateOrganizationRegionQuota, bool) {
	sandboxClass, gpuTypes, ok := organizationRegionQuotaEnums(ctx, data, diags)
	if !ok {
		return apiclient.UpdateOrganizationRegionQuota{}, false
	}

	payload := apiclient.UpdateOrganizationRegionQuota{
		SandboxClass:                  &sandboxClass,
		TotalCpuQuota:                 nullableFloat32FromFloat64(data.TotalCPUQuota),
		TotalMemoryQuota:              nullableFloat32FromFloat64(data.TotalMemoryQuota),
		TotalDiskQuota:                nullableFloat32FromFloat64(data.TotalDiskQuota),
		TotalGpuQuota:                 nullableFloat32FromFloat64(data.TotalGPUQuota),
		MaxCpuPerSandbox:              nullableFloat32FromFloat64(data.MaxCPUPerSandbox),
		MaxMemoryPerSandbox:           nullableFloat32FromFloat64(data.MaxMemoryPerSandbox),
		MaxDiskPerSandbox:             nullableFloat32FromFloat64(data.MaxDiskPerSandbox),
		MaxDiskPerNonEphemeralSandbox: nullableFloat32FromFloat64(data.MaxDiskPerNonEphemeralSandbox),
		MaxCpuPerGpuSandbox:           nullableFloat32FromFloat64(data.MaxCPUPerGPUSandbox),
		MaxMemoryPerGpuSandbox:        nullableFloat32FromFloat64(data.MaxMemoryPerGPUSandbox),
		MaxDiskPerGpuSandbox:          nullableFloat32FromFloat64(data.MaxDiskPerGPUSandbox),
	}
	if !data.AllowedGPUTypes.IsNull() && !data.AllowedGPUTypes.IsUnknown() {
		payload.AllowedGpuTypes = gpuTypes
	}

	return payload, true
}

func (r *AdminOrganizationRegionQuotaResource) createAdminOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) (organizationRegionQuotaResourceModel, bool) {
	payload, ok := createOrganizationRegionQuotaFromData(ctx, data, diags)
	if !ok {
		return data, false
	}

	quota, httpResp, err := r.client.api.AdminAPI.AdminCreateOrganizationRegionQuota(ctx, data.OrganizationID.ValueString(), data.RegionID.ValueString()).
		CreateOrganizationRegionQuota(payload).
		Execute()
	if err != nil {
		addAPIError(diags, "Unable to create Daytona admin organization region quota", "create admin organization region quota", httpResp, err)
		return data, false
	}
	if quota == nil {
		diags.AddError("Empty Daytona admin organization region quota response", "Daytona returned a successful response without organization region quota data.")
		return data, false
	}

	return flattenAdminOrganizationRegionQuota(ctx, quota, data), true
}

func (r *AdminOrganizationRegionQuotaResource) updateAdminOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) bool {
	payload, ok := updateOrganizationRegionQuotaFromData(ctx, data, diags)
	if !ok {
		return false
	}

	httpResp, err := r.client.api.AdminAPI.AdminUpdateOrganizationRegionQuota(ctx, data.OrganizationID.ValueString(), data.RegionID.ValueString()).
		UpdateOrganizationRegionQuota(payload).
		Execute()
	if err != nil {
		addAPIError(diags, "Unable to update Daytona admin organization region quota", "update admin organization region quota", httpResp, err)
		return false
	}

	return true
}

func (r *AdminOrganizationRegionQuotaResource) readAdminOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel) (organizationRegionQuotaResourceModel, bool, *http.Response, error) {
	sandboxClass := apiclient.SandboxClass(data.SandboxClass.ValueString())
	if !sandboxClass.IsValid() || sandboxClass == apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
		return data, false, nil, fmt.Errorf("unsupported sandbox class %q", data.SandboxClass.ValueString())
	}

	quota, httpResp, err := r.client.api.AdminAPI.AdminGetOrganizationRegionQuota(ctx, data.OrganizationID.ValueString(), data.RegionID.ValueString(), sandboxClass).Execute()
	if err != nil {
		return data, false, httpResp, err
	}
	if quota == nil {
		return data, false, httpResp, nil
	}

	return flattenAdminOrganizationRegionQuota(ctx, quota, data), true, httpResp, nil
}

func (r *AdminOrganizationRegionQuotaResource) deleteAdminOrganizationRegionQuota(ctx context.Context, data organizationRegionQuotaResourceModel, diags *diag.Diagnostics) bool {
	sandboxClass := apiclient.SandboxClass(data.SandboxClass.ValueString())
	if !sandboxClass.IsValid() || sandboxClass == apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
		diags.AddAttributeError(path.Root("sandbox_class"), "Invalid sandbox class", fmt.Sprintf("Unsupported sandbox class %q. Supported values are: %s.", data.SandboxClass.ValueString(), strings.Join(sandboxClassValues(), ", ")))
		return false
	}

	httpResp, err := r.client.api.AdminAPI.AdminDeleteOrganizationRegionQuota(ctx, data.OrganizationID.ValueString(), data.RegionID.ValueString(), sandboxClass).Execute()
	if isNotFound(httpResp) {
		return true
	}
	if err != nil {
		addAPIError(diags, "Unable to delete Daytona admin organization region quota", "delete admin organization region quota", httpResp, err)
		return false
	}

	return true
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

func flattenAdminOrganizationRegionQuota(ctx context.Context, quota *apiclient.RegionQuota, prior organizationRegionQuotaResourceModel) organizationRegionQuotaResourceModel {
	allowedGPUTypes := make([]string, 0, len(quota.AllowedGpuTypes))
	for _, gpuType := range quota.AllowedGpuTypes {
		allowedGPUTypes = append(allowedGPUTypes, string(gpuType))
	}

	organizationID := quota.OrganizationId
	if organizationID == "" {
		organizationID = prior.OrganizationID.ValueString()
	}
	regionID := quota.RegionId
	if regionID == "" {
		regionID = prior.RegionID.ValueString()
	}
	sandboxClass := quota.SandboxClass
	if sandboxClass == "" {
		sandboxClass = apiclient.SandboxClass(prior.SandboxClass.ValueString())
	}

	prior.ID = types.StringValue(organizationID + ":" + regionID + ":" + string(sandboxClass))
	prior.OrganizationID = types.StringValue(organizationID)
	prior.RegionID = types.StringValue(regionID)
	prior.SandboxClass = types.StringValue(string(sandboxClass))
	prior.TotalCPUQuota = float64Value(quota.TotalCpuQuota)
	prior.TotalMemoryQuota = float64Value(quota.TotalMemoryQuota)
	prior.TotalDiskQuota = float64Value(quota.TotalDiskQuota)
	prior.TotalGPUQuota = float64Value(quota.TotalGpuQuota)
	prior.AllowedGPUTypes = listStringValue(ctx, allowedGPUTypes)
	prior.MaxCPUPerSandbox = nullableFloat32Pointer(quota.MaxCpuPerSandbox.Get())
	prior.MaxMemoryPerSandbox = nullableFloat32Pointer(quota.MaxMemoryPerSandbox.Get())
	prior.MaxDiskPerSandbox = nullableFloat32Pointer(quota.MaxDiskPerSandbox.Get())
	prior.MaxDiskPerNonEphemeralSandbox = nullableFloat32Pointer(quota.MaxDiskPerNonEphemeralSandbox.Get())
	prior.MaxCPUPerGPUSandbox = nullableFloat32Pointer(quota.MaxCpuPerGpuSandbox.Get())
	prior.MaxMemoryPerGPUSandbox = nullableFloat32Pointer(quota.MaxMemoryPerGpuSandbox.Get())
	prior.MaxDiskPerGPUSandbox = nullableFloat32Pointer(quota.MaxDiskPerGpuSandbox.Get())

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
