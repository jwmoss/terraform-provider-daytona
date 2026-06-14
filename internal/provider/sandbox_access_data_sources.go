package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SandboxSSHAccessDataSource{}
var _ datasource.DataSource = &SandboxBuildLogsURLDataSource{}
var _ datasource.DataSource = &SandboxPortPreviewURLDataSource{}
var _ datasource.DataSource = &SandboxSignedPortPreviewURLDataSource{}
var _ datasource.DataSource = &SandboxPublicStatusDataSource{}
var _ datasource.DataSource = &SandboxAuthTokenValidationDataSource{}
var _ datasource.DataSource = &SandboxAccessDataSource{}
var _ datasource.DataSource = &SandboxIDFromSignedPreviewTokenDataSource{}
var _ datasource.DataSource = &SandboxSSHAccessValidationDataSource{}
var _ datasource.DataSource = &SnapshotBuildLogsURLDataSource{}

func NewSandboxSSHAccessDataSource() datasource.DataSource {
	return &SandboxSSHAccessDataSource{}
}

func NewSandboxBuildLogsURLDataSource() datasource.DataSource {
	return &SandboxBuildLogsURLDataSource{}
}

func NewSandboxPortPreviewURLDataSource() datasource.DataSource {
	return &SandboxPortPreviewURLDataSource{}
}

func NewSandboxSignedPortPreviewURLDataSource() datasource.DataSource {
	return &SandboxSignedPortPreviewURLDataSource{}
}

func NewSandboxPublicStatusDataSource() datasource.DataSource {
	return &SandboxPublicStatusDataSource{}
}

func NewSandboxAuthTokenValidationDataSource() datasource.DataSource {
	return &SandboxAuthTokenValidationDataSource{}
}

func NewSandboxAccessDataSource() datasource.DataSource {
	return &SandboxAccessDataSource{}
}

func NewSandboxIDFromSignedPreviewTokenDataSource() datasource.DataSource {
	return &SandboxIDFromSignedPreviewTokenDataSource{}
}

func NewSandboxSSHAccessValidationDataSource() datasource.DataSource {
	return &SandboxSSHAccessValidationDataSource{}
}

func NewSnapshotBuildLogsURLDataSource() datasource.DataSource {
	return &SnapshotBuildLogsURLDataSource{}
}

type SandboxSSHAccessDataSource struct {
	client *daytonaClient
}

type sandboxSSHAccessDataSourceModel struct {
	ID               types.String  `tfsdk:"id"`
	SandboxIDOrName  types.String  `tfsdk:"sandbox_id_or_name"`
	OrganizationID   types.String  `tfsdk:"organization_id"`
	ExpiresInMinutes types.Float64 `tfsdk:"expires_in_minutes"`
	SandboxID        types.String  `tfsdk:"sandbox_id"`
	Token            types.String  `tfsdk:"token"`
	SSHCommand       types.String  `tfsdk:"ssh_command"`
	ExpiresAt        types.String  `tfsdk:"expires_at"`
	CreatedAt        types.String  `tfsdk:"created_at"`
	UpdatedAt        types.String  `tfsdk:"updated_at"`
}

func (d *SandboxSSHAccessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_ssh_access"
}

func (d *SandboxSSHAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates temporary SSH access for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                 computedDataSourceStringAttribute("SSH access ID."),
			"sandbox_id_or_name": requiredDataSourceStringAttribute("Sandbox ID or name."),
			"organization_id":    optionalOrganizationIDDataSourceStringAttribute(),
			"expires_in_minutes": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "Expiration time in minutes. Daytona defaults to 60 minutes when omitted.",
			},
			"sandbox_id":  computedDataSourceStringAttribute("Sandbox ID."),
			"token":       sensitiveComputedDataSourceStringAttribute("Temporary SSH access token."),
			"ssh_command": sensitiveComputedDataSourceStringAttribute("SSH command containing temporary access material."),
			"expires_at":  computedDataSourceStringAttribute("SSH access expiration timestamp."),
			"created_at":  computedDataSourceStringAttribute("SSH access creation timestamp."),
			"updated_at":  computedDataSourceStringAttribute("SSH access update timestamp."),
		},
	}
}

