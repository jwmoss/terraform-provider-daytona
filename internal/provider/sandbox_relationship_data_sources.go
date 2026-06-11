// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SandboxOrganizationDataSource{}
var _ datasource.DataSource = &SandboxRegionQuotaDataSource{}
var _ datasource.DataSource = &SandboxParentDataSource{}
var _ datasource.DataSource = &SandboxAncestorsDataSource{}
var _ datasource.DataSource = &SandboxForksDataSource{}
var _ datasource.DataSource = &SandboxToolboxProxyURLDataSource{}

func NewSandboxOrganizationDataSource() datasource.DataSource {
	return &SandboxOrganizationDataSource{}
}

func NewSandboxRegionQuotaDataSource() datasource.DataSource {
	return &SandboxRegionQuotaDataSource{}
}

func NewSandboxParentDataSource() datasource.DataSource {
	return &SandboxParentDataSource{}
}

func NewSandboxAncestorsDataSource() datasource.DataSource {
	return &SandboxAncestorsDataSource{}
}

func NewSandboxForksDataSource() datasource.DataSource {
	return &SandboxForksDataSource{}
}

func NewSandboxToolboxProxyURLDataSource() datasource.DataSource {
	return &SandboxToolboxProxyURLDataSource{}
}

type SandboxOrganizationDataSource struct {
	client *daytonaClient
}

type sandboxOrganizationDataSourceModel struct {
	SandboxID                           types.String  `tfsdk:"sandbox_id"`
	RequestOrganizationID               types.String  `tfsdk:"request_organization_id"`
	ID                                  types.String  `tfsdk:"id"`
	Name                                types.String  `tfsdk:"name"`
	DefaultRegionID                     types.String  `tfsdk:"default_region_id"`
	CreatedBy                           types.String  `tfsdk:"created_by"`
	Personal                            types.Bool    `tfsdk:"personal"`
	Suspended                           types.Bool    `tfsdk:"suspended"`
	SuspendedAt                         types.String  `tfsdk:"suspended_at"`
	SuspensionReason                    types.String  `tfsdk:"suspension_reason"`
	SuspendedUntil                      types.String  `tfsdk:"suspended_until"`
	SuspensionCleanupGracePeriodHours   types.Float64 `tfsdk:"suspension_cleanup_grace_period_hours"`
	MaxCPUPerSandbox                    types.Float64 `tfsdk:"max_cpu_per_sandbox"`
	MaxMemoryPerSandbox                 types.Float64 `tfsdk:"max_memory_per_sandbox"`
	MaxDiskPerSandbox                   types.Float64 `tfsdk:"max_disk_per_sandbox"`
	SnapshotDeactivationTimeoutMinutes  types.Float64 `tfsdk:"snapshot_deactivation_timeout_minutes"`
	SandboxLimitedNetworkEgress         types.Bool    `tfsdk:"sandbox_limited_network_egress"`
	AuthenticatedRateLimit              types.Float64 `tfsdk:"authenticated_rate_limit"`
	SandboxCreateRateLimit              types.Float64 `tfsdk:"sandbox_create_rate_limit"`
	SandboxLifecycleRateLimit           types.Float64 `tfsdk:"sandbox_lifecycle_rate_limit"`
	AuthenticatedRateLimitTTLSeconds    types.Float64 `tfsdk:"authenticated_rate_limit_ttl_seconds"`
	SandboxCreateRateLimitTTLSeconds    types.Float64 `tfsdk:"sandbox_create_rate_limit_ttl_seconds"`
	SandboxLifecycleRateLimitTTLSeconds types.Float64 `tfsdk:"sandbox_lifecycle_rate_limit_ttl_seconds"`
	ExperimentalConfigJSON              types.String  `tfsdk:"experimental_config_json"`
	CreatedAt                           types.String  `tfsdk:"created_at"`
	UpdatedAt                           types.String  `tfsdk:"updated_at"`
}

func (d *SandboxOrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_organization"
}

