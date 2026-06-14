package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ConfigDataSource{}
var _ datasource.DataSource = &CurrentUserDataSource{}
var _ datasource.DataSource = &AccountProvidersDataSource{}
var _ datasource.DataSource = &OrganizationUsageDataSource{}
var _ datasource.DataSource = &OrganizationAuditLogsDataSource{}
var _ datasource.DataSource = &JobDataSource{}
var _ datasource.DataSource = &JobsDataSource{}
var _ datasource.DataSource = &DockerRegistryPushAccessDataSource{}
var _ datasource.DataSource = &ObjectStoragePushAccessDataSource{}

func NewConfigDataSource() datasource.DataSource {
	return &ConfigDataSource{}
}

func NewCurrentUserDataSource() datasource.DataSource {
	return &CurrentUserDataSource{}
}

func NewAccountProvidersDataSource() datasource.DataSource {
	return &AccountProvidersDataSource{}
}

func NewOrganizationUsageDataSource() datasource.DataSource {
	return &OrganizationUsageDataSource{}
}

func NewOrganizationAuditLogsDataSource() datasource.DataSource {
	return &OrganizationAuditLogsDataSource{}
}

func NewJobDataSource() datasource.DataSource {
	return &JobDataSource{}
}

func NewJobsDataSource() datasource.DataSource {
	return &JobsDataSource{}
}

func NewDockerRegistryPushAccessDataSource() datasource.DataSource {
	return &DockerRegistryPushAccessDataSource{}
}

func NewObjectStoragePushAccessDataSource() datasource.DataSource {
	return &ObjectStoragePushAccessDataSource{}
}

type ConfigDataSource struct {
	client *daytonaClient
}

type configDataSourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	Version                types.String  `tfsdk:"version"`
	LinkedAccountsEnabled  types.Bool    `tfsdk:"linked_accounts_enabled"`
	PylonAppID             types.String  `tfsdk:"pylon_app_id"`
	ProxyTemplateURL       types.String  `tfsdk:"proxy_template_url"`
	ProxyToolboxURL        types.String  `tfsdk:"proxy_toolbox_url"`
	DefaultSnapshot        types.String  `tfsdk:"default_snapshot"`
	DashboardURL           types.String  `tfsdk:"dashboard_url"`
	MaxAutoArchiveInterval types.Float64 `tfsdk:"max_auto_archive_interval"`
	MaintenanceMode        types.Bool    `tfsdk:"maintenance_mode"`
	Environment            types.String  `tfsdk:"environment"`
	BillingAPIURL          types.String  `tfsdk:"billing_api_url"`
	AnalyticsAPIURL        types.String  `tfsdk:"analytics_api_url"`
	SSHGatewayCommand      types.String  `tfsdk:"ssh_gateway_command"`
	SSHGatewayPublicKey    types.String  `tfsdk:"ssh_gateway_public_key"`
	PosthogJSON            types.String  `tfsdk:"posthog_json"`
	OIDCJSON               types.String  `tfsdk:"oidc_json"`
	AnnouncementsJSON      types.String  `tfsdk:"announcements_json"`
	RateLimitJSON          types.String  `tfsdk:"rate_limit_json"`
}

func (d *ConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (d *ConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Daytona server configuration visible to the configured credentials.",
		Attributes: map[string]schema.Attribute{
			"id":                        computedDataSourceStringAttribute("Data source identifier."),
			"version":                   computedDataSourceStringAttribute("Daytona server version."),
			"linked_accounts_enabled":   computedDataSourceBoolAttribute("Whether linked accounts are enabled."),
			"pylon_app_id":              computedDataSourceStringAttribute("Pylon app ID, when configured."),
			"proxy_template_url":        computedDataSourceStringAttribute("Proxy template URL."),
			"proxy_toolbox_url":         computedDataSourceStringAttribute("Proxy toolbox URL."),
			"default_snapshot":          computedDataSourceStringAttribute("Default snapshot image."),
			"dashboard_url":             computedDataSourceStringAttribute("Daytona dashboard URL."),
			"max_auto_archive_interval": computedDataSourceFloat64Attribute("Maximum auto-archive interval."),
			"maintenance_mode":          computedDataSourceBoolAttribute("Whether Daytona is in maintenance mode."),
			"environment":               computedDataSourceStringAttribute("Daytona environment name."),
			"billing_api_url":           computedDataSourceStringAttribute("Billing API URL, when configured."),
			"analytics_api_url":         computedDataSourceStringAttribute("Analytics API URL, when configured."),
			"ssh_gateway_command":       computedDataSourceStringAttribute("SSH gateway command, when configured."),
			"ssh_gateway_public_key":    computedDataSourceStringAttribute("SSH gateway public key, when configured."),
			"posthog_json":              computedDataSourceStringAttribute("PostHog configuration as JSON."),
			"oidc_json":                 computedDataSourceStringAttribute("OIDC configuration as JSON."),
			"announcements_json":        computedDataSourceStringAttribute("Announcements configuration as JSON."),
			"rate_limit_json":           computedDataSourceStringAttribute("Rate-limit configuration as JSON."),
		},
	}
}

