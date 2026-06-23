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
	return newCollectionDataSource("api_keys")
}

func NewVolumesDataSource() datasource.DataSource {
	return newCollectionDataSource("volumes")
}

func NewRegionsDataSource() datasource.DataSource {
	return newCollectionDataSource("regions")
}

func NewSharedRegionsDataSource() datasource.DataSource {
	return newCollectionDataSource("shared_regions")
}

func NewRunnersDataSource() datasource.DataSource {
	return newCollectionDataSource("runners")
}

func NewSandboxesDataSource() datasource.DataSource {
	return newCollectionDataSource("sandboxes")
}

func NewSnapshotsDataSource() datasource.DataSource {
	return newCollectionDataSource("snapshots")
}

func NewDockerRegistriesDataSource() datasource.DataSource {
	return newCollectionDataSource("docker_registries")
}

func NewOrganizationsDataSource() datasource.DataSource {
	return newCollectionDataSource("organizations")
}

func NewOrganizationRolesDataSource() datasource.DataSource {
	return newCollectionDataSource("organization_roles")
}

func NewOrganizationMembersDataSource() datasource.DataSource {
	return newCollectionDataSource("organization_members")
}

func NewOrganizationInvitationsDataSource() datasource.DataSource {
	return newCollectionDataSource("organization_invitations")
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
	if apiKey == nil {
		addEmptyAPIResponseError(&resp.Diagnostics, "Empty Daytona current API key response", "read current API key", httpResp)
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
	shape  collectionShape
}

type collectionReader func(context.Context, *daytonaClient, types.String, *diag.Diagnostics) ([]collectionItemModel, error)

type collectionShape struct {
	kind                   string
	markdownDescription    string
	requiresOrganizationID bool
	read                   collectionReader
}

func newCollectionDataSource(kind string) datasource.DataSource {
	shape, ok := collectionShapes[kind]
	if !ok {
		shape = collectionShape{kind: kind}
	}
	return &collectionDataSource{shape: shape}
}

type collectionDataSourceModel struct {
	ID             types.String          `tfsdk:"id"`
	OrganizationID types.String          `tfsdk:"organization_id"`
	Items          []collectionItemModel `tfsdk:"items"`
}

type collectionItemModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	State types.String `tfsdk:"state"`
}

func (d *collectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.shape.kind
}

func (d *collectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": computedDataSourceStringAttribute("Data source identifier."),
		"items": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Returned Daytona objects.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id":    computedDataSourceStringAttribute("Object ID."),
					"name":  computedDataSourceStringAttribute("Object name."),
					"state": computedDataSourceStringAttribute("Object state, when applicable."),
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

	data.ID = types.StringValue(d.shape.kind)
	data.Items = items
	if !d.requiresOrganizationID() {
		data.OrganizationID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *collectionDataSource) readItems(ctx context.Context, organizationID types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	if d.shape.read == nil {
		diags.AddError("Unsupported Daytona data source", fmt.Sprintf("Unsupported data source kind %q.", d.shape.kind))
		return nil, fmt.Errorf("unsupported data source kind %q", d.shape.kind)
	}
	return d.shape.read(ctx, d.client, organizationID, diags)
}

func (d *collectionDataSource) requiresOrganizationID() bool {
	return d.shape.requiresOrganizationID
}

func (d *collectionDataSource) markdownDescription() string {
	if d.shape.markdownDescription != "" {
		return d.shape.markdownDescription
	}
	return fmt.Sprintf("Lists Daytona %s visible to the configured credentials.", strings.ReplaceAll(d.shape.kind, "_", " "))
}

func newCollectionItem() collectionItemModel {
	return collectionItemModel{
		ID:    types.StringNull(),
		Name:  types.StringNull(),
		State: types.StringNull(),
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

func pointerStringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
