package provider

import (
	"context"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CurrentUserOrganizationInvitationsDataSource{}

func NewCurrentUserOrganizationInvitationsDataSource() datasource.DataSource {
	return &CurrentUserOrganizationInvitationsDataSource{}
}

type CurrentUserOrganizationInvitationsDataSource struct {
	client *daytonaClient
}

type currentUserOrganizationInvitationsDataSourceModel struct {
	ID         types.String                         `tfsdk:"id"`
	TotalCount types.Int64                          `tfsdk:"total_count"`
	Items      []currentUserOrganizationInviteModel `tfsdk:"items"`
}

type currentUserOrganizationInviteModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	OrganizationName types.String `tfsdk:"organization_name"`
	Email            types.String `tfsdk:"email"`
	InvitedBy        types.String `tfsdk:"invited_by"`
	Role             types.String `tfsdk:"role"`
	AssignedRoleIDs  types.Set    `tfsdk:"assigned_role_ids"`
	Status           types.String `tfsdk:"status"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *CurrentUserOrganizationInvitationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user_organization_invitations"
}

func (d *CurrentUserOrganizationInvitationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists pending Daytona organization invitations for the authenticated user.",
		Attributes: map[string]schema.Attribute{
			"id":          computedDataSourceStringAttribute("Data source identifier."),
			"total_count": computedDataSourceInt64Attribute("Number of pending organization invitations for the authenticated user."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Pending organization invitations for the authenticated user.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                computedDataSourceStringAttribute("Invitation ID."),
						"organization_id":   computedDataSourceStringAttribute("Daytona organization ID."),
						"organization_name": computedDataSourceStringAttribute("Daytona organization name."),
						"email":             computedDataSourceStringAttribute("Invited email address."),
						"invited_by":        computedDataSourceStringAttribute("Email address of the inviter."),
						"role":              computedDataSourceStringAttribute("Organization member role for the invitee."),
						"assigned_role_ids": computedDataSourceStringSetAttribute("Custom organization role IDs assigned to the invitee."),
						"status":            computedDataSourceStringAttribute("Invitation status."),
						"expires_at":        computedDataSourceStringAttribute("Invitation expiration timestamp."),
						"created_at":        computedDataSourceStringAttribute("Invitation creation timestamp."),
						"updated_at":        computedDataSourceStringAttribute("Invitation update timestamp."),
					},
				},
			},
		},
	}
}

func (d *CurrentUserOrganizationInvitationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *CurrentUserOrganizationInvitationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data currentUserOrganizationInvitationsDataSourceModel

	count, httpResp, err := d.client.api.OrganizationsAPI.GetOrganizationInvitationsCountForAuthenticatedUser(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to count Daytona organization invitations", "count authenticated user organization invitations", httpResp, err)
		return
	}

	invitations, httpResp, err := d.client.api.OrganizationsAPI.ListOrganizationInvitationsForAuthenticatedUser(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona organization invitations", "list authenticated user organization invitations", httpResp, err)
		return
	}

	data.ID = types.StringValue("current_user_organization_invitations")
	data.TotalCount = types.Int64Value(int64(count))
	data.Items = flattenCurrentUserOrganizationInvitations(ctx, invitations)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenCurrentUserOrganizationInvitations(ctx context.Context, invitations []apiclient.OrganizationInvitation) []currentUserOrganizationInviteModel {
	items := make([]currentUserOrganizationInviteModel, 0, len(invitations))
	for _, invitation := range invitations {
		items = append(items, currentUserOrganizationInviteModel{
			ID:               types.StringValue(invitation.Id),
			OrganizationID:   types.StringValue(invitation.OrganizationId),
			OrganizationName: types.StringValue(invitation.OrganizationName),
			Email:            types.StringValue(invitation.Email),
			InvitedBy:        types.StringValue(invitation.InvitedBy),
			Role:             types.StringValue(invitation.Role),
			AssignedRoleIDs:  setStringValue(ctx, organizationRoleIDs(invitation.AssignedRoles)),
			Status:           types.StringValue(invitation.Status),
			ExpiresAt:        terraformTimeString(invitation.ExpiresAt),
			CreatedAt:        terraformTimeString(invitation.CreatedAt),
			UpdatedAt:        terraformTimeString(invitation.UpdatedAt),
		})
	}
	return items
}