func (d *SandboxOrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Daytona organization that owns a sandbox.",
		Attributes: map[string]schema.Attribute{
			"sandbox_id":                               requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id":                  optionalOrganizationIDDataSourceStringAttribute(),
			"id":                                       computedDataSourceStringAttribute("Daytona organization ID."),
			"name":                                     computedDataSourceStringAttribute("Organization name."),
			"default_region_id":                        computedDataSourceStringAttribute("Default Daytona region ID for the organization."),
			"created_by":                               computedDataSourceStringAttribute("User ID of the organization creator."),
			"personal":                                 computedDataSourceBoolAttribute("Whether this is a personal organization."),
			"suspended":                                computedDataSourceBoolAttribute("Whether the organization is suspended."),
			"suspended_at":                             computedDataSourceStringAttribute("Organization suspension timestamp, when available."),
			"suspension_reason":                        computedDataSourceStringAttribute("Organization suspension reason, when available."),
			"suspended_until":                          computedDataSourceStringAttribute("Suspension end timestamp, when available."),
			"suspension_cleanup_grace_period_hours":    computedDataSourceFloat64Attribute("Suspension cleanup grace period in hours."),
			"max_cpu_per_sandbox":                      computedDataSourceFloat64Attribute("Maximum CPU per sandbox."),
			"max_memory_per_sandbox":                   computedDataSourceFloat64Attribute("Maximum memory per sandbox."),
			"max_disk_per_sandbox":                     computedDataSourceFloat64Attribute("Maximum disk per sandbox."),
			"snapshot_deactivation_timeout_minutes":    computedDataSourceFloat64Attribute("Snapshot deactivation timeout in minutes."),
			"sandbox_limited_network_egress":           computedDataSourceBoolAttribute("Default limited network egress setting for new sandboxes."),
			"authenticated_rate_limit":                 computedDataSourceFloat64Attribute("Authenticated request rate limit per minute."),
			"sandbox_create_rate_limit":                computedDataSourceFloat64Attribute("Sandbox create rate limit per minute."),
			"sandbox_lifecycle_rate_limit":             computedDataSourceFloat64Attribute("Sandbox lifecycle rate limit per minute."),
			"authenticated_rate_limit_ttl_seconds":     computedDataSourceFloat64Attribute("Authenticated request rate-limit TTL in seconds."),
			"sandbox_create_rate_limit_ttl_seconds":    computedDataSourceFloat64Attribute("Sandbox create rate-limit TTL in seconds."),
			"sandbox_lifecycle_rate_limit_ttl_seconds": computedDataSourceFloat64Attribute("Sandbox lifecycle rate-limit TTL in seconds."),
			"experimental_config_json":                 computedDataSourceStringAttribute("Experimental organization configuration as a JSON object string."),
			"created_at":                               computedDataSourceStringAttribute("Organization creation timestamp."),
			"updated_at":                               computedDataSourceStringAttribute("Organization update timestamp."),
		},
	}
}

