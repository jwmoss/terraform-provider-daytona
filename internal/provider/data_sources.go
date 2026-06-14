package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CurrentAPIKeyDataSource{}
var _ datasource.DataSource = &collectionDataSource{}

func NewCurrentAPIKeyDataSource() datasource.DataSource {
	return &CurrentAPIKeyDataSource{}
}

func NewAPIKeysDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "api_keys"}
}

func NewVolumesDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "volumes"}
}

func NewRegionsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "regions"}
}

func NewSharedRegionsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "shared_regions"}
}

func NewRunnersDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "runners"}
}

func NewSandboxesDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "sandboxes"}
}

func NewSnapshotsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "snapshots"}
}

func NewDockerRegistriesDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "docker_registries"}
}

func NewOrganizationsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "organizations"}
}

func NewOrganizationRolesDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "organization_roles"}
}

func NewOrganizationMembersDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "organization_members"}
}

func NewOrganizationInvitationsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "organization_invitations"}
}

type CurrentAPIKeyDataSource struct {
	client *daytonaClient
}

type currentAPIKeyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Permissions types.Set    `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	UserID      types.String `tfsdk:"user_id"`
}

func (d *CurrentAPIKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_api_key"
}

func (d *CurrentAPIKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads metadata for the Daytona API key currently used by the provider.",
		Attributes: map[string]schema.Attribute{
			"id":          computedDataSourceStringAttribute("API key name."),
			"name":        computedDataSourceStringAttribute("API key name."),
			"value":       sensitiveComputedDataSourceStringAttribute("Masked API key value."),
			"permissions": computedDataSourceStringSetAttribute("Permissions assigned to the API key."),
			"created_at":  computedDataSourceStringAttribute("API key creation timestamp."),
			"last_used_at": computedDataSourceStringAttribute(
				"API key last-used timestamp, when available.",
			),
			"expires_at": computedDataSourceStringAttribute("API key expiration timestamp, when available."),
			"user_id":    computedDataSourceStringAttribute("ID of the user who owns the API key."),
		},
	}
}

func (d *CurrentAPIKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *CurrentAPIKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiKey, httpResp, err := d.client.api.ApiKeysAPI.GetCurrentApiKey(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read current Daytona API key", "read current API key", httpResp, err)
		return
	}

	data := currentAPIKeyDataSourceModel{
		ID:          types.StringValue(apiKey.Name),
		Name:        types.StringValue(apiKey.Name),
		Value:       types.StringValue(apiKey.Value),
		Permissions: setStringValue(ctx, apiKey.Permissions),
		CreatedAt:   types.StringValue(apiKey.CreatedAt.Format(time.RFC3339)),
		UserID:      types.StringValue(apiKey.UserId),
	}

	if lastUsedAt, ok := apiKey.GetLastUsedAtOk(); ok && lastUsedAt != nil {
		data.LastUsedAt = types.StringValue(lastUsedAt.Format(time.RFC3339))
	} else {
		data.LastUsedAt = types.StringNull()
	}
	if expiresAt, ok := apiKey.GetExpiresAtOk(); ok && expiresAt != nil {
		data.ExpiresAt = types.StringValue(expiresAt.Format(time.RFC3339))
	} else {
		data.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type collectionDataSource struct {
	client *daytonaClient
	kind   string
}

type collectionDataSourceModel struct {
	ID             types.String          `tfsdk:"id"`
	OrganizationID types.String          `tfsdk:"organization_id"`
	Items          []collectionItemModel `tfsdk:"items"`
}

type collectionItemModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Value            types.String `tfsdk:"value"`
	Description      types.String `tfsdk:"description"`
	Email            types.String `tfsdk:"email"`
	UserID           types.String `tfsdk:"user_id"`
	CreatedBy        types.String `tfsdk:"created_by"`
	InvitedBy        types.String `tfsdk:"invited_by"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	OrganizationName types.String `tfsdk:"organization_name"`
	DefaultRegionID  types.String `tfsdk:"default_region_id"`
	State            types.String `tfsdk:"state"`
	Type             types.String `tfsdk:"type"`
	Region           types.String `tfsdk:"region"`
	RegionID         types.String `tfsdk:"region_id"`
	RunnerID         types.String `tfsdk:"runner_id"`
	Role             types.String `tfsdk:"role"`
	AssignedRoleIDs  types.Set    `tfsdk:"assigned_role_ids"`
	Permissions      types.Set    `tfsdk:"permissions"`
	Target           types.String `tfsdk:"target"`
	URL              types.String `tfsdk:"url"`
	Username         types.String `tfsdk:"username"`
	Project          types.String `tfsdk:"project"`
	Public           types.Bool   `tfsdk:"public"`
	Personal         types.Bool   `tfsdk:"personal"`
	Suspended        types.Bool   `tfsdk:"suspended"`
	IsGlobal         types.Bool   `tfsdk:"is_global"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	LastUsedAt       types.String `tfsdk:"last_used_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *collectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.kind
}

