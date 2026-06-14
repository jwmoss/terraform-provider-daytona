package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationRoleResource{}
var _ resource.ResourceWithImportState = &OrganizationRoleResource{}

func NewOrganizationRoleResource() resource.Resource {
	return &OrganizationRoleResource{}
}

type OrganizationRoleResource struct {
	client *daytonaClient
}

type organizationRoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Permissions    types.Set    `tfsdk:"permissions"`
	IsGlobal       types.Bool   `tfsdk:"is_global"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *OrganizationRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *OrganizationRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom role in a Daytona organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization role ID.",
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
			"name":        requiredStringAttribute("Role name."),
			"description": requiredStringAttribute("Role description."),
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Permissions assigned to the role.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"is_global":  computedBoolAttribute("Whether this is a global Daytona role."),
			"created_at": computedStringAttribute("Role creation timestamp."),
			"updated_at": computedStringAttribute("Role update timestamp."),
		},
	}
}

func (r *OrganizationRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := []string{}
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, httpResp, err := r.client.api.OrganizationsAPI.CreateOrganizationRole(ctx, data.OrganizationID.ValueString()).
		CreateOrganizationRole(*apiclient.NewCreateOrganizationRole(data.Name.ValueString(), data.Description.ValueString(), permissions)).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona organization role", "create organization role", httpResp, err)
		return
	}

	data = flattenOrganizationRole(ctx, role, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, httpResp, err := r.findRole(ctx, data.OrganizationID.ValueString(), data.ID.ValueString())
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization role", "read organization role", httpResp, err)
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data = flattenOrganizationRole(ctx, role, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := []string{}
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, httpResp, err := r.client.api.OrganizationsAPI.UpdateOrganizationRole(ctx, data.OrganizationID.ValueString(), data.ID.ValueString()).
		UpdateOrganizationRole(*apiclient.NewUpdateOrganizationRole(data.Name.ValueString(), data.Description.ValueString(), permissions)).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona organization role", "update organization role", httpResp, err)
		return
	}

	data = flattenOrganizationRole(ctx, role, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.DeleteOrganizationRole(ctx, data.OrganizationID.ValueString(), data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona organization role", "delete organization role", httpResp, err)
		return
	}
}

func (r *OrganizationRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	organizationID, roleID, err := parseCompositeImportID(req.ID, "organization_id", "role_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid organization role import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), types.StringValue(organizationID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(roleID))...)
}

func (r *OrganizationRoleResource) findRole(ctx context.Context, organizationID, roleID string) (*apiclient.OrganizationRole, *http.Response, error) {
	roles, httpResp, err := r.client.api.OrganizationsAPI.ListOrganizationRoles(ctx, organizationID).Execute()
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

func flattenOrganizationRole(ctx context.Context, role *apiclient.OrganizationRole, prior organizationRoleResourceModel) organizationRoleResourceModel {
	if role == nil {
		return prior
	}

	prior.ID = types.StringValue(role.Id)
	prior.Name = types.StringValue(role.Name)
	prior.Description = types.StringValue(role.Description)
	prior.Permissions = setStringValue(ctx, role.Permissions)
	prior.IsGlobal = types.BoolValue(role.IsGlobal)
	prior.CreatedAt = terraformTimeString(role.CreatedAt)
	prior.UpdatedAt = terraformTimeString(role.UpdatedAt)

	return prior
}
