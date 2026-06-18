package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AdminAuditLogsDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminAuditLogsDataSource{}

func NewAdminAuditLogsDataSource() datasource.DataSource {
	return &AdminAuditLogsDataSource{}
}

type AdminAuditLogsDataSource struct {
	client *daytonaClient
}

type adminAuditLogsDataSourceModel struct {
	ID         types.String    `tfsdk:"id"`
	Page       types.Int64     `tfsdk:"page"`
	Limit      types.Int64     `tfsdk:"limit"`
	From       types.String    `tfsdk:"from"`
	To         types.String    `tfsdk:"to"`
	Cursor     types.String    `tfsdk:"cursor"`
	Total      types.Int64     `tfsdk:"total"`
	TotalPages types.Int64     `tfsdk:"total_pages"`
	NextToken  types.String    `tfsdk:"next_token"`
	Items      []auditLogModel `tfsdk:"items"`
}

type adminAuditLogsConfigModel struct {
	Page   types.Int64  `tfsdk:"page"`
	Limit  types.Int64  `tfsdk:"limit"`
	From   types.String `tfsdk:"from"`
	To     types.String `tfsdk:"to"`
	Cursor types.String `tfsdk:"cursor"`
}

func (d *AdminAuditLogsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_audit_logs"
}

func (d *AdminAuditLogsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads paginated Daytona audit logs across all organizations through the Daytona admin API. This data source requires Daytona admin privileges.",
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

func (d *AdminAuditLogsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminAuditLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminAuditLogsConfigModel
	var data adminAuditLogsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.AdminAPI.AdminGetAllAuditLogs(ctx)
	if !config.Page.IsNull() {
		request = request.Page(float32(config.Page.ValueInt64()))
		data.Page = config.Page
	} else {
		data.Page = types.Int64Value(1)
	}
	if !config.Limit.IsNull() {
		request = request.Limit(float32(config.Limit.ValueInt64()))
		data.Limit = config.Limit
	} else {
		data.Limit = types.Int64Value(100)
	}
	if !config.From.IsNull() {
		from, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "from", config.From.ValueString())
		if !ok {
			return
		}
		request = request.From(from)
		data.From = config.From
	}
	if !config.To.IsNull() {
		to, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "to", config.To.ValueString())
		if !ok {
			return
		}
		request = request.To(to)
		data.To = config.To
	}
	if !config.Cursor.IsNull() {
		request = request.NextToken(config.Cursor.ValueString())
		data.Cursor = config.Cursor
	}

	result, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin audit logs", "read admin audit logs", httpResp, err)
		return
	}
	if result == nil {
		addEmptyAPIResponseError(&resp.Diagnostics, "Empty Daytona admin audit logs response", "read admin audit logs", httpResp)
		return
	}

	data.ID = types.StringValue("admin_audit_logs")
	data.Total = types.Int64Value(int64(result.Total))
	data.TotalPages = types.Int64Value(int64(result.TotalPages))
	data.NextToken = pointerStringValue(result.NextToken)
	data.Items = auditLogModels(result.Items)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