func (d *SandboxSSHAccessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxSSHAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxSSHAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.CreateSshAccess(ctx, data.SandboxIDOrName.ValueString())
	if terraformFloat64Configured(data.ExpiresInMinutes) {
		request = request.ExpiresInMinutes(float32(data.ExpiresInMinutes.ValueFloat64()))
	}
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	access, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona sandbox SSH access", "create sandbox SSH access", httpResp, err)
		return
	}
	if access == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona sandbox SSH access response",
			"Daytona returned a successful response without sandbox SSH access data.",
		)
		return
	}

	data.ID = types.StringValue(access.Id)
	data.SandboxID = types.StringValue(access.SandboxId)
	data.Token = types.StringValue(access.Token)
	data.SSHCommand = types.StringValue(access.SshCommand)
	data.ExpiresAt = terraformTimeString(access.ExpiresAt)
	data.CreatedAt = terraformTimeString(access.CreatedAt)
	data.UpdatedAt = terraformTimeString(access.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxBuildLogsURLDataSource struct {
	client *daytonaClient
}

type sandboxBuildLogsURLDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	URL             types.String `tfsdk:"url"`
}

func (d *SandboxBuildLogsURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_build_logs_url"
}

func (d *SandboxBuildLogsURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a build logs URL for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                 computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id_or_name": requiredDataSourceStringAttribute("Sandbox ID or name."),
			"organization_id":    optionalOrganizationIDDataSourceStringAttribute(),
			"url":                sensitiveComputedDataSourceStringAttribute("Sandbox build logs URL."),
		},
	}
}

func (d *SandboxBuildLogsURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxBuildLogsURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxBuildLogsURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetBuildLogsUrl(ctx, data.SandboxIDOrName.ValueString())
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	url, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox build logs URL", "read sandbox build logs URL", httpResp, err)
		return
	}
	if url == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona sandbox build logs URL response",
			"Daytona returned a successful response without sandbox build logs URL data.",
		)
		return
	}

	data.ID = types.StringValue(data.SandboxIDOrName.ValueString() + ":build_logs_url")
	data.URL = types.StringValue(url.Url)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxPortPreviewURLDataSource struct {
	client *daytonaClient
}

type sandboxPortPreviewURLDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	Port            types.Int64  `tfsdk:"port"`
	SandboxID       types.String `tfsdk:"sandbox_id"`
	URL             types.String `tfsdk:"url"`
	Token           types.String `tfsdk:"token"`
}

func (d *SandboxPortPreviewURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_port_preview_url"
}

func (d *SandboxPortPreviewURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a preview URL for a Daytona sandbox port.",
		Attributes: map[string]schema.Attribute{
			"id":                 computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id_or_name": requiredDataSourceStringAttribute("Sandbox ID or name."),
			"organization_id":    optionalOrganizationIDDataSourceStringAttribute(),
			"port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Sandbox port number.",
			},
			"sandbox_id": computedDataSourceStringAttribute("Sandbox ID."),
			"url":        sensitiveComputedDataSourceStringAttribute("Sandbox port preview URL."),
			"token":      sensitiveComputedDataSourceStringAttribute("Sandbox port preview access token."),
		},
	}
}