func (d *ConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *ConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	config, httpResp, err := d.client.api.ConfigAPI.ConfigControllerGetConfig(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona configuration", "read configuration", httpResp, err)
		return
	}

	data := configDataSourceModel{
		ID:                     types.StringValue("config"),
		Version:                types.StringValue(config.Version),
		LinkedAccountsEnabled:  types.BoolValue(config.LinkedAccountsEnabled),
		PylonAppID:             pointerStringValue(config.PylonAppId),
		ProxyTemplateURL:       types.StringValue(config.ProxyTemplateUrl),
		ProxyToolboxURL:        types.StringValue(config.ProxyToolboxUrl),
		DefaultSnapshot:        types.StringValue(config.DefaultSnapshot),
		DashboardURL:           types.StringValue(config.DashboardUrl),
		MaxAutoArchiveInterval: types.Float64Value(float64(config.MaxAutoArchiveInterval)),
		MaintenanceMode:        types.BoolValue(config.MaintananceMode),
		Environment:            types.StringValue(config.Environment),
		BillingAPIURL:          pointerStringValue(config.BillingApiUrl),
		AnalyticsAPIURL:        pointerStringValue(config.AnalyticsApiUrl),
		SSHGatewayCommand:      pointerStringValue(config.SshGatewayCommand),
		SSHGatewayPublicKey:    pointerStringValue(config.SshGatewayPublicKey),
		PosthogJSON:            jsonStringValue(config.Posthog),
		OIDCJSON:               jsonStringValue(config.Oidc),
		AnnouncementsJSON:      jsonStringValue(config.Announcements),
		RateLimitJSON:          jsonStringValue(config.RateLimit),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type CurrentUserDataSource struct {
	client *daytonaClient
}

type currentUserDataSourceModel struct {
	ID         types.String                `tfsdk:"id"`
	Name       types.String                `tfsdk:"name"`
	Email      types.String                `tfsdk:"email"`
	PublicKeys []currentUserPublicKeyModel `tfsdk:"public_keys"`
	CreatedAt  types.String                `tfsdk:"created_at"`
}

type currentUserPublicKeyModel struct {
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (d *CurrentUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user"
}

func (d *CurrentUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Daytona user associated with the configured credentials.",
		Attributes: map[string]schema.Attribute{
			"id":         computedDataSourceStringAttribute("User ID."),
			"name":       computedDataSourceStringAttribute("User display name."),
			"email":      computedDataSourceStringAttribute("User email address."),
			"created_at": computedDataSourceStringAttribute("User creation timestamp."),
			"public_keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "User public SSH keys.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": computedDataSourceStringAttribute("Public key name."),
						"key":  computedDataSourceStringAttribute("Public key value."),
					},
				},
			},
		},
	}
}

func (d *CurrentUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *CurrentUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	user, httpResp, err := d.client.api.UsersAPI.GetAuthenticatedUser(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona current user", "read current user", httpResp, err)
		return
	}

	publicKeys := make([]currentUserPublicKeyModel, 0, len(user.PublicKeys))
	for _, publicKey := range user.PublicKeys {
		publicKeys = append(publicKeys, currentUserPublicKeyModel{
			Name: types.StringValue(publicKey.Name),
			Key:  types.StringValue(publicKey.Key),
		})
	}

	data := currentUserDataSourceModel{
		ID:         types.StringValue(user.Id),
		Name:       types.StringValue(user.Name),
		Email:      types.StringValue(user.Email),
		PublicKeys: publicKeys,
		CreatedAt:  terraformTimeString(user.CreatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type AccountProvidersDataSource struct {
	client *daytonaClient
}

type accountProvidersDataSourceModel struct {
	ID    types.String           `tfsdk:"id"`
	Items []accountProviderModel `tfsdk:"items"`
}

type accountProviderModel struct {
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
}

func (d *AccountProvidersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_providers"
}

func (d *AccountProvidersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists external account providers available to Daytona users.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Available account providers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         computedDataSourceStringAttribute("Provider name."),
						"display_name": computedDataSourceStringAttribute("Provider display name."),
					},
				},
			},
		},
	}
}

