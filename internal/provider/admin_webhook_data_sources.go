package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AdminWebhookStatusDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminWebhookStatusDataSource{}
var _ datasource.DataSource = &AdminWebhookMessageAttemptsDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminWebhookMessageAttemptsDataSource{}

func NewAdminWebhookStatusDataSource() datasource.DataSource {
	return &AdminWebhookStatusDataSource{}
}

func NewAdminWebhookMessageAttemptsDataSource() datasource.DataSource {
	return &AdminWebhookMessageAttemptsDataSource{}
}

type AdminWebhookStatusDataSource struct {
	client *daytonaClient
}

type adminWebhookStatusDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (d *AdminWebhookStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_webhook_status"
}

func (d *AdminWebhookStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Daytona admin webhook service status. This data source requires Daytona admin privileges.",
		Attributes: map[string]schema.Attribute{
			"id":      computedDataSourceStringAttribute("Data source identifier."),
			"enabled": computedDataSourceBoolAttribute("Whether Daytona webhooks are enabled."),
		},
	}
}

func (d *AdminWebhookStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminWebhookStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	status, httpResp, err := d.client.api.AdminAPI.AdminGetWebhookStatus(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin webhook status", "read admin webhook status", httpResp, err)
		return
	}
	if status == nil {
		resp.Diagnostics.AddError(
			"Empty Daytona admin webhook status response",
			"Daytona returned a successful response without admin webhook status data.",
		)
		return
	}

	data := adminWebhookStatusDataSourceModel{
		ID:      types.StringValue("admin_webhook_status"),
		Enabled: pointerBoolValue(status.Enabled),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type AdminWebhookMessageAttemptsDataSource struct {
	client *daytonaClient
}

type adminWebhookMessageAttemptsDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	MessageID      types.String `tfsdk:"message_id"`
	AttemptsJSON   types.String `tfsdk:"attempts_json"`
}

type adminWebhookMessageAttemptsConfigModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	MessageID      types.String `tfsdk:"message_id"`
}

func (d *AdminWebhookMessageAttemptsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_webhook_message_attempts"
}

func (d *AdminWebhookMessageAttemptsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Daytona admin webhook delivery attempts for a message. This data source requires Daytona admin privileges.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID that owns the webhook message.",
			},
			"message_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona webhook message ID.",
			},
			"attempts_json": sensitiveComputedDataSourceStringAttribute("Webhook delivery attempts as JSON."),
		},
	}
}

func (d *AdminWebhookMessageAttemptsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminWebhookMessageAttemptsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminWebhookMessageAttemptsConfigModel
	var data adminWebhookMessageAttemptsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := config.OrganizationID.ValueString()
	messageID := config.MessageID.ValueString()
	attempts, httpResp, err := d.client.api.AdminAPI.AdminGetMessageAttempts(ctx, organizationID, messageID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin webhook message attempts", "read admin webhook message attempts", httpResp, err)
		return
	}

	data.ID = types.StringValue(organizationID + ":" + messageID + ":admin_webhook_message_attempts")
	data.OrganizationID = types.StringValue(organizationID)
	data.MessageID = types.StringValue(messageID)
	data.AttemptsJSON = jsonStringValue(attempts)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
