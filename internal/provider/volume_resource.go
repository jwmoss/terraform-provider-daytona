// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VolumeResource{}
var _ resource.ResourceWithImportState = &VolumeResource{}

func NewVolumeResource() resource.Resource {
	return &VolumeResource{}
}

type VolumeResource struct {
	client *daytonaClient
}

type volumeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	State          types.String `tfsdk:"state"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	LastUsedAt     types.String `tfsdk:"last_used_at"`
	ErrorReason    types.String `tfsdk:"error_reason"`
}

func (r *VolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *VolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona persistent volume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona volume ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Volume name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization ID that owns the volume.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current volume state.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume update timestamp.",
			},
			"last_used_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume last-used timestamp, when available.",
			},
			"error_reason": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume error reason, when available.",
			},
		},
	}
}

func (r *VolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data volumeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, httpResp, err := r.client.api.VolumesAPI.CreateVolume(ctx).
		CreateVolume(*apiclient.NewCreateVolume(data.Name.ValueString())).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona volume", "create volume", httpResp, err)
		return
	}

	data = flattenVolume(volume, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data volumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, httpResp, err := r.client.api.VolumesAPI.GetVolume(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona volume", "read volume", httpResp, err)
		return
	}

	data = flattenVolume(volume, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Daytona volume cannot be updated",
		"The Daytona API does not expose mutable volume fields. Terraform should have planned replacement for any configurable change.",
	)
}

func (r *VolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data volumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.VolumesAPI.DeleteVolume(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona volume", "delete volume", httpResp, err)
		return
	}
}

func (r *VolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenVolume(volume *apiclient.VolumeDto, prior volumeResourceModel) volumeResourceModel {
	if volume == nil {
		return prior
	}

	prior.ID = types.StringValue(volume.Id)
	prior.Name = types.StringValue(volume.Name)
	prior.OrganizationID = types.StringValue(volume.OrganizationId)
	prior.State = types.StringValue(string(volume.State))
	prior.CreatedAt = types.StringValue(volume.CreatedAt)
	prior.UpdatedAt = types.StringValue(volume.UpdatedAt)

	if value, ok := volume.GetLastUsedAtOk(); ok && value != nil {
		prior.LastUsedAt = types.StringValue(*value)
	} else {
		prior.LastUsedAt = types.StringNull()
	}

	if value, ok := volume.GetErrorReasonOk(); ok && value != nil {
		prior.ErrorReason = types.StringValue(*value)
	} else {
		prior.ErrorReason = types.StringNull()
	}

	return prior
}