func (d *AccountProvidersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AccountProvidersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	providers, httpResp, err := d.client.api.UsersAPI.GetAvailableAccountProviders(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona account providers", "list account providers", httpResp, err)
		return
	}

	items := make([]accountProviderModel, 0, len(providers))
	for _, provider := range providers {
		items = append(items, accountProviderModel{
			Name:        types.StringValue(provider.Name),
			DisplayName: types.StringValue(provider.DisplayName),
		})
	}

	data := accountProvidersDataSourceModel{
		ID:    types.StringValue("account_providers"),
		Items: items,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationUsageDataSource struct {
	client *daytonaClient
}

type organizationUsageDataSourceModel struct {
	ID                   types.String       `tfsdk:"id"`
	OrganizationID       types.String       `tfsdk:"organization_id"`
	TotalSnapshotQuota   types.Float64      `tfsdk:"total_snapshot_quota"`
	CurrentSnapshotUsage types.Float64      `tfsdk:"current_snapshot_usage"`
	TotalVolumeQuota     types.Float64      `tfsdk:"total_volume_quota"`
	CurrentVolumeUsage   types.Float64      `tfsdk:"current_volume_usage"`
	RegionUsage          []regionUsageModel `tfsdk:"region_usage"`
}

type regionUsageModel struct {
	RegionID                      types.String  `tfsdk:"region_id"`
	SandboxClass                  types.String  `tfsdk:"sandbox_class"`
	TotalCPUQuota                 types.Float64 `tfsdk:"total_cpu_quota"`
	CurrentCPUUsage               types.Float64 `tfsdk:"current_cpu_usage"`
	TotalMemoryQuota              types.Float64 `tfsdk:"total_memory_quota"`
	CurrentMemoryUsage            types.Float64 `tfsdk:"current_memory_usage"`
	TotalDiskQuota                types.Float64 `tfsdk:"total_disk_quota"`
	CurrentDiskUsage              types.Float64 `tfsdk:"current_disk_usage"`
	TotalGPUQuota                 types.Float64 `tfsdk:"total_gpu_quota"`
	CurrentGPUUsage               types.Float64 `tfsdk:"current_gpu_usage"`
	AllowedGPUTypes               types.List    `tfsdk:"allowed_gpu_types"`
	MaxCPUPerSandbox              types.Float64 `tfsdk:"max_cpu_per_sandbox"`
	MaxMemoryPerSandbox           types.Float64 `tfsdk:"max_memory_per_sandbox"`
	MaxDiskPerSandbox             types.Float64 `tfsdk:"max_disk_per_sandbox"`
	MaxDiskPerNonEphemeralSandbox types.Float64 `tfsdk:"max_disk_per_non_ephemeral_sandbox"`
	MaxCPUPerGPUSandbox           types.Float64 `tfsdk:"max_cpu_per_gpu_sandbox"`
	MaxMemoryPerGPUSandbox        types.Float64 `tfsdk:"max_memory_per_gpu_sandbox"`
	MaxDiskPerGPUSandbox          types.Float64 `tfsdk:"max_disk_per_gpu_sandbox"`
}

func (d *OrganizationUsageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_usage"
}

func (d *OrganizationUsageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads quota and current usage for a Daytona organization.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID to read.",
			},
			"total_snapshot_quota":   computedDataSourceFloat64Attribute("Total snapshot quota."),
			"current_snapshot_usage": computedDataSourceFloat64Attribute("Current snapshot usage."),
			"total_volume_quota":     computedDataSourceFloat64Attribute("Total volume quota."),
			"current_volume_usage":   computedDataSourceFloat64Attribute("Current volume usage."),
			"region_usage": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Usage and quota broken down by region and sandbox class.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region_id":                          computedDataSourceStringAttribute("Region ID."),
						"sandbox_class":                      computedDataSourceStringAttribute("Sandbox class."),
						"total_cpu_quota":                    computedDataSourceFloat64Attribute("Total CPU quota."),
						"current_cpu_usage":                  computedDataSourceFloat64Attribute("Current CPU usage."),
						"total_memory_quota":                 computedDataSourceFloat64Attribute("Total memory quota."),
						"current_memory_usage":               computedDataSourceFloat64Attribute("Current memory usage."),
						"total_disk_quota":                   computedDataSourceFloat64Attribute("Total disk quota."),
						"current_disk_usage":                 computedDataSourceFloat64Attribute("Current disk usage."),
						"total_gpu_quota":                    computedDataSourceFloat64Attribute("Total GPU quota."),
						"current_gpu_usage":                  computedDataSourceFloat64Attribute("Current GPU usage."),
						"allowed_gpu_types":                  computedDataSourceStringListAttribute("Allowed GPU types."),
						"max_cpu_per_sandbox":                computedDataSourceFloat64Attribute("Maximum CPU per sandbox."),
						"max_memory_per_sandbox":             computedDataSourceFloat64Attribute("Maximum memory per sandbox."),
						"max_disk_per_sandbox":               computedDataSourceFloat64Attribute("Maximum disk per sandbox."),
						"max_disk_per_non_ephemeral_sandbox": computedDataSourceFloat64Attribute("Maximum disk per non-ephemeral sandbox."),
						"max_cpu_per_gpu_sandbox":            computedDataSourceFloat64Attribute("Maximum CPU per GPU sandbox."),
						"max_memory_per_gpu_sandbox":         computedDataSourceFloat64Attribute("Maximum memory per GPU sandbox."),
						"max_disk_per_gpu_sandbox":           computedDataSourceFloat64Attribute("Maximum disk per GPU sandbox."),
					},
				},
			},
		},
	}
}

