package provider

import (
	"context"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WebhookInitializationStatusDataSource{}
var _ datasource.DataSource = &WebhookAppPortalAccessDataSource{}

func NewWebhookInitializationStatusDataSource() datasource.DataSource {
	return &WebhookInitializationStatusDataSource{}
}

func NewWebhookAppPortalAccessDataSource() datasource.DataSource {
	return &WebhookAppPortalAccessDataSource{}
}

type WebhookInitializationStatusDataSource struct {
	client *daytonaClient
}

type webhookInitializationStatusDataSourceModel struct {
	ID                types.String  `tfsdk:"id"`
	OrganizationID    types.String  `tfsdk:"organization_id"`
	Initialized       types.Bool    `tfsdk:"initialized"`
	SvixApplicationID types.String  `tfsdk:"svix_application_id"`
	LastError         types.String  `tfsdk:"last_error"`
	RetryCount        types.Float64 `tfsdk:"retry_count"`
	CreatedAt         types.String  `tfsdk:"created_at"`
	UpdatedAt         types.String  `tfsdk:"updated_at"`
}

func (d *WebhookInitializationStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_initialization_status"
}

func (d *WebhookInitializationStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads webhook initialization status for a Daytona organization. If no initialization record exists, the data source returns `initialized = false`.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID.",
			},
			"initialized":         computedDataSourceBoolAttribute("Whether Daytona has initialized a Svix application for the organization."),
			"svix_application_id": computedDataSourceStringAttribute("Svix application ID, when initialized."),
			"last_error":          computedDataSourceStringAttribute("Last webhook initialization error, when available."),
			"retry_count":         computedDataSourceFloat64Attribute("Number of webhook initialization attempts."),
			"created_at":          computedDataSourceStringAttribute("Webhook initialization record creation timestamp."),
			"updated_at":          computedDataSourceStringAttribute("Webhook initialization record update timestamp."),
		},
	}
}

func (d *WebhookInitializationStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *WebhookInitializationStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data webhookInitializationStatusDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := data.OrganizationID.ValueString()
	status, httpResp, err := d.client.api.WebhooksAPI.
		WebhookControllerGetInitializationStatus(ctx, organizationID).
		XDaytonaOrganizationID(organizationID).
		Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			data.ID = types.StringValue(organizationID + ":webhook_initialization_status")
			data.OrganizationID = types.StringValue(organizationID)
			data.Initialized = types.BoolValue(false)
			data.SvixApplicationID = types.StringNull()
			data.LastError = types.StringNull()
			data.RetryCount = types.Float64Null()
			data.CreatedAt = types.StringNull()
			data.UpdatedAt = types.StringNull()

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}

		addAPIError(&resp.Diagnostics, "Unable to read Daytona webhook initialization status", "read webhook initialization status", httpResp, err)
		return
	}
	if status == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona webhook initialization status response",
			"Daytona returned a successful response without webhook initialization status data.",
		)
		return
	}

	data.ID = types.StringValue(organizationID + ":webhook_initialization_status")
	data.OrganizationID = types.StringValue(status.OrganizationId)
	data.SvixApplicationID = nullableStringValue(status.SvixApplicationId)
	data.LastError = nullableStringValue(status.LastError)
	data.Initialized = types.BoolValue(!data.SvixApplicationID.IsNull() && data.SvixApplicationID.ValueString() != "")
	data.RetryCount = float64Value(status.RetryCount)
	data.CreatedAt = types.StringValue(status.CreatedAt)
	data.UpdatedAt = types.StringValue(status.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type WebhookAppPortalAccessDataSource struct {
	client *daytonaClient
}

type webhookAppPortalAccessDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Token          types.String `tfsdk:"token"`
	URL            types.String `tfsdk:"url"`
}

func (d *WebhookAppPortalAccessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_app_portal_access"
}

func (d *WebhookAppPortalAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Svix consumer app portal access for a Daytona organization.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID.",
			},
			"token": sensitiveComputedDataSourceStringAttribute("Svix consumer app portal authentication token."),
			"url":   computedDataSourceStringAttribute("Svix consumer app portal URL."),
		},
	}
}

func (d *WebhookAppPortalAccessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *WebhookAppPortalAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data webhookAppPortalAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := data.OrganizationID.ValueString()
	access, httpResp, err := d.client.api.WebhooksAPI.
		WebhookControllerGetAppPortalAccess(ctx, organizationID).
		XDaytonaOrganizationID(organizationID).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona webhook app portal access", "read webhook app portal access", httpResp, err)
		return
	}
	if access == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona webhook app portal access response",
			"Daytona returned a successful response without webhook app portal access data.",
		)
		return
	}

	data.ID = types.StringValue(organizationID + ":webhook_app_portal_access")
	data.OrganizationID = types.StringValue(organizationID)
	data.Token = types.StringValue(access.Token)
	data.URL = types.StringValue(access.Url)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func nullableStringValue(value apiclient.NullableString) types.String {
	if !value.IsSet() || value.Get() == nil {
		return types.StringNull()
	}
	return types.StringValue(*value.Get())
}
