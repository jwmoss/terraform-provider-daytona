package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationInvitationResource{}
var _ resource.ResourceWithImportState = &OrganizationInvitationResource{}

func NewOrganizationInvitationResource() resource.Resource {
	return &OrganizationInvitationResource{}
}

type OrganizationInvitationResource struct {
	client *daytonaClient
}

type organizationInvitationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	OrganizationName types.String `tfsdk:"organization_name"`
	Email            types.String `tfsdk:"email"`
	InvitedBy        types.String `tfsdk:"invited_by"`
	Role             types.String `tfsdk:"role"`
	AssignedRoleIDs  types.Set    `tfsdk:"assigned_role_ids"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *OrganizationInvitationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_invitation"
}

func (r *OrganizationInvitationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona organization invitation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization invitation ID.",
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
			"organization_name": computedStringAttribute("Daytona organization name."),
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Email address to invite.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"invited_by": computedStringAttribute("Email address of the inviter."),
			"role":       requiredStringAttribute("Organization member role for the invitee."),
			"assigned_role_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Custom organization role IDs assigned to the invitee. Use an empty set for no custom roles.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional RFC3339 expiration timestamp for the invitation.",
			},
			"status":     computedStringAttribute("Invitation status."),
			"created_at": computedStringAttribute("Invitation creation timestamp."),
			"updated_at": computedStringAttribute("Invitation update timestamp."),
		},
	}
}

func (r *OrganizationInvitationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationInvitationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationInvitationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assignedRoleIDs := []string{}
	resp.Diagnostics.Append(data.AssignedRoleIDs.ElementsAs(ctx, &assignedRoleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitation := apiclient.NewCreateOrganizationInvitation(data.Email.ValueString(), data.Role.ValueString(), assignedRoleIDs)
	if expiresAt := optionalString(data.ExpiresAt); expiresAt != nil {
		parsed, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("expires_at"), "Invalid expires_at timestamp", "expires_at must be formatted as RFC3339, for example 2026-12-31T23:59:59Z.")
			return
		}
		invitation.SetExpiresAt(parsed)
	}

	created, httpResp, err := r.client.api.OrganizationsAPI.CreateOrganizationInvitation(ctx, data.OrganizationID.ValueString()).
		CreateOrganizationInvitation(*invitation).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona organization invitation", "create organization invitation", httpResp, err)
		return
	}

	data = flattenOrganizationInvitation(ctx, created, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationInvitationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationInvitationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitation, httpResp, err := r.findInvitation(ctx, data.OrganizationID.ValueString(), data.ID.ValueString())
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization invitation", "read organization invitation", httpResp, err)
		return
	}
	if invitation == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data = flattenOrganizationInvitation(ctx, invitation, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationInvitationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationInvitationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assignedRoleIDs := []string{}
	resp.Diagnostics.Append(data.AssignedRoleIDs.ElementsAs(ctx, &assignedRoleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitation := apiclient.NewUpdateOrganizationInvitation(data.Role.ValueString(), assignedRoleIDs)
	if expiresAt := optionalString(data.ExpiresAt); expiresAt != nil {
		parsed, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("expires_at"), "Invalid expires_at timestamp", "expires_at must be formatted as RFC3339, for example 2026-12-31T23:59:59Z.")
			return
		}
		invitation.SetExpiresAt(parsed)
	}

	updated, httpResp, err := r.client.api.OrganizationsAPI.UpdateOrganizationInvitation(ctx, data.OrganizationID.ValueString(), data.ID.ValueString()).
		UpdateOrganizationInvitation(*invitation).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona organization invitation", "update organization invitation", httpResp, err)
		return
	}

	data = flattenOrganizationInvitation(ctx, updated, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationInvitationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationInvitationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.CancelOrganizationInvitation(ctx, data.OrganizationID.ValueString(), data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to cancel Daytona organization invitation", "cancel organization invitation", httpResp, err)
		return
	}
}

func (r *OrganizationInvitationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	organizationID, invitationID, err := parseCompositeImportID(req.ID, "organization_id", "invitation_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid organization invitation import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), types.StringValue(organizationID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(invitationID))...)
}

func (r *OrganizationInvitationResource) findInvitation(ctx context.Context, organizationID, invitationID string) (*apiclient.OrganizationInvitation, *http.Response, error) {
	invitations, httpResp, err := r.client.api.OrganizationsAPI.ListOrganizationInvitations(ctx, organizationID).Execute()
	if err != nil {
		return nil, httpResp, err
	}
	for i := range invitations {
		if invitations[i].Id == invitationID {
			return &invitations[i], httpResp, nil
		}
	}
	return nil, httpResp, nil
}

func flattenOrganizationInvitation(ctx context.Context, invitation *apiclient.OrganizationInvitation, prior organizationInvitationResourceModel) organizationInvitationResourceModel {
	if invitation == nil {
		return prior
	}

	prior.ID = types.StringValue(invitation.Id)
	prior.OrganizationID = types.StringValue(invitation.OrganizationId)
	prior.OrganizationName = types.StringValue(invitation.OrganizationName)
	prior.Email = types.StringValue(invitation.Email)
	prior.InvitedBy = types.StringValue(invitation.InvitedBy)
	prior.Role = types.StringValue(invitation.Role)
	prior.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(invitation.AssignedRoles))
	prior.ExpiresAt = terraformTimeString(invitation.ExpiresAt)
	prior.Status = types.StringValue(invitation.Status)
	prior.CreatedAt = terraformTimeString(invitation.CreatedAt)
	prior.UpdatedAt = terraformTimeString(invitation.UpdatedAt)

	return prior
}
