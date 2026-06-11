// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
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

func NewVolumesDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "volumes"}
}

func NewRegionsDataSource() datasource.DataSource {
	return &collectionDataSource{kind: "regions"}
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
	ID    types.String          `tfsdk:"id"`
	Items []collectionItemModel `tfsdk:"items"`
}

type collectionItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	State          types.String `tfsdk:"state"`
	Type           types.String `tfsdk:"type"`
	Region         types.String `tfsdk:"region"`
	RegionID       types.String `tfsdk:"region_id"`
	RunnerID       types.String `tfsdk:"runner_id"`
	Target         types.String `tfsdk:"target"`
	URL            types.String `tfsdk:"url"`
	Username       types.String `tfsdk:"username"`
	Project        types.String `tfsdk:"project"`
	Public         types.Bool   `tfsdk:"public"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *collectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.kind
}

func (d *collectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("Lists Daytona %s visible to the configured credentials.", d.kind),
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned Daytona objects.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":              computedDataSourceStringAttribute("Object ID."),
						"name":            computedDataSourceStringAttribute("Object name."),
						"organization_id": computedDataSourceStringAttribute("Owning organization ID."),
						"state":           computedDataSourceStringAttribute("Object state."),
						"type":            computedDataSourceStringAttribute("Object type."),
						"region":          computedDataSourceStringAttribute("Region name."),
						"region_id":       computedDataSourceStringAttribute("Region ID."),
						"runner_id":       computedDataSourceStringAttribute("Runner ID."),
						"target":          computedDataSourceStringAttribute("Target region or environment."),
						"url":             computedDataSourceStringAttribute("Object URL, when applicable."),
						"username":        computedDataSourceStringAttribute("Username, when applicable."),
						"project":         computedDataSourceStringAttribute("Project or namespace, when applicable."),
						"public":          computedDataSourceBoolAttribute("Whether the object is public, when applicable."),
						"created_at":      computedDataSourceStringAttribute("Creation timestamp."),
						"updated_at":      computedDataSourceStringAttribute("Update timestamp."),
					},
				},
			},
		},
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
	items, err := d.readItems(ctx, &resp.Diagnostics)
	if err != nil {
		return
	}

	data := collectionDataSourceModel{
		ID:    types.StringValue(d.kind),
		Items: items,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *collectionDataSource) readItems(ctx context.Context, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	switch d.kind {
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
		sandboxes, httpResp, err := d.client.api.SandboxAPI.ListSandboxes(ctx).Limit(100).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona sandboxes", "list sandboxes", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(sandboxes.Items))
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
		return items, nil
	case "snapshots":
		snapshots, httpResp, err := d.client.api.SnapshotsAPI.GetAllSnapshots(ctx).Limit(100).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona snapshots", "list snapshots", httpResp, err)
			return nil, err
		}
		items := make([]collectionItemModel, 0, len(snapshots.Items))
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
	default:
		diags.AddError("Unsupported Daytona data source", fmt.Sprintf("Unsupported data source kind %q.", d.kind))
		return nil, fmt.Errorf("unsupported data source kind %q", d.kind)
	}
}

func newCollectionItem() collectionItemModel {
	return collectionItemModel{
		ID:             types.StringNull(),
		Name:           types.StringNull(),
		OrganizationID: types.StringNull(),
		State:          types.StringNull(),
		Type:           types.StringNull(),
		Region:         types.StringNull(),
		RegionID:       types.StringNull(),
		RunnerID:       types.StringNull(),
		Target:         types.StringNull(),
		URL:            types.StringNull(),
		Username:       types.StringNull(),
		Project:        types.StringNull(),
		Public:         types.BoolNull(),
		CreatedAt:      types.StringNull(),
		UpdatedAt:      types.StringNull(),
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
