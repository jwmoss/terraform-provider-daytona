package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &JobPollDataSource{}

func NewJobPollDataSource() datasource.DataSource {
	return &JobPollDataSource{}
}

type JobPollDataSource struct {
	client *daytonaClient
}

type jobPollDataSourceModel struct {
	ID      types.String         `tfsdk:"id"`
	Timeout types.Int64          `tfsdk:"timeout"`
	Limit   types.Int64          `tfsdk:"limit"`
	Items   []jobDataSourceModel `tfsdk:"items"`
}

func (d *JobPollDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_poll"
}

func (d *JobPollDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Long-polls Daytona for pending jobs visible to runner/job-worker credentials.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Long-poll timeout in seconds. Daytona defaults to 30 seconds and allows up to 60 seconds.",
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum number of jobs to return. Daytona defaults to 10 and allows up to 100.",
			},
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

func (d *JobPollDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *JobPollDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data jobPollDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.JobsAPI.PollJobs(ctx)
	if !data.Timeout.IsNull() {
		request = request.Timeout(float32(data.Timeout.ValueInt64()))
	} else {
		data.Timeout = types.Int64Value(30)
	}
	if !data.Limit.IsNull() {
		request = request.Limit(float32(data.Limit.ValueInt64()))
	} else {
		data.Limit = types.Int64Value(10)
	}

	result, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to poll Daytona jobs", "poll jobs", httpResp, err)
		return
	}
	if result == nil {
		addEmptyAPIResponseError(&resp.Diagnostics, "Empty Daytona job poll response", "poll jobs", httpResp)
		return
	}

	items := make([]jobDataSourceModel, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		items = append(items, jobModel(job))
	}

	data.ID = types.StringValue("job_poll")
	data.Items = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
