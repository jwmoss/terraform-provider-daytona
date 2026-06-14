package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AdminRunnerDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminRunnerDataSource{}
var _ datasource.DataSource = &AdminRunnersDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminRunnersDataSource{}

func NewAdminRunnerDataSource() datasource.DataSource {
	return &AdminRunnerDataSource{}
}

func NewAdminRunnersDataSource() datasource.DataSource {
	return &AdminRunnersDataSource{}
}

type AdminRunnerDataSource struct {
	client *daytonaClient
}

type AdminRunnersDataSource struct {
	client *daytonaClient
}

type adminRunnerDataSourceConfigModel struct {
	ID types.String `tfsdk:"id"`
}

type adminRunnersDataSourceModel struct {
	ID       types.String                `tfsdk:"id"`
	RegionID types.String                `tfsdk:"region_id"`
	Items    []runnerFullDataSourceModel `tfsdk:"items"`
}

type adminRunnersDataSourceConfigModel struct {
	RegionID types.String `tfsdk:"region_id"`
}

func (d *AdminRunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_runner"
}

func (d *AdminRunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads full Daytona runner details by runner ID using Daytona admin APIs.",
		Attributes:          runnerFullDataSourceAttributes("id"),
	}
}

func (d *AdminRunnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminRunnerDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runnerID := strings.TrimSpace(config.ID.ValueString())
	if runnerID == "" {
		resp.Diagnostics.AddError("Missing Daytona runner ID", "Configure the id attribute with the Daytona runner ID to read.")
		return
	}

	runner, httpResp, err := d.client.api.AdminAPI.AdminGetRunnerById(ctx, runnerID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin runner", "read admin runner", httpResp, err)
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Empty Daytona admin runner response", fmt.Sprintf("Daytona returned a successful response without runner %q.", runnerID))
		return
	}

	data := runnerFullDataSourceModel{ID: config.ID}
	data = flattenRunnerFullDataSource(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *AdminRunnersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_runners"
}

func (d *AdminRunnersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Daytona runners using Daytona admin APIs.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"region_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional Daytona region ID used to filter admin runner results.",
			},
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona runners.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: runnerFullDataSourceAttributes(""),
				},
			},
		},
	}
}

func (d *AdminRunnersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminRunnersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminRunnersDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.AdminAPI.AdminListRunners(ctx)
	regionID := strings.TrimSpace(config.RegionID.ValueString())
	if regionID != "" {
		request = request.RegionId(regionID)
	}

	runners, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona admin runners", "list admin runners", httpResp, err)
		return
	}

	items := make([]runnerFullDataSourceModel, 0, len(runners))
	for i := range runners {
		items = append(items, flattenRunnerFullDataSource(ctx, &runners[i], runnerFullDataSourceModel{}))
	}

	data := adminRunnersDataSourceModel{
		ID:       types.StringValue("admin_runners"),
		RegionID: config.RegionID,
		Items:    items,
	}
	if regionID != "" {
		data.ID = types.StringValue("admin_runners:" + regionID)
		data.RegionID = types.StringValue(regionID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
