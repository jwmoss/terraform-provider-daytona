package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &APIKeyDataSource{}
var _ datasource.DataSource = &OrganizationRoleDataSource{}
var _ datasource.DataSource = &OrganizationMemberDataSource{}
var _ datasource.DataSource = &OrganizationInvitationDataSource{}

func NewAPIKeyDataSource() datasource.DataSource {
	return &APIKeyDataSource{}
}

func NewOrganizationRoleDataSource() datasource.DataSource {
	return &OrganizationRoleDataSource{}
}

func NewOrganizationMemberDataSource() datasource.DataSource {
	return &OrganizationMemberDataSource{}
}

func NewOrganizationInvitationDataSource() datasource.DataSource {
	return &OrganizationInvitationDataSource{}
}

type APIKeyDataSource struct {
	client *daytonaClient
}

func (d *APIKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *APIKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona API key by name.",
		Attributes: map[string]schema.Attribute{
			"id":          computedDataSourceStringAttribute("API key name, used as the object ID."),
			"name":        requiredDataSourceStringAttribute("API key name."),
			"permissions": computedDataSourceStringSetAttribute("Daytona organization resource permissions assigned to the API key."),
			"expires_at":  computedDataSourceStringAttribute("API key expiration timestamp, when available."),
			"value":       sensitiveComputedDataSourceStringAttribute("Masked API key value."),
			"created_at":  computedDataSourceStringAttribute("API key creation timestamp."),
			"last_used_at": computedDataSourceStringAttribute(
				"API key last-used timestamp, when available.",
			),
			"user_id": computedDataSourceStringAttribute("ID of the user who owns the API key."),
		},
	}
}

func (d *APIKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *APIKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data apiKeyResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, httpResp, err := d.client.api.ApiKeysAPI.GetApiKey(ctx, data.Name.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona API key", "read API key", httpResp, err)
		return
	}
	if apiKey == nil {
		resp.Diagnostics.AddError("Empty Daytona API key response", fmt.Sprintf("Daytona returned a successful response without API key %q.", data.Name.ValueString()))
		return
	}

	data = flattenAPIKeyList(apiKey, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationRoleDataSource struct {
	client *daytonaClient
}

func (d *OrganizationRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (d *OrganizationRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona organization role by organization ID and role ID.",
		Attributes: map[string]schema.Attribute{
			"id":              requiredDataSourceStringAttribute("Daytona organization role ID."),
			"organization_id": requiredDataSourceStringAttribute("Daytona organization ID."),
			"name":            computedDataSourceStringAttribute("Role name."),
			"description":     computedDataSourceStringAttribute("Role description."),
			"permissions":     computedDataSourceStringSetAttribute("Permissions assigned to the role."),
			"is_global":       computedDataSourceBoolAttribute("Whether this is a global Daytona role."),
			"created_at":      computedDataSourceStringAttribute("Role creation timestamp."),
			"updated_at":      computedDataSourceStringAttribute("Role update timestamp."),
		},
	}
}

func (d *OrganizationRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationRoleResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, httpResp, err := findOrganizationRole(ctx, d.client, data.OrganizationID.ValueString(), data.ID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization role", "read organization role", httpResp, err)
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Daytona organization role not found", fmt.Sprintf("No organization role %q was found in organization %q.", data.ID.ValueString(), data.OrganizationID.ValueString()))
		return
	}

	data = flattenOrganizationRole(ctx, role, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationMemberDataSource struct {
	client *daytonaClient
}

func (d *OrganizationMemberDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (d *OrganizationMemberDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona organization member by organization ID and user ID.",
		Attributes: map[string]schema.Attribute{
			"id":                computedDataSourceStringAttribute("Daytona organization member user ID."),
			"organization_id":   requiredDataSourceStringAttribute("Daytona organization ID."),
			"user_id":           requiredDataSourceStringAttribute("Daytona user ID of the organization member."),
			"name":              computedDataSourceStringAttribute("Member display name."),
			"email":             computedDataSourceStringAttribute("Member email address."),
			"role":              computedDataSourceStringAttribute("Organization member role."),
			"assigned_role_ids": computedDataSourceStringSetAttribute("Custom organization role IDs assigned to the member."),
			"created_at":        computedDataSourceStringAttribute("Organization membership creation timestamp."),
			"updated_at":        computedDataSourceStringAttribute("Organization membership update timestamp."),
		},
	}
}

func (d *OrganizationMemberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationMemberAccessResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, httpResp, err := findOrganizationMember(ctx, d.client, data.OrganizationID.ValueString(), data.UserID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization member", "read organization member", httpResp, err)
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Daytona organization member not found", fmt.Sprintf("No organization member %q was found in organization %q.", data.UserID.ValueString(), data.OrganizationID.ValueString()))
		return
	}

	data = flattenOrganizationMemberAccess(ctx, member, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationInvitationDataSource struct {
	client *daytonaClient
}

func (d *OrganizationInvitationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_invitation"
}

func (d *OrganizationInvitationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona organization invitation by organization ID and invitation ID.",
		Attributes: map[string]schema.Attribute{
			"id":                requiredDataSourceStringAttribute("Daytona organization invitation ID."),
			"organization_id":   requiredDataSourceStringAttribute("Daytona organization ID."),
			"organization_name": computedDataSourceStringAttribute("Daytona organization name."),
			"email":             computedDataSourceStringAttribute("Invited email address."),
			"invited_by":        computedDataSourceStringAttribute("Email address of the inviter."),
			"role":              computedDataSourceStringAttribute("Organization member role for the invitee."),
			"assigned_role_ids": computedDataSourceStringSetAttribute("Custom organization role IDs assigned to the invitee."),
			"expires_at":        computedDataSourceStringAttribute("Invitation expiration timestamp."),
			"status":            computedDataSourceStringAttribute("Invitation status."),
			"created_at":        computedDataSourceStringAttribute("Invitation creation timestamp."),
			"updated_at":        computedDataSourceStringAttribute("Invitation update timestamp."),
		},
	}
}

func (d *OrganizationInvitationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationInvitationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationInvitationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitation, httpResp, err := findOrganizationInvitation(ctx, d.client, data.OrganizationID.ValueString(), data.ID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization invitation", "read organization invitation", httpResp, err)
		return
	}
	if invitation == nil {
		resp.Diagnostics.AddError("Daytona organization invitation not found", fmt.Sprintf("No organization invitation %q was found in organization %q.", data.ID.ValueString(), data.OrganizationID.ValueString()))
		return
	}

	data = flattenOrganizationInvitation(ctx, invitation, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findOrganizationRole(ctx context.Context, client *daytonaClient, organizationID, roleID string) (*apiclient.OrganizationRole, *http.Response, error) {
	roles, httpResp, err := client.api.OrganizationsAPI.ListOrganizationRoles(ctx, organizationID).Execute()
	if err != nil {
		return nil, httpResp, err
	}
	for i := range roles {
		if roles[i].Id == roleID {
			return &roles[i], httpResp, nil
		}
	}
	return nil, httpResp, nil
}

func findOrganizationMember(ctx context.Context, client *daytonaClient, organizationID, userID string) (*apiclient.OrganizationUser, *http.Response, error) {
	members, httpResp, err := client.api.OrganizationsAPI.ListOrganizationMembers(ctx, organizationID).Execute()
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

func findOrganizationInvitation(ctx context.Context, client *daytonaClient, organizationID, invitationID string) (*apiclient.OrganizationInvitation, *http.Response, error) {
	invitations, httpResp, err := client.api.OrganizationsAPI.ListOrganizationInvitations(ctx, organizationID).Execute()
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