func (d *collectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": computedDataSourceStringAttribute("Data source identifier."),
		"items": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Returned Daytona objects.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id":                computedDataSourceStringAttribute("Object ID."),
					"name":              computedDataSourceStringAttribute("Object name."),
					"value":             sensitiveComputedDataSourceStringAttribute("Sensitive or masked object value, when applicable."),
					"description":       computedDataSourceStringAttribute("Object description."),
					"email":             computedDataSourceStringAttribute("Email address, when applicable."),
					"user_id":           computedDataSourceStringAttribute("User ID, when applicable."),
					"created_by":        computedDataSourceStringAttribute("Creator user ID, when applicable."),
					"invited_by":        computedDataSourceStringAttribute("Inviter email address, when applicable."),
					"organization_id":   computedDataSourceStringAttribute("Owning organization ID."),
					"organization_name": computedDataSourceStringAttribute("Owning organization name."),
					"default_region_id": computedDataSourceStringAttribute("Default organization region ID."),
					"state":             computedDataSourceStringAttribute("Object state."),
					"type":              computedDataSourceStringAttribute("Object type."),
					"region":            computedDataSourceStringAttribute("Region name."),
					"region_id":         computedDataSourceStringAttribute("Region ID."),
					"runner_id":         computedDataSourceStringAttribute("Runner ID."),
					"role":              computedDataSourceStringAttribute("Organization member role."),
					"assigned_role_ids": computedDataSourceStringSetAttribute("Assigned organization role IDs."),
					"permissions":       computedDataSourceStringSetAttribute("Assigned permissions."),
					"target":            computedDataSourceStringAttribute("Target region or environment."),
					"url":               computedDataSourceStringAttribute("Object URL, when applicable."),
					"username":          computedDataSourceStringAttribute("Username, when applicable."),
					"project":           computedDataSourceStringAttribute("Project or namespace, when applicable."),
					"public":            computedDataSourceBoolAttribute("Whether the object is public, when applicable."),
					"personal":          computedDataSourceBoolAttribute("Whether the organization is personal."),
					"suspended":         computedDataSourceBoolAttribute("Whether the organization is suspended."),
					"is_global":         computedDataSourceBoolAttribute("Whether the role is a global Daytona role."),
					"expires_at":        computedDataSourceStringAttribute("Expiration timestamp, when applicable."),
					"last_used_at":      computedDataSourceStringAttribute("Last-used timestamp, when applicable."),
					"created_at":        computedDataSourceStringAttribute("Creation timestamp."),
					"updated_at":        computedDataSourceStringAttribute("Update timestamp."),
				},
			},
		},
	}

	if d.requiresOrganizationID() {
		attributes["organization_id"] = schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Daytona organization ID to read.",
		}
	} else {
		attributes["organization_id"] = computedDataSourceStringAttribute("Organization ID, when this data source is scoped to one organization.")
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: d.markdownDescription(),
		Attributes:          attributes,
	}
}

