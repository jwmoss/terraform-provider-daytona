// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationMemberAccessResource{}
var _ resource.ResourceWithImportState = &OrganizationMemberAccessResource{}

func NewOrganizationMemberAccessResource() resource.Resource {
	return &OrganizationMemberAccessResource{}
}

type OrganizationMemberAccessResource struct {
	client *daytonaClient
}

type organizationMemberAccessResourceModel struct {
	ID              types.String `tfsdk:"id"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	UserID          types.String `tfsdk:"user_id"`
	Name            types.String `tfsdk:"name"`
	Email           types.String `tfsdk:"email"`
	Role            types.String `tfsdk:"role"`
	AssignedRoleIDs types.Set    `tfsdk:"assigned_role_ids"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *OrganizationMemberAccessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member_access"
}

func (r *OrganizationMemberAccessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages role access for an existing Daytona organization member. Destroying this resource removes the member from the organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization member user ID.",
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
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona user ID of the organization member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name":  computedStringAttribute("Member display name."),
			"email": computedStringAttribute("Member email address."),
			"role":  requiredStringAttribute("Organization member role."),
			"assigned_role_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Custom organization role IDs assigned to the member. Use an empty set for no custom roles.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": computedStringAttribute("Organization membership creation timestamp."),
			"updated_at": computedStringAttribute("Organization membership update timestamp."),
		},
	}
}

func (r *OrganizationMemberAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationMemberAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationMemberAccessResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member := r.updateAccess(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data = flattenOrganizationMemberAccess(ctx, member, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationMemberAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationMemberAccessResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, httpResp, err := r.findMember(ctx, data.OrganizationID.ValueString(), data.UserID.ValueString())
	if isNotFound(httpResp) || member == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization member access", "read organization member access", httpResp, err)
		return
	}

	data = flattenOrganizationMemberAccess(ctx, member, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationMemberAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationMemberAccessResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member := r.updateAccess(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data = flattenOrganizationMemberAccess(ctx, member, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationMemberAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationMemberAccessResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.DeleteOrganizationMember(ctx, data.OrganizationID.ValueString(), data.UserID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona organization member", "delete organization member", httpResp, err)
		return
	}
}

func (r *OrganizationMemberAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	organizationID, userID, err := parseCompositeImportID(req.ID, "organization_id", "user_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid organization member access import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), types.StringValue(organizationID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), types.StringValue(userID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(userID))...)
}

func (r *OrganizationMemberAccessResource) updateAccess(ctx context.Context, data organizationMemberAccessResourceModel, diags *diag.Diagnostics) *apiclient.OrganizationUser {
	assignedRoleIDs := []string{}
	diags.Append(data.AssignedRoleIDs.ElementsAs(ctx, &assignedRoleIDs, false)...)
	if diags.HasError() {
		return nil
	}

	member, httpResp, err := r.client.api.OrganizationsAPI.UpdateAccessForOrganizationMember(ctx, data.OrganizationID.ValueString(), data.UserID.ValueString()).
		UpdateOrganizationMemberAccess(*apiclient.NewUpdateOrganizationMemberAccess(data.Role.ValueString(), assignedRoleIDs)).
		Execute()
	if err != nil {
		addAPIError(diags, "Unable to update Daytona organization member access", "update organization member access", httpResp, err)
		return nil
	}

	return member
}

func (r *OrganizationMemberAccessResource) findMember(ctx context.Context, organizationID, userID string) (*apiclient.OrganizationUser, *http.Response, error) {
	members, httpResp, err := r.client.api.OrganizationsAPI.ListOrganizationMembers(ctx, organizationID).Execute()
	if err != nil {
		return nil, httpResp, err
	}
	for i := range members {
		if members[i].UserId == userID {
			return &members[i], httpResp, nil
		}
	}
	return nil, httpResp, nil
}

func flattenOrganizationMemberAccess(ctx context.Context, member *apiclient.OrganizationUser, prior organizationMemberAccessResourceModel) organizationMemberAccessResourceModel {
	if member == nil {
		return prior
	}

	prior.ID = types.StringValue(member.UserId)
	prior.UserID = types.StringValue(member.UserId)
	prior.OrganizationID = types.StringValue(member.OrganizationId)
	prior.Name = types.StringValue(member.Name)
	prior.Email = types.StringValue(member.Email)
	prior.Role = types.StringValue(member.Role)
	prior.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(member.AssignedRoles))
	prior.CreatedAt = terraformTimeString(member.CreatedAt)
	prior.UpdatedAt = terraformTimeString(member.UpdatedAt)

	return prior
}