func (d *SandboxPortPreviewURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxPortPreviewURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxPortPreviewURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetPortPreviewUrl(ctx, data.SandboxIDOrName.ValueString(), float32(data.Port.ValueInt64()))
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	preview, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox port preview URL", "read sandbox port preview URL", httpResp, err)
		return
	}
	if preview == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona sandbox port preview URL response",
			"Daytona returned a successful response without sandbox port preview URL data.",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d:port_preview_url", data.SandboxIDOrName.ValueString(), data.Port.ValueInt64()))
	data.SandboxID = types.StringValue(preview.SandboxId)
	data.URL = types.StringValue(preview.Url)
	data.Token = types.StringValue(preview.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxSignedPortPreviewURLDataSource struct {
	client *daytonaClient
}

type sandboxSignedPortPreviewURLDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	SandboxIDOrName  types.String `tfsdk:"sandbox_id_or_name"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	Port             types.Int64  `tfsdk:"port"`
	ExpiresInSeconds types.Int64  `tfsdk:"expires_in_seconds"`
	SandboxID        types.String `tfsdk:"sandbox_id"`
	URL              types.String `tfsdk:"url"`
	Token            types.String `tfsdk:"token"`
}

func (d *SandboxSignedPortPreviewURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_signed_port_preview_url"
}

func (d *SandboxSignedPortPreviewURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a signed preview URL for a Daytona sandbox port.",
		Attributes: map[string]schema.Attribute{
			"id":                 computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id_or_name": requiredDataSourceStringAttribute("Sandbox ID or name."),
			"organization_id":    optionalOrganizationIDDataSourceStringAttribute(),
			"port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Sandbox port number.",
			},
			"expires_in_seconds": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Expiration time in seconds. Daytona defaults to 60 seconds when omitted.",
			},
			"sandbox_id": computedDataSourceStringAttribute("Sandbox ID."),
			"url":        sensitiveComputedDataSourceStringAttribute("Signed sandbox port preview URL."),
			"token":      sensitiveComputedDataSourceStringAttribute("Signed sandbox port preview access token."),
		},
	}
}

func (d *SandboxSignedPortPreviewURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxSignedPortPreviewURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxSignedPortPreviewURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSignedPortPreviewUrl(ctx, data.SandboxIDOrName.ValueString(), int32(data.Port.ValueInt64()))
	if expiresInSeconds := optionalInt32(data.ExpiresInSeconds); expiresInSeconds != nil {
		request = request.ExpiresInSeconds(*expiresInSeconds)
	}
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	preview, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox signed port preview URL", "read sandbox signed port preview URL", httpResp, err)
		return
	}
	if preview == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona sandbox signed port preview URL response",
			"Daytona returned a successful response without sandbox signed port preview URL data.",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d:signed_port_preview_url", data.SandboxIDOrName.ValueString(), data.Port.ValueInt64()))
	data.SandboxID = types.StringValue(preview.SandboxId)
	data.Port = types.Int64Value(int64(preview.Port))
	data.URL = types.StringValue(preview.Url)
	data.Token = types.StringValue(preview.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxPublicStatusDataSource struct {
	client *daytonaClient
}

type sandboxPublicStatusDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	SandboxID types.String `tfsdk:"sandbox_id"`
	Public    types.Bool   `tfsdk:"public"`
}

type sandboxPublicStatusConfigModel struct {
	SandboxID types.String `tfsdk:"sandbox_id"`
}

func (d *SandboxPublicStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_public_status"
}

func (d *SandboxPublicStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks whether a Daytona sandbox is public through the preview API.",
		Attributes: map[string]schema.Attribute{
			"id":         computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id": requiredDataSourceStringAttribute("Sandbox ID."),
			"public":     computedDataSourceBoolAttribute("Whether the sandbox is public."),
		},
	}
}

func (d *SandboxPublicStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxPublicStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxPublicStatusConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	public, httpResp, err := d.client.api.PreviewAPI.IsSandboxPublic(ctx, config.SandboxID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to check Daytona sandbox public status", "check sandbox public status", httpResp, err)
		return
	}

	data := sandboxPublicStatusDataSourceModel{
		ID:        types.StringValue(config.SandboxID.ValueString() + ":public_status"),
		SandboxID: config.SandboxID,
		Public:    types.BoolValue(public),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxAuthTokenValidationDataSource struct {
	client *daytonaClient
}

type sandboxAuthTokenValidationDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	SandboxID types.String `tfsdk:"sandbox_id"`
	AuthToken types.String `tfsdk:"auth_token"`
	Valid     types.Bool   `tfsdk:"valid"`
}

type sandboxAuthTokenValidationConfigModel struct {
	SandboxID types.String `tfsdk:"sandbox_id"`
	AuthToken types.String `tfsdk:"auth_token"`
}

func (d *SandboxAuthTokenValidationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_auth_token_validation"
}

func (d *SandboxAuthTokenValidationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Validates a Daytona sandbox auth token through the preview API.",
		Attributes: map[string]schema.Attribute{
			"id":         computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id": requiredDataSourceStringAttribute("Sandbox ID."),
			"auth_token": requiredSensitiveDataSourceStringAttribute("Sandbox auth token to validate."),
			"valid":      computedDataSourceBoolAttribute("Whether the sandbox auth token is valid."),
		},
	}
}

func (d *SandboxAuthTokenValidationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxAuthTokenValidationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxAuthTokenValidationConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authToken := strings.TrimSpace(config.AuthToken.ValueString())
	if authToken == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox auth token",
			"Configure auth_token with the Daytona sandbox auth token to validate.",
		)
		return
	}

	valid, httpResp, err := d.client.api.PreviewAPI.IsValidAuthToken(ctx, config.SandboxID.ValueString(), authToken).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to validate Daytona sandbox auth token", "validate sandbox auth token", httpResp, err)
		return
	}

	data := sandboxAuthTokenValidationDataSourceModel{
		ID:        types.StringValue(config.SandboxID.ValueString() + ":auth_token_validation"),
		SandboxID: config.SandboxID,
		AuthToken: config.AuthToken,
		Valid:     types.BoolValue(valid),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxAccessDataSource struct {
	client *daytonaClient
}

type sandboxAccessDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	SandboxID types.String `tfsdk:"sandbox_id"`
	HasAccess types.Bool   `tfsdk:"has_access"`
}

type sandboxAccessConfigModel struct {
	SandboxID types.String `tfsdk:"sandbox_id"`
}

func (d *SandboxAccessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_access"
}

func (d *SandboxAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks whether the authenticated Daytona user has access to a sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":         computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id": requiredDataSourceStringAttribute("Sandbox ID."),
			"has_access": computedDataSourceBoolAttribute("Whether the authenticated user has access to the sandbox."),
		},
	}
}

func (d *SandboxAccessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxAccessConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasAccess, httpResp, err := d.client.api.PreviewAPI.HasSandboxAccess(ctx, config.SandboxID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to check Daytona sandbox access", "check sandbox access", httpResp, err)
		return
	}

	data := sandboxAccessDataSourceModel{
		ID:        types.StringValue(config.SandboxID.ValueString() + ":access"),
		SandboxID: config.SandboxID,
		HasAccess: types.BoolValue(hasAccess),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxIDFromSignedPreviewTokenDataSource struct {
	client *daytonaClient
}

type sandboxIDFromSignedPreviewTokenDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	SignedPreviewToken types.String `tfsdk:"signed_preview_token"`
	Port               types.Int64  `tfsdk:"port"`
	SandboxID          types.String `tfsdk:"sandbox_id"`
}

type sandboxIDFromSignedPreviewTokenConfigModel struct {
	SignedPreviewToken types.String `tfsdk:"signed_preview_token"`
	Port               types.Int64  `tfsdk:"port"`
}

func (d *SandboxIDFromSignedPreviewTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_id_from_signed_preview_token"
}

func (d *SandboxIDFromSignedPreviewTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Daytona sandbox ID encoded in a signed preview URL token.",
		Attributes: map[string]schema.Attribute{
			"id":                   computedDataSourceStringAttribute("Data source identifier."),
			"signed_preview_token": requiredSensitiveDataSourceStringAttribute("Signed preview URL token."),
			"port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Sandbox port number from the signed preview URL.",
			},
			"sandbox_id": computedDataSourceStringAttribute("Sandbox ID from the signed preview URL token."),
		},
	}
}

func (d *SandboxIDFromSignedPreviewTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxIDFromSignedPreviewTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxIDFromSignedPreviewTokenConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	signedPreviewToken := strings.TrimSpace(config.SignedPreviewToken.ValueString())
	if signedPreviewToken == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona signed preview URL token",
			"Configure signed_preview_token with the signed preview URL token to inspect.",
		)
		return
	}

	port, ok := int32Port(config.Port.ValueInt64())
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Daytona sandbox port",
			"Configure port with an integer from 1 to 65535.",
		)
		return
	}

	sandboxID, httpResp, err := d.client.api.PreviewAPI.GetSandboxIdFromSignedPreviewUrlToken(ctx, signedPreviewToken, float32(port)).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox ID from signed preview URL token", "read sandbox ID from signed preview URL token", httpResp, err)
		return
	}
	sandboxID = normalizeStringResponse(sandboxID)

	data := sandboxIDFromSignedPreviewTokenDataSourceModel{
		ID:                 types.StringValue(fmt.Sprintf("%d:signed_preview_token_sandbox", port)),
		SignedPreviewToken: config.SignedPreviewToken,
		Port:               config.Port,
		SandboxID:          types.StringValue(sandboxID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func normalizeStringResponse(value string) string {
	value = strings.TrimSpace(value)
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	return value
}

type SandboxSSHAccessValidationDataSource struct {
	client *daytonaClient
}

type sandboxSSHAccessValidationDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Token                 types.String `tfsdk:"token"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
	Valid                 types.Bool   `tfsdk:"valid"`
	SandboxID             types.String `tfsdk:"sandbox_id"`
}

