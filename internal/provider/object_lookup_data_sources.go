// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeDataSource{}
var _ datasource.DataSource = &DockerRegistryDataSource{}
var _ datasource.DataSource = &RegionDataSource{}

func NewVolumeDataSource() datasource.DataSource {
	return &VolumeDataSource{}
}

func NewDockerRegistryDataSource() datasource.DataSource {
	return &DockerRegistryDataSource{}
}

func NewRegionDataSource() datasource.DataSource {
	return &RegionDataSource{}
}

type VolumeDataSource struct {
	client *daytonaClient
}

type volumeDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	State          types.String `tfsdk:"state"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	LastUsedAt     types.String `tfsdk:"last_used_at"`
	ErrorReason    types.String `tfsdk:"error_reason"`
}

func (d *VolumeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (d *VolumeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona persistent volume by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              requiredDataSourceStringAttribute("Daytona volume ID."),
			"name":            computedDataSourceStringAttribute("Volume name."),
			"organization_id": computedDataSourceStringAttribute("Daytona organization ID that owns the volume."),
			"state":           computedDataSourceStringAttribute("Current volume state."),
			"created_at":      computedDataSourceStringAttribute("Volume creation timestamp."),
			"updated_at":      computedDataSourceStringAttribute("Volume update timestamp."),
			"last_used_at":    computedDataSourceStringAttribute("Volume last-used timestamp, when available."),
			"error_reason":    computedDataSourceStringAttribute("Volume error reason, when available."),
		},
	}
}

func (d *VolumeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *VolumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, httpResp, err := d.client.api.VolumesAPI.GetVolume(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona volume", "read volume", httpResp, err)
		return
	}
	if volume == nil {
		resp.Diagnostics.AddError("Empty Daytona volume response", fmt.Sprintf("Daytona returned a successful response without volume %q.", data.ID.ValueString()))
		return
	}

	data = flattenVolumeDataSource(volume)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type DockerRegistryDataSource struct {
	client *daytonaClient
}

type dockerRegistryDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Username     types.String `tfsdk:"username"`
	Project      types.String `tfsdk:"project"`
	RegistryType types.String `tfsdk:"registry_type"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (d *DockerRegistryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_registry"
}

func (d *DockerRegistryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona Docker registry by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            requiredDataSourceStringAttribute("Daytona Docker registry ID."),
			"name":          computedDataSourceStringAttribute("Registry name."),
			"url":           computedDataSourceStringAttribute("Registry URL."),
			"username":      computedDataSourceStringAttribute("Registry username."),
			"project":       computedDataSourceStringAttribute("Registry project or namespace."),
			"registry_type": computedDataSourceStringAttribute("Registry type."),
			"created_at":    computedDataSourceStringAttribute("Registry creation timestamp."),
			"updated_at":    computedDataSourceStringAttribute("Registry update timestamp."),
		},
	}
}

func (d *DockerRegistryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *DockerRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dockerRegistryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, httpResp, err := d.client.api.DockerRegistryAPI.GetRegistry(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona Docker registry", "read Docker registry", httpResp, err)
		return
	}
	if registry == nil {
		resp.Diagnostics.AddError("Empty Daytona Docker registry response", fmt.Sprintf("Daytona returned a successful response without Docker registry %q.", data.ID.ValueString()))
		return
	}

	data = flattenDockerRegistryDataSource(registry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type RegionDataSource struct {
	client *daytonaClient
}

type regionDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	RegionType         types.String `tfsdk:"region_type"`
	ProxyURL           types.String `tfsdk:"proxy_url"`
	SSHGatewayURL      types.String `tfsdk:"ssh_gateway_url"`
	SnapshotManagerURL types.String `tfsdk:"snapshot_manager_url"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (d *RegionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region"
}

func (d *RegionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona region by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                   requiredDataSourceStringAttribute("Daytona region ID."),
			"name":                 computedDataSourceStringAttribute("Region name."),
			"organization_id":      computedDataSourceStringAttribute("Daytona organization ID that owns the region."),
			"region_type":          computedDataSourceStringAttribute("Region type."),
			"proxy_url":            computedDataSourceStringAttribute("Proxy URL for the region."),
			"ssh_gateway_url":      computedDataSourceStringAttribute("SSH gateway URL for the region."),
			"snapshot_manager_url": computedDataSourceStringAttribute("Snapshot manager URL for the region."),
			"created_at":           computedDataSourceStringAttribute("Region creation timestamp."),
			"updated_at":           computedDataSourceStringAttribute("Region update timestamp."),
		},
	}
}

func (d *RegionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data regionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	region, httpResp, err := d.client.api.OrganizationsAPI.GetRegionById(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona region", "read region", httpResp, err)
		return
	}
	if region == nil {
		resp.Diagnostics.AddError("Empty Daytona region response", fmt.Sprintf("Daytona returned a successful response without region %q.", data.ID.ValueString()))
		return
	}

	data = flattenRegionDataSource(region)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenVolumeDataSource(volume *apiclient.VolumeDto) volumeDataSourceModel {
	data := volumeDataSourceModel{
		ID:             types.StringValue(volume.Id),
		Name:           types.StringValue(volume.Name),
		OrganizationID: types.StringValue(volume.OrganizationId),
		State:          types.StringValue(string(volume.State)),
		CreatedAt:      types.StringValue(volume.CreatedAt),
		UpdatedAt:      types.StringValue(volume.UpdatedAt),
		LastUsedAt:     types.StringNull(),
		ErrorReason:    types.StringNull(),
	}

	if value, ok := volume.GetLastUsedAtOk(); ok && value != nil {
		data.LastUsedAt = types.StringValue(*value)
	}
	if value, ok := volume.GetErrorReasonOk(); ok && value != nil {
		data.ErrorReason = types.StringValue(*value)
	}

	return data
}

func flattenDockerRegistryDataSource(registry *apiclient.DockerRegistry) dockerRegistryDataSourceModel {
	return dockerRegistryDataSourceModel{
		ID:           types.StringValue(registry.Id),
		Name:         types.StringValue(registry.Name),
		URL:          types.StringValue(registry.Url),
		Username:     types.StringValue(registry.Username),
		Project:      types.StringValue(registry.Project),
		RegistryType: types.StringValue(registry.RegistryType),
		CreatedAt:    types.StringValue(registry.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:    types.StringValue(registry.UpdatedAt.Format(time.RFC3339)),
	}
}

func flattenRegionDataSource(region *apiclient.Region) regionDataSourceModel {
	return regionDataSourceModel{
		ID:                 types.StringValue(region.Id),
		Name:               types.StringValue(region.Name),
		OrganizationID:     nullableStringValue(region.OrganizationId),
		RegionType:         types.StringValue(string(region.RegionType)),
		ProxyURL:           nullableStringValue(region.ProxyUrl),
		SSHGatewayURL:      nullableStringValue(region.SshGatewayUrl),
		SnapshotManagerURL: nullableStringValue(region.SnapshotManagerUrl),
		CreatedAt:          types.StringValue(region.CreatedAt),
		UpdatedAt:          types.StringValue(region.UpdatedAt),
	}
}