func (d *SandboxOrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxOrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxOrganizationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetOrganizationBySandboxId(ctx, data.SandboxID.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	organization, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox organization", "read sandbox organization", httpResp, err)
		return
	}
	if organization == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox organization response", fmt.Sprintf("Daytona returned a successful response without an organization for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data = flattenSandboxOrganization(organization, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxRegionQuotaDataSource struct {
	client *daytonaClient
}

type sandboxRegionQuotaDataSourceModel struct {
	ID                            types.String  `tfsdk:"id"`
	SandboxID                     types.String  `tfsdk:"sandbox_id"`
	RequestOrganizationID         types.String  `tfsdk:"request_organization_id"`
	OrganizationID                types.String  `tfsdk:"organization_id"`
	RegionID                      types.String  `tfsdk:"region_id"`
	SandboxClass                  types.String  `tfsdk:"sandbox_class"`
	TotalCPUQuota                 types.Float64 `tfsdk:"total_cpu_quota"`
	TotalMemoryQuota              types.Float64 `tfsdk:"total_memory_quota"`
	TotalDiskQuota                types.Float64 `tfsdk:"total_disk_quota"`
	TotalGPUQuota                 types.Float64 `tfsdk:"total_gpu_quota"`
	AllowedGPUTypes               types.List    `tfsdk:"allowed_gpu_types"`
	MaxCPUPerSandbox              types.Float64 `tfsdk:"max_cpu_per_sandbox"`
	MaxMemoryPerSandbox           types.Float64 `tfsdk:"max_memory_per_sandbox"`
	MaxDiskPerSandbox             types.Float64 `tfsdk:"max_disk_per_sandbox"`
	MaxDiskPerNonEphemeralSandbox types.Float64 `tfsdk:"max_disk_per_non_ephemeral_sandbox"`
	MaxCPUPerGPUSandbox           types.Float64 `tfsdk:"max_cpu_per_gpu_sandbox"`
	MaxMemoryPerGPUSandbox        types.Float64 `tfsdk:"max_memory_per_gpu_sandbox"`
	MaxDiskPerGPUSandbox          types.Float64 `tfsdk:"max_disk_per_gpu_sandbox"`
}

func (d *SandboxRegionQuotaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_region_quota"
}

func (d *SandboxRegionQuotaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Daytona region quota that applies to a sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                                 computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":                         requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id":            optionalOrganizationIDDataSourceStringAttribute(),
			"organization_id":                    computedDataSourceStringAttribute("Daytona organization ID for the quota."),
			"region_id":                          computedDataSourceStringAttribute("Daytona region ID for the quota."),
			"sandbox_class":                      computedDataSourceStringAttribute("Sandbox class."),
			"total_cpu_quota":                    computedDataSourceFloat64Attribute("Total CPU quota."),
			"total_memory_quota":                 computedDataSourceFloat64Attribute("Total memory quota."),
			"total_disk_quota":                   computedDataSourceFloat64Attribute("Total disk quota."),
			"total_gpu_quota":                    computedDataSourceFloat64Attribute("Total GPU quota."),
			"allowed_gpu_types":                  computedDataSourceStringListAttribute("Allowed GPU types."),
			"max_cpu_per_sandbox":                computedDataSourceFloat64Attribute("Maximum CPU per sandbox."),
			"max_memory_per_sandbox":             computedDataSourceFloat64Attribute("Maximum memory per sandbox."),
			"max_disk_per_sandbox":               computedDataSourceFloat64Attribute("Maximum disk per sandbox."),
			"max_disk_per_non_ephemeral_sandbox": computedDataSourceFloat64Attribute("Maximum disk per non-ephemeral sandbox."),
			"max_cpu_per_gpu_sandbox":            computedDataSourceFloat64Attribute("Maximum CPU per GPU sandbox."),
			"max_memory_per_gpu_sandbox":         computedDataSourceFloat64Attribute("Maximum memory per GPU sandbox."),
			"max_disk_per_gpu_sandbox":           computedDataSourceFloat64Attribute("Maximum disk per GPU sandbox."),
		},
	}
}

