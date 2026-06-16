package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SnapshotResource{}
var _ resource.ResourceWithImportState = &SnapshotResource{}

func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

type SnapshotResource struct {
	client *daytonaClient
}

type snapshotResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ImageName      types.String `tfsdk:"image_name"`
	Entrypoint     types.List   `tfsdk:"entrypoint"`
	CPU            types.Int64  `tfsdk:"cpu"`
	GPU            types.Int64  `tfsdk:"gpu"`
	GPUType        types.String `tfsdk:"gpu_type"`
	GPUTypes       types.List   `tfsdk:"gpu_types"`
	Memory         types.Int64  `tfsdk:"memory"`
	Disk           types.Int64  `tfsdk:"disk"`
	BuildInfo      types.Object `tfsdk:"build_info"`
	RegionID       types.String `tfsdk:"region_id"`
	RegionIDs      types.List   `tfsdk:"region_ids"`
	SandboxClass   types.String `tfsdk:"sandbox_class"`
	State          types.String `tfsdk:"state"`
	Ref            types.String `tfsdk:"ref"`
	General        types.Bool   `tfsdk:"general"`
	Size           types.String `tfsdk:"size"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ErrorReason    types.String `tfsdk:"error_reason"`
}

func (r *SnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona snapshot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona snapshot ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Snapshot name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization ID that owns the snapshot.",
			},
			"image_name":    replaceStringAttribute("Image name used to build the snapshot."),
			"entrypoint":    optionalComputedReplaceStringListAttribute("Entrypoint command for the snapshot."),
			"cpu":           optionalComputedReplaceInt64Attribute("CPU cores allocated to sandboxes created from this snapshot. Defaults to a Daytona-assigned value when not set."),
			"gpu":           optionalComputedReplaceInt64Attribute("GPU units allocated to sandboxes created from this snapshot. Defaults to a Daytona-assigned value when not set."),
			"gpu_type":      computedStringAttribute("GPU type assigned to the snapshot, when assigned."),
			"gpu_types":     replaceStringListAttribute(fmt.Sprintf("Ordered preferred GPU types for sandboxes created from this snapshot. Supported values are: %s.", strings.Join(gpuTypeValues(), ", "))),
			"memory":        optionalComputedReplaceInt64Attribute("Memory allocated to sandboxes created from this snapshot in GB. Defaults to a Daytona-assigned value when not set."),
			"disk":          optionalComputedReplaceInt64Attribute("Disk allocated to sandboxes created from this snapshot in GB. Defaults to a Daytona-assigned value when not set."),
			"build_info":    buildInfoAttribute("Build information used to create a Dockerfile-backed snapshot."),
			"region_id":     replaceStringAttribute("Region ID where the snapshot will be available."),
			"sandbox_class": optionalComputedReplaceStringAttribute("Sandbox class for sandboxes created from this snapshot. Defaults to a Daytona-assigned class when not set."),
			"region_ids":    computedStringListAttribute("Region IDs where the snapshot is available."),
			"state":         computedStringAttribute("Current snapshot state."),
			"ref":           computedStringAttribute("Snapshot reference."),
			"general":       computedBoolAttribute("Whether this is a general Daytona snapshot."),
			"size":          computedStringAttribute("Snapshot size, when available."),
			"created_at":    computedStringAttribute("Snapshot creation timestamp."),
			"updated_at":    computedStringAttribute("Snapshot update timestamp."),
			"error_reason":  computedStringAttribute("Snapshot error reason, when available."),
		},
	}
}

func replaceStringListAttribute(description string) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.RequiresReplace(),
		},
	}
}

func optionalComputedReplaceStringListAttribute(description string) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
			listplanmodifier.RequiresReplace(),
		},
	}
}

func computedStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func computedBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func computedStringListAttribute(description string) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType:         types.StringType,
		Computed:            true,
		MarkdownDescription: description,
	}
}

func (r *SnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data snapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createSnapshot := apiclient.NewCreateSnapshot(data.Name.ValueString())
	if value := optionalString(data.ImageName); value != nil {
		createSnapshot.SetImageName(*value)
	}
	if value := optionalString(data.RegionID); value != nil {
		createSnapshot.SetRegionId(*value)
	}
	if value := optionalString(data.SandboxClass); value != nil {
		sandboxClass := apiclient.SandboxClass(*value)
		createSnapshot.SetSandboxClass(sandboxClass)
	}
	if value := optionalInt32(data.CPU); value != nil {
		createSnapshot.SetCpu(*value)
	}
	if value := optionalInt32(data.GPU); value != nil {
		createSnapshot.SetGpu(*value)
	}
	gpuTypes, diags := expandGPUTypes(ctx, data.GPUTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(gpuTypes) > 0 {
		createSnapshot.SetGpuType(gpuTypes)
	}
	if value := optionalInt32(data.Memory); value != nil {
		createSnapshot.SetMemory(*value)
	}
	if value := optionalInt32(data.Disk); value != nil {
		createSnapshot.SetDisk(*value)
	}
	buildInfo, diags := expandCreateBuildInfo(ctx, data.BuildInfo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if buildInfo != nil {
		createSnapshot.SetBuildInfo(*buildInfo)
	}

	entrypoint, diags := stringList(ctx, data.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(entrypoint) > 0 {
		createSnapshot.SetEntrypoint(entrypoint)
	}

	snapshot, httpResp, err := r.client.api.SnapshotsAPI.CreateSnapshot(ctx).
		CreateSnapshot(*createSnapshot).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona snapshot", "create snapshot", httpResp, err)
		return
	}

	data = flattenSnapshot(ctx, snapshot, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data snapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, httpResp, err := r.client.api.SnapshotsAPI.GetSnapshot(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona snapshot", "read snapshot", httpResp, err)
		return
	}

	data = flattenSnapshot(ctx, snapshot, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Daytona snapshot cannot be updated",
		"Daytona snapshot attributes are immutable through the API. Terraform should have planned replacement for configurable changes.",
	)
}

func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data snapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.SnapshotsAPI.RemoveSnapshot(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona snapshot", "delete snapshot", httpResp, err)
		return
	}
}

func (r *SnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenSnapshot(ctx context.Context, snapshot *apiclient.SnapshotDto, prior snapshotResourceModel) snapshotResourceModel {
	if snapshot == nil {
		return prior
	}

	prior.ID = types.StringValue(snapshot.Id)
	prior.Name = types.StringValue(snapshot.Name)
	prior.General = types.BoolValue(snapshot.General)
	prior.State = types.StringValue(string(snapshot.State))
	prior.CPU = types.Int64Value(int64(snapshot.Cpu))
	prior.GPU = types.Int64Value(int64(snapshot.Gpu))
	prior.Memory = types.Int64Value(int64(snapshot.Mem))
	prior.Disk = types.Int64Value(int64(snapshot.Disk))
	prior.CreatedAt = types.StringValue(snapshot.CreatedAt.Format(time.RFC3339))
	prior.UpdatedAt = types.StringValue(snapshot.UpdatedAt.Format(time.RFC3339))
	prior.Entrypoint = listStringValue(ctx, snapshot.Entrypoint)
	prior.RegionIDs = listStringValue(ctx, snapshot.RegionIds)
	prior.BuildInfo = flattenBuildInfo(ctx, snapshot.BuildInfo, prior.BuildInfo)

	if snapshot.GpuType != nil {
		prior.GPUType = types.StringValue(string(*snapshot.GpuType))
	} else {
		prior.GPUType = types.StringNull()
	}
	if prior.GPUTypes.IsUnknown() {
		prior.GPUTypes = types.ListNull(types.StringType)
	}

	if snapshot.OrganizationId != nil {
		prior.OrganizationID = types.StringValue(*snapshot.OrganizationId)
	} else {
		prior.OrganizationID = types.StringNull()
	}
	if snapshot.ImageName != nil && *snapshot.ImageName != "" {
		prior.ImageName = types.StringValue(*snapshot.ImageName)
	} else if prior.ImageName.IsUnknown() {
		prior.ImageName = types.StringNull()
	}
	if snapshot.Ref != nil {
		prior.Ref = types.StringValue(*snapshot.Ref)
	} else {
		prior.Ref = types.StringNull()
	}
	if snapshot.SandboxClass != nil {
		prior.SandboxClass = types.StringValue(*snapshot.SandboxClass)
	} else if prior.SandboxClass.IsUnknown() {
		prior.SandboxClass = types.StringNull()
	}
	if size, ok := snapshot.GetSizeOk(); ok && size != nil {
		prior.Size = types.StringValue(fmt.Sprintf("%g", *size))
	} else {
		prior.Size = types.StringNull()
	}
	if errorReason, ok := snapshot.GetErrorReasonOk(); ok && errorReason != nil {
		prior.ErrorReason = types.StringValue(*errorReason)
	} else {
		prior.ErrorReason = types.StringNull()
	}

	return prior
}