func (d *OrganizationUsageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationUsageDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	usage, httpResp, err := d.client.api.OrganizationsAPI.GetOrganizationUsageOverview(ctx, data.OrganizationID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization usage", "read organization usage", httpResp, err)
		return
	}

	data.ID = data.OrganizationID
	data.TotalSnapshotQuota = float64Value(usage.TotalSnapshotQuota)
	data.CurrentSnapshotUsage = float64Value(usage.CurrentSnapshotUsage)
	data.TotalVolumeQuota = float64Value(usage.TotalVolumeQuota)
	data.CurrentVolumeUsage = float64Value(usage.CurrentVolumeUsage)
	data.RegionUsage = regionUsageModels(ctx, usage.RegionUsage)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationAuditLogsDataSource struct {
	client *daytonaClient
}

type organizationAuditLogsDataSourceModel struct {
	ID             types.String    `tfsdk:"id"`
	OrganizationID types.String    `tfsdk:"organization_id"`
	Page           types.Int64     `tfsdk:"page"`
	Limit          types.Int64     `tfsdk:"limit"`
	From           types.String    `tfsdk:"from"`
	To             types.String    `tfsdk:"to"`
	Cursor         types.String    `tfsdk:"cursor"`
	Total          types.Int64     `tfsdk:"total"`
	TotalPages     types.Int64     `tfsdk:"total_pages"`
	NextToken      types.String    `tfsdk:"next_token"`
	Items          []auditLogModel `tfsdk:"items"`
}

type auditLogModel struct {
	ID                types.String  `tfsdk:"id"`
	ActorID           types.String  `tfsdk:"actor_id"`
	ActorEmail        types.String  `tfsdk:"actor_email"`
	ActorAPIKeyPrefix types.String  `tfsdk:"actor_api_key_prefix"`
	ActorAPIKeySuffix types.String  `tfsdk:"actor_api_key_suffix"`
	OrganizationID    types.String  `tfsdk:"organization_id"`
	Action            types.String  `tfsdk:"action"`
	TargetType        types.String  `tfsdk:"target_type"`
	TargetID          types.String  `tfsdk:"target_id"`
	StatusCode        types.Float64 `tfsdk:"status_code"`
	ErrorMessage      types.String  `tfsdk:"error_message"`
	IPAddress         types.String  `tfsdk:"ip_address"`
	UserAgent         types.String  `tfsdk:"user_agent"`
	Source            types.String  `tfsdk:"source"`
	MetadataJSON      types.String  `tfsdk:"metadata_json"`
	CreatedAt         types.String  `tfsdk:"created_at"`
}

func (d *OrganizationAuditLogsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_audit_logs"
}

func (d *OrganizationAuditLogsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads paginated Daytona audit logs for an organization.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID to read audit logs from.",
			},
			"page": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Page number to request. Defaults to 1.",
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of audit log entries to request. Defaults to 100.",
			},
			"from": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Start timestamp in RFC3339 format.",
			},
			"to": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "End timestamp in RFC3339 format.",
			},
			"cursor": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Cursor token to request the next page of audit logs.",
			},
			"total":       computedDataSourceInt64Attribute("Total matching audit log entries."),
			"total_pages": computedDataSourceInt64Attribute("Total result pages."),
			"next_token":  computedDataSourceStringAttribute("Cursor token for the next page, when available."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Audit log entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                   computedDataSourceStringAttribute("Audit log ID."),
						"actor_id":             computedDataSourceStringAttribute("Actor user ID."),
						"actor_email":          computedDataSourceStringAttribute("Actor email address."),
						"actor_api_key_prefix": computedDataSourceStringAttribute("Actor API key prefix, when available."),
						"actor_api_key_suffix": computedDataSourceStringAttribute("Actor API key suffix, when available."),
						"organization_id":      computedDataSourceStringAttribute("Organization ID."),
						"action":               computedDataSourceStringAttribute("Audit action."),
						"target_type":          computedDataSourceStringAttribute("Target object type."),
						"target_id":            computedDataSourceStringAttribute("Target object ID."),
						"status_code":          computedDataSourceFloat64Attribute("Resulting status code."),
						"error_message":        computedDataSourceStringAttribute("Error message, when available."),
						"ip_address":           computedDataSourceStringAttribute("Actor IP address."),
						"user_agent":           computedDataSourceStringAttribute("Actor user agent."),
						"source":               computedDataSourceStringAttribute("Audit source."),
						"metadata_json":        computedDataSourceStringAttribute("Audit metadata as JSON."),
						"created_at":           computedDataSourceStringAttribute("Audit log creation timestamp."),
					},
				},
			},
		},
	}
}