func (d *SandboxRegionQuotaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxRegionQuotaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxRegionQuotaDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetRegionQuotaBySandboxId(ctx, data.SandboxID.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	quota, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox region quota", "read sandbox region quota", httpResp, err)
		return
	}
	if quota == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox region quota response", fmt.Sprintf("Daytona returned a successful response without region quota data for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data = flattenSandboxRegionQuota(ctx, data.SandboxID.ValueString(), quota, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxParentDataSource struct {
	client *daytonaClient
}

type sandboxRelationshipDataSourceModel struct {
	SandboxIDOrName       types.String `tfsdk:"sandbox_id_or_name"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	OrganizationID        types.String `tfsdk:"organization_id"`
	Snapshot              types.String `tfsdk:"snapshot"`
	User                  types.String `tfsdk:"user"`
	Env                   types.Map    `tfsdk:"env"`
	Labels                types.Map    `tfsdk:"labels"`
	Public                types.Bool   `tfsdk:"public"`
	NetworkBlockAll       types.Bool   `tfsdk:"network_block_all"`
	NetworkAllowList      types.String `tfsdk:"network_allow_list"`
	Target                types.String `tfsdk:"target"`
	CPU                   types.Int64  `tfsdk:"cpu"`
	GPU                   types.Int64  `tfsdk:"gpu"`
	Memory                types.Int64  `tfsdk:"memory"`
	Disk                  types.Int64  `tfsdk:"disk"`
	AutoStopInterval      types.Int64  `tfsdk:"auto_stop_interval"`
	AutoArchiveInterval   types.Int64  `tfsdk:"auto_archive_interval"`
	AutoDeleteInterval    types.Int64  `tfsdk:"auto_delete_interval"`
	State                 types.String `tfsdk:"state"`
	RunnerID              types.String `tfsdk:"runner_id"`
	ToolboxProxyURL       types.String `tfsdk:"toolbox_proxy_url"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	ErrorReason           types.String `tfsdk:"error_reason"`
}

type sandboxRelationshipItemModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	Snapshot            types.String `tfsdk:"snapshot"`
	User                types.String `tfsdk:"user"`
	Env                 types.Map    `tfsdk:"env"`
	Labels              types.Map    `tfsdk:"labels"`
	Public              types.Bool   `tfsdk:"public"`
	NetworkBlockAll     types.Bool   `tfsdk:"network_block_all"`
	NetworkAllowList    types.String `tfsdk:"network_allow_list"`
	Target              types.String `tfsdk:"target"`
	CPU                 types.Int64  `tfsdk:"cpu"`
	GPU                 types.Int64  `tfsdk:"gpu"`
	Memory              types.Int64  `tfsdk:"memory"`
	Disk                types.Int64  `tfsdk:"disk"`
	AutoStopInterval    types.Int64  `tfsdk:"auto_stop_interval"`
	AutoArchiveInterval types.Int64  `tfsdk:"auto_archive_interval"`
	AutoDeleteInterval  types.Int64  `tfsdk:"auto_delete_interval"`
	State               types.String `tfsdk:"state"`
	RunnerID            types.String `tfsdk:"runner_id"`
	ToolboxProxyURL     types.String `tfsdk:"toolbox_proxy_url"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	ErrorReason         types.String `tfsdk:"error_reason"`
}

type sandboxRelationshipCollectionDataSourceModel struct {
	ID                    types.String                   `tfsdk:"id"`
	SandboxIDOrName       types.String                   `tfsdk:"sandbox_id_or_name"`
	RequestOrganizationID types.String                   `tfsdk:"request_organization_id"`
	IncludeDestroyed      types.Bool                     `tfsdk:"include_destroyed"`
	Items                 []sandboxRelationshipItemModel `tfsdk:"items"`
}

func (d *SandboxParentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_parent"
}

func (d *SandboxParentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the fork parent of a Daytona sandbox.",
		Attributes:          sandboxRelationshipDataSourceAttributes(),
	}
}

func (d *SandboxParentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxParentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxRelationshipDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxParent(ctx, data.SandboxIDOrName.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	parent, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox parent", "read sandbox parent", httpResp, err)
		return
	}
	if parent == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox parent response", fmt.Sprintf("Daytona returned a successful response without parent data for sandbox %q.", data.SandboxIDOrName.ValueString()))
		return
	}

	data = flattenSandboxRelationshipDataSource(ctx, parent, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxAncestorsDataSource struct {
	client *daytonaClient
}

func (d *SandboxAncestorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_ancestors"
}

func (d *SandboxAncestorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the fork ancestor chain of a Daytona sandbox.",
		Attributes:          sandboxRelationshipCollectionAttributes(false),
	}
}

func (d *SandboxAncestorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxAncestorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxRelationshipCollectionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxAncestors(ctx, data.SandboxIDOrName.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	ancestors, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox ancestors", "read sandbox ancestors", httpResp, err)
		return
	}

	data.ID = types.StringValue(data.SandboxIDOrName.ValueString() + ":ancestors")
	data.Items = flattenSandboxRelationshipItems(ctx, ancestors)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxForksDataSource struct {
	client *daytonaClient
}

func (d *SandboxForksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_forks"
}

func (d *SandboxForksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads fork children of a Daytona sandbox.",
		Attributes:          sandboxRelationshipCollectionAttributes(true),
	}
}

func (d *SandboxForksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxForksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxRelationshipCollectionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxForks(ctx, data.SandboxIDOrName.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if terraformBoolConfigured(data.IncludeDestroyed) {
		request = request.IncludeDestroyed(data.IncludeDestroyed.ValueBool())
	}

	forks, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox forks", "read sandbox forks", httpResp, err)
		return
	}

	data.ID = types.StringValue(data.SandboxIDOrName.ValueString() + ":forks")
	data.Items = flattenSandboxRelationshipItems(ctx, forks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxToolboxProxyURLDataSource struct {
	client *daytonaClient
}

type sandboxToolboxProxyURLDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	SandboxID             types.String `tfsdk:"sandbox_id"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
	URL                   types.String `tfsdk:"url"`
}

func (d *SandboxToolboxProxyURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_toolbox_proxy_url"
}

func (d *SandboxToolboxProxyURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the toolbox proxy URL for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":              requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"url":                     computedDataSourceStringAttribute("Toolbox proxy URL for the sandbox."),
		},
	}
}