func (d *collectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data collectionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.readItems(ctx, data.OrganizationID, &resp.Diagnostics)
	if err != nil {
		return
	}

	data.ID = types.StringValue(d.kind)
	data.Items = items
	if !d.requiresOrganizationID() {
		data.OrganizationID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *collectionDataSource) readItems(ctx context.Context, organizationID types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	switch d.kind {
	case "api_keys":
		apiKeys, httpResp, err := d.client.api.ApiKeysAPI.ListApiKeys(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona API keys", "list API keys", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(apiKeys))
		for _, apiKey := range apiKeys {
			item := newCollectionItem()
			item.ID = types.StringValue(apiKey.Name)
			item.Name = types.StringValue(apiKey.Name)
			item.Value = types.StringValue(apiKey.Value)
			item.Permissions = setStringValue(ctx, apiKey.Permissions)
			item.UserID = types.StringValue(apiKey.UserId)
			item.CreatedAt = terraformTimeString(apiKey.CreatedAt)
			if value, ok := apiKey.GetLastUsedAtOk(); ok && value != nil {
				item.LastUsedAt = terraformTimeString(*value)
			}
			if value, ok := apiKey.GetExpiresAtOk(); ok && value != nil {
				item.ExpiresAt = terraformTimeString(*value)
			}
			items = append(items, item)
		}
		return items, nil
	case "volumes":
		volumes, httpResp, err := d.client.api.VolumesAPI.ListVolumes(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona volumes", "list volumes", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(volumes))
		for _, volume := range volumes {
			item := newCollectionItem()
			item.ID = types.StringValue(volume.Id)
			item.Name = types.StringValue(volume.Name)
			item.OrganizationID = types.StringValue(volume.OrganizationId)
			item.State = types.StringValue(string(volume.State))
			item.CreatedAt = types.StringValue(volume.CreatedAt)
			item.UpdatedAt = types.StringValue(volume.UpdatedAt)
			items = append(items, item)
		}
		return items, nil
	case "regions":
		regions, httpResp, err := d.client.api.OrganizationsAPI.ListAvailableRegions(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona regions", "list regions", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(regions))
		for _, region := range regions {
			item := newCollectionItem()
			item.ID = types.StringValue(region.Id)
			item.Name = types.StringValue(region.Name)
			item.Type = types.StringValue(string(region.RegionType))
			item.CreatedAt = types.StringValue(region.CreatedAt)
			item.UpdatedAt = types.StringValue(region.UpdatedAt)
			if value, ok := region.GetOrganizationIdOk(); ok && value != nil {
				item.OrganizationID = types.StringValue(*value)
			}
			items = append(items, item)
		}
		return items, nil
	case "shared_regions":
		regions, httpResp, err := d.client.api.RegionsAPI.ListSharedRegions(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona shared regions", "list shared regions", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(regions))
		for _, region := range regions {
			item := newCollectionItem()
			item.ID = types.StringValue(region.Id)
			item.Name = types.StringValue(region.Name)
			item.Type = types.StringValue(string(region.RegionType))
			item.CreatedAt = types.StringValue(region.CreatedAt)
			item.UpdatedAt = types.StringValue(region.UpdatedAt)
			if value, ok := region.GetOrganizationIdOk(); ok && value != nil {
				item.OrganizationID = types.StringValue(*value)
			}
			items = append(items, item)
		}
		return items, nil
	case "runners":
		runners, httpResp, err := d.client.api.RunnersAPI.ListRunners(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona runners", "list runners", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(runners))
		for _, runner := range runners {
			item := newCollectionItem()
			item.ID = types.StringValue(runner.Id)
			item.Name = types.StringValue(runner.Name)
			item.State = types.StringValue(string(runner.State))
			item.Region = types.StringValue(runner.Region)
			item.CreatedAt = types.StringValue(runner.CreatedAt)
			item.UpdatedAt = types.StringValue(runner.UpdatedAt)
			items = append(items, item)
		}
		return items, nil
	case "sandboxes":
		var items []collectionItemModel
		cursor := ""
		for {
			request := d.client.api.SandboxAPI.ListSandboxes(ctx).Limit(100)
			if cursor != "" {
				request = request.Cursor(cursor)
			}
			sandboxes, httpResp, err := request.Execute()
			if err != nil {
				addAPIError(diags, "Unable to list Daytona sandboxes", "list sandboxes", httpResp, err)
				return nil, err
			}
			for _, sandbox := range sandboxes.Items {
				item := newCollectionItem()
				item.ID = types.StringValue(sandbox.Id)
				item.Name = types.StringValue(sandbox.Name)
				item.OrganizationID = types.StringValue(sandbox.OrganizationId)
				item.Target = types.StringValue(sandbox.Target)
				item.Public = types.BoolValue(sandbox.Public)
				item.CreatedAt = pointerStringValue(sandbox.CreatedAt)
				item.UpdatedAt = pointerStringValue(sandbox.UpdatedAt)
				if sandbox.State != nil {
					item.State = types.StringValue(string(*sandbox.State))
				}
				if sandbox.RunnerId != nil {
					item.RunnerID = types.StringValue(*sandbox.RunnerId)
				}
				items = append(items, item)
			}
			next := sandboxes.NextCursor.Get()
			if next == nil || *next == "" || *next == cursor || len(sandboxes.Items) == 0 {
				break
			}
			cursor = *next
		}
		return items, nil
	case "snapshots":
		var items []collectionItemModel
		for page := float32(1); ; page++ {
			snapshots, httpResp, err := d.client.api.SnapshotsAPI.GetAllSnapshots(ctx).Page(page).Limit(100).Execute()
			if err != nil {
				addAPIError(diags, "Unable to list Daytona snapshots", "list snapshots", httpResp, err)
				return nil, err
			}
			for _, snapshot := range snapshots.Items {
				item := newCollectionItem()
				item.ID = types.StringValue(snapshot.Id)
				item.Name = types.StringValue(snapshot.Name)
				item.State = types.StringValue(string(snapshot.State))
				item.CreatedAt = types.StringValue(snapshot.CreatedAt.Format(time.RFC3339))
				item.UpdatedAt = types.StringValue(snapshot.UpdatedAt.Format(time.RFC3339))
				if snapshot.OrganizationId != nil {
					item.OrganizationID = types.StringValue(*snapshot.OrganizationId)
				}
				items = append(items, item)
			}
			if page >= snapshots.TotalPages || len(snapshots.Items) == 0 {
				break
			}
		}
		return items, nil
	case "docker_registries":
		registries, httpResp, err := d.client.api.DockerRegistryAPI.ListRegistries(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona Docker registries", "list Docker registries", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(registries))
		for _, registry := range registries {
			item := newCollectionItem()
			item.ID = types.StringValue(registry.Id)
			item.Name = types.StringValue(registry.Name)
			item.URL = types.StringValue(registry.Url)
			item.Username = types.StringValue(registry.Username)
			item.Project = types.StringValue(registry.Project)
			item.Type = types.StringValue(registry.RegistryType)
			item.CreatedAt = types.StringValue(registry.CreatedAt.Format(time.RFC3339))
			item.UpdatedAt = types.StringValue(registry.UpdatedAt.Format(time.RFC3339))
			items = append(items, item)
		}
		return items, nil
	case "organizations":
		organizations, httpResp, err := d.client.api.OrganizationsAPI.ListOrganizations(ctx).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona organizations", "list organizations", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(organizations))
		for _, organization := range organizations {
			item := newCollectionItem()
			item.ID = types.StringValue(organization.Id)
			item.Name = types.StringValue(organization.Name)
			item.CreatedBy = types.StringValue(organization.CreatedBy)
			item.Personal = types.BoolValue(organization.Personal)
			item.Suspended = types.BoolValue(organization.Suspended)
			item.CreatedAt = terraformTimeString(organization.CreatedAt)
			item.UpdatedAt = terraformTimeString(organization.UpdatedAt)
			if value, ok := organization.GetDefaultRegionIdOk(); ok && value != nil {
				item.DefaultRegionID = types.StringValue(*value)
			}
			items = append(items, item)
		}
		return items, nil
	case "organization_roles":
		roles, httpResp, err := d.client.api.OrganizationsAPI.ListOrganizationRoles(ctx, organizationID.ValueString()).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona organization roles", "list organization roles", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(roles))
		for _, role := range roles {
			item := newCollectionItem()
			item.ID = types.StringValue(role.Id)
			item.OrganizationID = organizationID
			item.Name = types.StringValue(role.Name)
			item.Description = types.StringValue(role.Description)
			item.Permissions = setStringValue(ctx, role.Permissions)
			item.IsGlobal = types.BoolValue(role.IsGlobal)
			item.CreatedAt = terraformTimeString(role.CreatedAt)
			item.UpdatedAt = terraformTimeString(role.UpdatedAt)
			items = append(items, item)
		}
		return items, nil
	case "organization_members":
		members, httpResp, err := d.client.api.OrganizationsAPI.ListOrganizationMembers(ctx, organizationID.ValueString()).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona organization members", "list organization members", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(members))
		for _, member := range members {
			item := newCollectionItem()
			item.ID = types.StringValue(member.UserId)
			item.OrganizationID = types.StringValue(member.OrganizationId)
			item.Name = types.StringValue(member.Name)
			item.Email = types.StringValue(member.Email)
			item.Role = types.StringValue(member.Role)
			item.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(member.AssignedRoles))
			item.CreatedAt = terraformTimeString(member.CreatedAt)
			item.UpdatedAt = terraformTimeString(member.UpdatedAt)
			items = append(items, item)
		}
		return items, nil
	case "organization_invitations":
		invitations, httpResp, err := d.client.api.OrganizationsAPI.ListOrganizationInvitations(ctx, organizationID.ValueString()).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona organization invitations", "list organization invitations", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(invitations))
		for _, invitation := range invitations {
			item := newCollectionItem()
			item.ID = types.StringValue(invitation.Id)
			item.OrganizationID = types.StringValue(invitation.OrganizationId)
			item.OrganizationName = types.StringValue(invitation.OrganizationName)
			item.Email = types.StringValue(invitation.Email)
			item.InvitedBy = types.StringValue(invitation.InvitedBy)
			item.Role = types.StringValue(invitation.Role)
			item.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(invitation.AssignedRoles))
			item.State = types.StringValue(invitation.Status)
			item.ExpiresAt = terraformTimeString(invitation.ExpiresAt)
			item.CreatedAt = terraformTimeString(invitation.CreatedAt)
			item.UpdatedAt = terraformTimeString(invitation.UpdatedAt)
			items = append(items, item)
		}
		return items, nil
	default:
		diags.AddError("Unsupported Daytona data source", fmt.Sprintf("Unsupported data source kind %q.", d.kind))
		return nil, fmt.Errorf("unsupported data source kind %q", d.kind)
	}
}

func (d *collectionDataSource) requiresOrganizationID() bool {
	switch d.kind {
	case "organization_roles", "organization_members", "organization_invitations":
		return true
	default:
		return false
	}
}

func (d *collectionDataSource) displayName() string {
	return strings.ReplaceAll(d.kind, "_", " ")
}

func (d *collectionDataSource) markdownDescription() string {
	if d.kind == "runners" {
		return "Lists Daytona custom runners visible to the configured credentials. Daytona exposes this endpoint only when organization infrastructure is enabled for the organization."
	}

	return fmt.Sprintf("Lists Daytona %s visible to the configured credentials.", d.displayName())
}

func newCollectionItem() collectionItemModel {
	return collectionItemModel{
		ID:               types.StringNull(),
		Name:             types.StringNull(),
		Value:            types.StringNull(),
		Description:      types.StringNull(),
		Email:            types.StringNull(),
		UserID:           types.StringNull(),
		CreatedBy:        types.StringNull(),
		InvitedBy:        types.StringNull(),
		OrganizationID:   types.StringNull(),
		OrganizationName: types.StringNull(),
		DefaultRegionID:  types.StringNull(),
		State:            types.StringNull(),
		Type:             types.StringNull(),
		Region:           types.StringNull(),
		RegionID:         types.StringNull(),
		RunnerID:         types.StringNull(),
		Role:             types.StringNull(),
		AssignedRoleIDs:  types.SetNull(types.StringType),
		Permissions:      types.SetNull(types.StringType),
		Target:           types.StringNull(),
		URL:              types.StringNull(),
		Username:         types.StringNull(),
		Project:          types.StringNull(),
		Public:           types.BoolNull(),
		Personal:         types.BoolNull(),
		Suspended:        types.BoolNull(),
		IsGlobal:         types.BoolNull(),
		ExpiresAt:        types.StringNull(),
		LastUsedAt:       types.StringNull(),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
	}
}

func computedDataSourceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func computedDataSourceBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func sensitiveComputedDataSourceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:            true,
		Sensitive:           true,
		MarkdownDescription: description,
	}
}

func computedDataSourceStringSetAttribute(description string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Computed:            true,
		MarkdownDescription: description,
	}
}

func configureDataSourceClient(providerData any, diags *diag.Diagnostics) *daytonaClient {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*daytonaClient)
	if !ok {
		diags.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

func pointerStringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
