// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OrganizationOtelConfigDataSource{}

func NewOrganizationOtelConfigDataSource() datasource.DataSource {
	return &OrganizationOtelConfigDataSource{}
}

type OrganizationOtelConfigDataSource struct {
	client *daytonaClient
}

type organizationOtelConfigDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Endpoint       types.String `tfsdk:"endpoint"`
	Headers        types.Map    `tfsdk:"headers"`
}

func (d *OrganizationOtelConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_otel_config"
}

func (d *OrganizationOtelConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads OpenTelemetry export configuration for a Daytona organization.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID.",
			},
			"endpoint": computedDataSourceStringAttribute("OpenTelemetry collector endpoint."),
			"headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "OpenTelemetry request headers.",
			},
		},
	}
}

func (d *OrganizationOtelConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationOtelConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationOtelConfigDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, httpResp, err := d.client.api.OrganizationsAPI.GetOrganizationOtelConfig(ctx, data.OrganizationID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization OpenTelemetry configuration", "read organization OpenTelemetry configuration", httpResp, err)
		return
	}

	data.ID = data.OrganizationID
	data.Endpoint = types.StringValue(config.Endpoint)
	data.Headers = stringMapValue(ctx, config.Headers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