func (d *SandboxToolboxProxyURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxToolboxProxyURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxToolboxProxyURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetToolboxProxyUrl(ctx, data.SandboxID.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	proxyURL, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox toolbox proxy URL", "read sandbox toolbox proxy URL", httpResp, err)
		return
	}
	if proxyURL == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox toolbox proxy URL response", fmt.Sprintf("Daytona returned a successful response without toolbox proxy URL data for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data.ID = types.StringValue(data.SandboxID.ValueString() + ":toolbox_proxy_url")
	data.URL = types.StringValue(proxyURL.Url)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func sandboxRelationshipDataSourceAttributes() map[string]schema.Attribute {
	attributes := sandboxRelationshipComputedAttributes()
	attributes["sandbox_id_or_name"] = requiredDataSourceStringAttribute("Daytona sandbox ID or name whose parent should be read.")
	attributes["request_organization_id"] = optionalOrganizationIDDataSourceStringAttribute()
	return attributes
}

func sandboxRelationshipCollectionAttributes(includeDestroyed bool) map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{
		"id":                      computedDataSourceStringAttribute("Data source identifier."),
		"sandbox_id_or_name":      requiredDataSourceStringAttribute("Daytona sandbox ID or name."),
		"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
		"items": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Returned Daytona sandboxes.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: sandboxRelationshipComputedAttributes(),
			},
		},
	}
	if includeDestroyed {
		attributes["include_destroyed"] = schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Whether destroyed fork children should be included.",
		}
	}
	return attributes
}

func sandboxRelationshipComputedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                    computedDataSourceStringAttribute("Daytona sandbox ID."),
		"name":                  computedDataSourceStringAttribute("Sandbox name."),
		"organization_id":       computedDataSourceStringAttribute("Daytona organization ID that owns the sandbox."),
		"snapshot":              computedDataSourceStringAttribute("Snapshot ID or name used to create the sandbox."),
		"user":                  computedDataSourceStringAttribute("User associated with the sandbox project."),
		"env":                   sensitiveComputedDataSourceStringMapAttribute("Environment variables for the sandbox."),
		"labels":                computedDataSourceStringMapAttribute("Labels for the sandbox."),
		"public":                computedDataSourceBoolAttribute("Whether HTTP previews are publicly accessible."),
		"network_block_all":     computedDataSourceBoolAttribute("Whether all sandbox network access is blocked."),
		"network_allow_list":    computedDataSourceStringAttribute("Comma-separated list of allowed CIDR network addresses."),
		"target":                computedDataSourceStringAttribute("Target region where the sandbox is created."),
		"cpu":                   computedDataSourceInt64Attribute("CPU cores allocated to the sandbox."),
		"gpu":                   computedDataSourceInt64Attribute("GPU units allocated to the sandbox."),
		"memory":                computedDataSourceInt64Attribute("Memory allocated to the sandbox in GB."),
		"disk":                  computedDataSourceInt64Attribute("Disk allocated to the sandbox in GB."),
		"auto_stop_interval":    computedDataSourceInt64Attribute("Auto-stop interval in minutes."),
		"auto_archive_interval": computedDataSourceInt64Attribute("Auto-archive interval in minutes."),
		"auto_delete_interval":  computedDataSourceInt64Attribute("Auto-delete interval in minutes."),
		"state":                 computedDataSourceStringAttribute("Current sandbox state."),
		"runner_id":             computedDataSourceStringAttribute("Runner ID hosting the sandbox, when assigned."),
		"toolbox_proxy_url":     computedDataSourceStringAttribute("Toolbox proxy URL for the sandbox."),
		"created_at":            computedDataSourceStringAttribute("Sandbox creation timestamp."),
		"updated_at":            computedDataSourceStringAttribute("Sandbox update timestamp."),
		"error_reason":          computedDataSourceStringAttribute("Sandbox error reason, when available."),
	}
}