func (d *OrganizationAuditLogsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationAuditLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationAuditLogsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.AuditAPI.GetOrganizationAuditLogs(ctx, data.OrganizationID.ValueString())
	if !data.Page.IsNull() {
		request = request.Page(float32(data.Page.ValueInt64()))
	} else {
		data.Page = types.Int64Value(1)
	}
	if !data.Limit.IsNull() {
		request = request.Limit(float32(data.Limit.ValueInt64()))
	} else {
		data.Limit = types.Int64Value(100)
	}
	if !data.From.IsNull() {
		from, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "from", data.From.ValueString())
		if !ok {
			return
		}
		request = request.From(from)
	}
	if !data.To.IsNull() {
		to, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "to", data.To.ValueString())
		if !ok {
			return
		}
		request = request.To(to)
	}
	if !data.Cursor.IsNull() {
		request = request.NextToken(data.Cursor.ValueString())
	}

	result, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization audit logs", "read organization audit logs", httpResp, err)
		return
	}

	data.ID = types.StringValue(data.OrganizationID.ValueString() + ":audit_logs")
	data.Total = types.Int64Value(int64(result.Total))
	data.TotalPages = types.Int64Value(int64(result.TotalPages))
	data.NextToken = pointerStringValue(result.NextToken)
	data.Items = auditLogModels(result.Items)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type JobDataSource struct {
	client *daytonaClient
}

type jobDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Status       types.String `tfsdk:"status"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Payload      types.String `tfsdk:"payload"`
	TraceContext types.String `tfsdk:"trace_context_json"`
	ErrorMessage types.String `tfsdk:"error_message"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (d *JobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (d *JobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona background job by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona job ID.",
			},
			"type":               computedDataSourceStringAttribute("Job type."),
			"status":             computedDataSourceStringAttribute("Job status."),
			"resource_type":      computedDataSourceStringAttribute("Resource type associated with the job."),
			"resource_id":        computedDataSourceStringAttribute("Resource ID associated with the job."),
			"payload":            computedDataSourceStringAttribute("Job payload."),
			"trace_context_json": computedDataSourceStringAttribute("Job trace context as JSON."),
			"error_message":      computedDataSourceStringAttribute("Error message, when available."),
			"created_at":         computedDataSourceStringAttribute("Job creation timestamp."),
			"updated_at":         computedDataSourceStringAttribute("Job update timestamp."),
		},
	}
}