type sandboxSSHAccessValidationConfigModel struct {
	Token                 types.String `tfsdk:"token"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
}

func (d *SandboxSSHAccessValidationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_ssh_access_validation"
}

func (d *SandboxSSHAccessValidationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Validates a Daytona sandbox SSH access token.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"token":                   requiredSensitiveDataSourceStringAttribute("SSH access token to validate."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"valid":                   computedDataSourceBoolAttribute("Whether the SSH access token is valid."),
			"sandbox_id":              computedDataSourceStringAttribute("Sandbox ID for the SSH access token."),
		},
	}
}

func (d *SandboxSSHAccessValidationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxSSHAccessValidationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxSSHAccessValidationConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := strings.TrimSpace(config.Token.ValueString())
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona SSH access token",
			"Configure token with the Daytona SSH access token to validate.",
		)
		return
	}

	request := d.client.api.SandboxAPI.ValidateSshAccess(ctx).Token(token)
	if organizationID := optionalString(config.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	validation, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to validate Daytona sandbox SSH access", "validate sandbox SSH access", httpResp, err)
		return
	}
	if validation == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona sandbox SSH access validation response",
			"Daytona returned a successful response without sandbox SSH access validation data.",
		)
		return
	}

	data := sandboxSSHAccessValidationDataSourceModel{
		ID:                    types.StringValue(validation.SandboxId + ":ssh_access_validation"),
		Token:                 config.Token,
		RequestOrganizationID: config.RequestOrganizationID,
		Valid:                 types.BoolValue(validation.Valid),
		SandboxID:             types.StringValue(validation.SandboxId),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SnapshotBuildLogsURLDataSource struct {
	client *daytonaClient
}

type snapshotBuildLogsURLDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	SnapshotID     types.String `tfsdk:"snapshot_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	URL            types.String `tfsdk:"url"`
}

func (d *SnapshotBuildLogsURLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_build_logs_url"
}

func (d *SnapshotBuildLogsURLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a build logs URL for a Daytona snapshot.",
		Attributes: map[string]schema.Attribute{
			"id":              computedDataSourceStringAttribute("Data source identifier."),
			"snapshot_id":     requiredDataSourceStringAttribute("Snapshot ID."),
			"organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"url":             sensitiveComputedDataSourceStringAttribute("Snapshot build logs URL."),
		},
	}
}

func (d *SnapshotBuildLogsURLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SnapshotBuildLogsURLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data snapshotBuildLogsURLDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SnapshotsAPI.GetSnapshotBuildLogsUrl(ctx, data.SnapshotID.ValueString())
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	url, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona snapshot build logs URL", "read snapshot build logs URL", httpResp, err)
		return
	}
	if url == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona snapshot build logs URL response",
			"Daytona returned a successful response without snapshot build logs URL data.",
		)
		return
	}

	data.ID = types.StringValue(data.SnapshotID.ValueString() + ":build_logs_url")
	data.URL = types.StringValue(url.Url)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func requiredDataSourceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:            true,
		MarkdownDescription: description,
	}
}

func requiredSensitiveDataSourceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:            true,
		Sensitive:           true,
		MarkdownDescription: description,
	}
}

func optionalOrganizationIDDataSourceStringAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Daytona organization ID to send as `X-Daytona-Organization-ID`.",
	}
}