func flattenSandboxOrganization(organization *apiclient.Organization, prior sandboxOrganizationDataSourceModel) sandboxOrganizationDataSourceModel {
	flattened := flattenOrganization(organization, organizationResourceModel{})

	prior.ID = flattened.ID
	prior.Name = flattened.Name
	prior.DefaultRegionID = flattened.DefaultRegionID
	prior.CreatedBy = flattened.CreatedBy
	prior.Personal = flattened.Personal
	prior.Suspended = flattened.Suspended
	prior.SuspendedAt = flattened.SuspendedAt
	prior.SuspensionReason = flattened.SuspensionReason
	prior.SuspendedUntil = flattened.SuspendedUntil
	prior.SuspensionCleanupGracePeriodHours = flattened.SuspensionCleanupGracePeriodHours
	prior.MaxCPUPerSandbox = flattened.MaxCPUPerSandbox
	prior.MaxMemoryPerSandbox = flattened.MaxMemoryPerSandbox
	prior.MaxDiskPerSandbox = flattened.MaxDiskPerSandbox
	prior.SnapshotDeactivationTimeoutMinutes = flattened.SnapshotDeactivationTimeoutMinutes
	prior.SandboxLimitedNetworkEgress = flattened.SandboxLimitedNetworkEgress
	prior.AuthenticatedRateLimit = flattened.AuthenticatedRateLimit
	prior.SandboxCreateRateLimit = flattened.SandboxCreateRateLimit
	prior.SandboxLifecycleRateLimit = flattened.SandboxLifecycleRateLimit
	prior.AuthenticatedRateLimitTTLSeconds = flattened.AuthenticatedRateLimitTTLSeconds
	prior.SandboxCreateRateLimitTTLSeconds = flattened.SandboxCreateRateLimitTTLSeconds
	prior.SandboxLifecycleRateLimitTTLSeconds = flattened.SandboxLifecycleRateLimitTTLSeconds
	prior.ExperimentalConfigJSON = flattened.ExperimentalConfigJSON
	prior.CreatedAt = flattened.CreatedAt
	prior.UpdatedAt = flattened.UpdatedAt

	return prior
}

func flattenSandboxRegionQuota(ctx context.Context, sandboxID string, quota *apiclient.RegionQuota, prior sandboxRegionQuotaDataSourceModel) sandboxRegionQuotaDataSourceModel {
	allowedGPUTypes := make([]string, 0, len(quota.AllowedGpuTypes))
	for _, gpuType := range quota.AllowedGpuTypes {
		allowedGPUTypes = append(allowedGPUTypes, string(gpuType))
	}

	prior.ID = types.StringValue(sandboxID + ":region_quota")
	prior.OrganizationID = types.StringValue(quota.OrganizationId)
	prior.RegionID = types.StringValue(quota.RegionId)
	prior.SandboxClass = types.StringValue(string(quota.SandboxClass))
	prior.TotalCPUQuota = float64Value(quota.TotalCpuQuota)
	prior.TotalMemoryQuota = float64Value(quota.TotalMemoryQuota)
	prior.TotalDiskQuota = float64Value(quota.TotalDiskQuota)
	prior.TotalGPUQuota = float64Value(quota.TotalGpuQuota)
	prior.AllowedGPUTypes = listStringValue(ctx, allowedGPUTypes)
	prior.MaxCPUPerSandbox = nullableFloat32(quota.MaxCpuPerSandbox)
	prior.MaxMemoryPerSandbox = nullableFloat32(quota.MaxMemoryPerSandbox)
	prior.MaxDiskPerSandbox = nullableFloat32(quota.MaxDiskPerSandbox)
	prior.MaxDiskPerNonEphemeralSandbox = nullableFloat32(quota.MaxDiskPerNonEphemeralSandbox)
	prior.MaxCPUPerGPUSandbox = nullableFloat32(quota.MaxCpuPerGpuSandbox)
	prior.MaxMemoryPerGPUSandbox = nullableFloat32(quota.MaxMemoryPerGpuSandbox)
	prior.MaxDiskPerGPUSandbox = nullableFloat32(quota.MaxDiskPerGpuSandbox)

	return prior
}