func (d *JobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *JobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data jobDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, httpResp, err := d.client.api.JobsAPI.GetJob(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona job", "read job", httpResp, err)
		return
	}

	data = jobModel(*job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type JobsDataSource struct {
	client *daytonaClient
}

type jobsDataSourceModel struct {
	ID         types.String         `tfsdk:"id"`
	Page       types.Int64          `tfsdk:"page"`
	Limit      types.Int64          `tfsdk:"limit"`
	Offset     types.Int64          `tfsdk:"offset"`
	Status     types.String         `tfsdk:"status"`
	Total      types.Int64          `tfsdk:"total"`
	TotalPages types.Int64          `tfsdk:"total_pages"`
	Items      []jobDataSourceModel `tfsdk:"items"`
}

func (d *JobsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jobs"
}

func (d *JobsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads paginated Daytona background jobs visible to the configured credentials.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"page": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Page number to request. Defaults to 1.",
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum number of jobs to return. Defaults to 100.",
			},
			"offset": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of jobs to skip. Defaults to 0.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional job status filter, such as `PENDING`, `IN_PROGRESS`, `COMPLETED`, or `FAILED`.",
			},
			"total":       computedDataSourceInt64Attribute("Total matching jobs."),
			"total_pages": computedDataSourceInt64Attribute("Total result pages."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned jobs.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: jobNestedAttributes(),
				},
			},
		},
	}
}

func (d *JobsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *JobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data jobsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.JobsAPI.ListJobs(ctx)
	if !data.Page.IsNull() {
		request = request.Page(float32(data.Page.ValueInt64()))
	} else {
		data.Page = types.Int64Value(1)
	}
	if !data.Limit.IsNull() {
		request = request.Limit(float32(data.Limit.ValueInt64()))
	} else {
		data.Limit = types.Int64Value(100)
	}
	if !data.Offset.IsNull() {
		request = request.Offset(float32(data.Offset.ValueInt64()))
	} else {
		data.Offset = types.Int64Value(0)
	}
	if !data.Status.IsNull() {
		statusValue := apiclient.JobStatus(strings.TrimSpace(data.Status.ValueString()))
		if !statusValue.IsValid() || statusValue == apiclient.JOBSTATUS_UNKNOWN_DEFAULT_OPEN_API {
			resp.Diagnostics.AddAttributeError(
				path.Root("status"),
				"Invalid Daytona job status",
				fmt.Sprintf("Status must be one of %s.", strings.Join(jobStatusValues(), ", ")),
			)
			return
		}
		request = request.Status(statusValue)
	}

	result, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona jobs", "list jobs", httpResp, err)
		return
	}

	items := make([]jobDataSourceModel, 0, len(result.Items))
	for _, job := range result.Items {
		items = append(items, jobModel(job))
	}

	data.ID = types.StringValue("jobs")
	data.Total = types.Int64Value(int64(result.Total))
	data.TotalPages = types.Int64Value(int64(result.TotalPages))
	data.Items = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type ObjectStoragePushAccessDataSource struct {
	client *daytonaClient
}

type DockerRegistryPushAccessDataSource struct {
	client *daytonaClient
}

type dockerRegistryPushAccessDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	RegionID       types.String `tfsdk:"region_id"`
	Username       types.String `tfsdk:"username"`
	Secret         types.String `tfsdk:"secret"`
	RegistryURL    types.String `tfsdk:"registry_url"`
	RegistryID     types.String `tfsdk:"registry_id"`
	Project        types.String `tfsdk:"project"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
}

func (d *DockerRegistryPushAccessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_registry_push_access"
}

func (d *DockerRegistryPushAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads temporary Daytona Docker registry credentials for pushing snapshots.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona organization ID. Defaults to the provider-level organization when configured.",
			},
			"region_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona region ID where the pushed snapshot will be available. Defaults to the organization default region.",
			},
			"username":     sensitiveComputedDataSourceStringAttribute("Temporary registry username."),
			"secret":       sensitiveComputedDataSourceStringAttribute("Temporary registry secret."),
			"registry_url": computedDataSourceStringAttribute("Registry URL."),
			"registry_id":  computedDataSourceStringAttribute("Registry ID."),
			"project":      computedDataSourceStringAttribute("Registry project ID."),
			"expires_at":   computedDataSourceStringAttribute("Credential expiration timestamp."),
		},
	}
}

func (d *DockerRegistryPushAccessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *DockerRegistryPushAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dockerRegistryPushAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.DockerRegistryAPI.GetTransientPushAccess(ctx)
	if !data.OrganizationID.IsNull() {
		request = request.XDaytonaOrganizationID(data.OrganizationID.ValueString())
	}
	if !data.RegionID.IsNull() {
		request = request.RegionId(data.RegionID.ValueString())
	}

	access, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona Docker registry push access", "read Docker registry push access", httpResp, err)
		return
	}

	data.ID = types.StringValue(access.RegistryId + ":" + access.Project)
	data.Username = types.StringValue(access.Username)
	data.Secret = types.StringValue(access.Secret)
	data.RegistryURL = types.StringValue(access.RegistryUrl)
	data.RegistryID = types.StringValue(access.RegistryId)
	data.Project = types.StringValue(access.Project)
	data.ExpiresAt = types.StringValue(access.ExpiresAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type objectStoragePushAccessDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	AccessKey      types.String `tfsdk:"access_key"`
	Secret         types.String `tfsdk:"secret"`
	SessionToken   types.String `tfsdk:"session_token"`
	StorageURL     types.String `tfsdk:"storage_url"`
	Bucket         types.String `tfsdk:"bucket"`
}

func (d *ObjectStoragePushAccessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_storage_push_access"
}

func (d *ObjectStoragePushAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads temporary Daytona object-storage credentials for pushing objects.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Daytona organization ID. Defaults to the provider-level organization when configured.",
			},
			"access_key":    sensitiveComputedDataSourceStringAttribute("Temporary storage access key."),
			"secret":        sensitiveComputedDataSourceStringAttribute("Temporary storage secret key."),
			"session_token": sensitiveComputedDataSourceStringAttribute("Temporary storage session token."),
			"storage_url":   computedDataSourceStringAttribute("Storage endpoint URL."),
			"bucket":        computedDataSourceStringAttribute("Storage bucket name."),
		},
	}
}

func (d *ObjectStoragePushAccessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *ObjectStoragePushAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data objectStoragePushAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.ObjectStorageAPI.GetPushAccess(ctx)
	if !data.OrganizationID.IsNull() {
		request = request.XDaytonaOrganizationID(data.OrganizationID.ValueString())
	}

	access, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona object-storage push access", "read object-storage push access", httpResp, err)
		return
	}

	data.ID = types.StringValue(access.OrganizationId + ":" + access.Bucket)
	data.OrganizationID = types.StringValue(access.OrganizationId)
	data.AccessKey = types.StringValue(access.AccessKey)
	data.Secret = types.StringValue(access.Secret)
	data.SessionToken = types.StringValue(access.SessionToken)
	data.StorageURL = types.StringValue(access.StorageUrl)
	data.Bucket = types.StringValue(access.Bucket)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func regionUsageModels(ctx context.Context, usages []apiclient.RegionUsageOverview) []regionUsageModel {
	items := make([]regionUsageModel, 0, len(usages))
	for _, usage := range usages {
		allowedGPUTypes := make([]string, 0, len(usage.AllowedGpuTypes))
		for _, gpuType := range usage.AllowedGpuTypes {
			allowedGPUTypes = append(allowedGPUTypes, string(gpuType))
		}

		item := regionUsageModel{
			RegionID:                      types.StringValue(usage.RegionId),
			SandboxClass:                  types.StringValue(string(usage.SandboxClass)),
			TotalCPUQuota:                 float64Value(usage.TotalCpuQuota),
			CurrentCPUUsage:               float64Value(usage.CurrentCpuUsage),
			TotalMemoryQuota:              float64Value(usage.TotalMemoryQuota),
			CurrentMemoryUsage:            float64Value(usage.CurrentMemoryUsage),
			TotalDiskQuota:                float64Value(usage.TotalDiskQuota),
			CurrentDiskUsage:              float64Value(usage.CurrentDiskUsage),
			TotalGPUQuota:                 float64Value(usage.TotalGpuQuota),
			CurrentGPUUsage:               float64Value(usage.CurrentGpuUsage),
			AllowedGPUTypes:               listStringValue(ctx, allowedGPUTypes),
			MaxCPUPerSandbox:              nullableFloat32Pointer(usage.MaxCpuPerSandbox.Get()),
			MaxMemoryPerSandbox:           nullableFloat32Pointer(usage.MaxMemoryPerSandbox.Get()),
			MaxDiskPerSandbox:             nullableFloat32Pointer(usage.MaxDiskPerSandbox.Get()),
			MaxDiskPerNonEphemeralSandbox: nullableFloat32Pointer(usage.MaxDiskPerNonEphemeralSandbox.Get()),
			MaxCPUPerGPUSandbox:           nullableFloat32Pointer(usage.MaxCpuPerGpuSandbox.Get()),
			MaxMemoryPerGPUSandbox:        nullableFloat32Pointer(usage.MaxMemoryPerGpuSandbox.Get()),
			MaxDiskPerGPUSandbox:          nullableFloat32Pointer(usage.MaxDiskPerGpuSandbox.Get()),
		}
		items = append(items, item)
	}
	return items
}

func auditLogModels(logs []apiclient.AuditLog) []auditLogModel {
	items := make([]auditLogModel, 0, len(logs))
	for _, log := range logs {
		items = append(items, auditLogModel{
			ID:                types.StringValue(log.Id),
			ActorID:           types.StringValue(log.ActorId),
			ActorEmail:        types.StringValue(log.ActorEmail),
			ActorAPIKeyPrefix: pointerStringValue(log.ActorApiKeyPrefix),
			ActorAPIKeySuffix: pointerStringValue(log.ActorApiKeySuffix),
			OrganizationID:    pointerStringValue(log.OrganizationId),
			Action:            types.StringValue(log.Action),
			TargetType:        pointerStringValue(log.TargetType),
			TargetID:          pointerStringValue(log.TargetId),
			StatusCode:        nullableFloat32Pointer(log.StatusCode),
			ErrorMessage:      pointerStringValue(log.ErrorMessage),
			IPAddress:         pointerStringValue(log.IpAddress),
			UserAgent:         pointerStringValue(log.UserAgent),
			Source:            pointerStringValue(log.Source),
			MetadataJSON:      jsonStringValue(log.Metadata),
			CreatedAt:         terraformTimeString(log.CreatedAt),
		})
	}
	return items
}

func jobModel(job apiclient.Job) jobDataSourceModel {
	return jobDataSourceModel{
		ID:           types.StringValue(job.Id),
		Type:         types.StringValue(string(job.Type)),
		Status:       types.StringValue(string(job.Status)),
		ResourceType: types.StringValue(job.ResourceType),
		ResourceID:   types.StringValue(job.ResourceId),
		Payload:      pointerStringValue(job.Payload),
		TraceContext: jsonStringValue(job.TraceContext),
		ErrorMessage: pointerStringValue(job.ErrorMessage),
		CreatedAt:    types.StringValue(job.CreatedAt),
		UpdatedAt:    pointerStringValue(job.UpdatedAt),
	}
}

func jobNestedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                 computedDataSourceStringAttribute("Job ID."),
		"type":               computedDataSourceStringAttribute("Job type."),
		"status":             computedDataSourceStringAttribute("Job status."),
		"resource_type":      computedDataSourceStringAttribute("Resource type associated with the job."),
		"resource_id":        computedDataSourceStringAttribute("Resource ID associated with the job."),
		"payload":            computedDataSourceStringAttribute("Job payload."),
		"trace_context_json": computedDataSourceStringAttribute("Job trace context as JSON."),
		"error_message":      computedDataSourceStringAttribute("Error message, when available."),
		"created_at":         computedDataSourceStringAttribute("Job creation timestamp."),
		"updated_at":         computedDataSourceStringAttribute("Job update timestamp."),
	}
}

func parseRFC3339DataSourceTime(diags *diag.Diagnostics, name string, value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		diags.AddAttributeError(
			path.Root(name),
			"Invalid RFC3339 timestamp",
			fmt.Sprintf("The %q value must be an RFC3339 timestamp, such as 2026-01-02T03:04:05Z.", name),
		)
		return time.Time{}, false
	}
	return parsed, true
}

func jobStatusValues() []string {
	values := make([]string, 0, len(apiclient.AllowedJobStatusEnumValues)-1)
	for _, value := range apiclient.AllowedJobStatusEnumValues {
		if value == apiclient.JOBSTATUS_UNKNOWN_DEFAULT_OPEN_API {
			continue
		}
		values = append(values, string(value))
	}
	return values
}

func jsonStringValue(value any) types.String {
	if value == nil {
		return types.StringNull()
	}

	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}

func nullableFloat32Pointer(value *float32) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return float64Value(*value)
}

func float64Value(value float32) types.Float64 {
	return types.Float64Value(float64(value))
}

func computedDataSourceFloat64Attribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func computedDataSourceInt64Attribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func computedDataSourceStringListAttribute(description string) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType:         types.StringType,
		Computed:            true,
		MarkdownDescription: description,
	}
}