func flattenSandboxRelationshipDataSource(ctx context.Context, sandbox *apiclient.Sandbox, prior sandboxRelationshipDataSourceModel) sandboxRelationshipDataSourceModel {
	item := flattenSandboxRelationshipItem(ctx, sandbox)

	prior.ID = item.ID
	prior.Name = item.Name
	prior.OrganizationID = item.OrganizationID
	prior.Snapshot = item.Snapshot
	prior.User = item.User
	prior.Env = item.Env
	prior.Labels = item.Labels
	prior.Public = item.Public
	prior.NetworkBlockAll = item.NetworkBlockAll
	prior.NetworkAllowList = item.NetworkAllowList
	prior.Target = item.Target
	prior.CPU = item.CPU
	prior.GPU = item.GPU
	prior.Memory = item.Memory
	prior.Disk = item.Disk
	prior.AutoStopInterval = item.AutoStopInterval
	prior.AutoArchiveInterval = item.AutoArchiveInterval
	prior.AutoDeleteInterval = item.AutoDeleteInterval
	prior.State = item.State
	prior.RunnerID = item.RunnerID
	prior.ToolboxProxyURL = item.ToolboxProxyURL
	prior.CreatedAt = item.CreatedAt
	prior.UpdatedAt = item.UpdatedAt
	prior.ErrorReason = item.ErrorReason

	return prior
}

func flattenSandboxRelationshipItems(ctx context.Context, sandboxes []apiclient.Sandbox) []sandboxRelationshipItemModel {
	items := make([]sandboxRelationshipItemModel, 0, len(sandboxes))
	for i := range sandboxes {
		items = append(items, flattenSandboxRelationshipItem(ctx, &sandboxes[i]))
	}
	return items
}

func flattenSandboxRelationshipItem(ctx context.Context, sandbox *apiclient.Sandbox) sandboxRelationshipItemModel {
	flattened := flattenSandboxDataSource(ctx, "", sandbox)
	return sandboxRelationshipItemModel{
		ID:                  flattened.ID,
		Name:                flattened.Name,
		OrganizationID:      flattened.OrganizationID,
		Snapshot:            flattened.Snapshot,
		User:                flattened.User,
		Env:                 flattened.Env,
		Labels:              flattened.Labels,
		Public:              flattened.Public,
		NetworkBlockAll:     flattened.NetworkBlockAll,
		NetworkAllowList:    flattened.NetworkAllowList,
		Target:              flattened.Target,
		CPU:                 flattened.CPU,
		GPU:                 flattened.GPU,
		Memory:              flattened.Memory,
		Disk:                flattened.Disk,
		AutoStopInterval:    flattened.AutoStopInterval,
		AutoArchiveInterval: flattened.AutoArchiveInterval,
		AutoDeleteInterval:  flattened.AutoDeleteInterval,
		State:               flattened.State,
		RunnerID:            flattened.RunnerID,
		ToolboxProxyURL:     flattened.ToolboxProxyURL,
		CreatedAt:           flattened.CreatedAt,
		UpdatedAt:           flattened.UpdatedAt,
		ErrorReason:         flattened.ErrorReason,
	}
}
