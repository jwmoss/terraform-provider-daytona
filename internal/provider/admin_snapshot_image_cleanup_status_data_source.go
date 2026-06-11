// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AdminSnapshotImageCleanupStatusDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminSnapshotImageCleanupStatusDataSource{}

func NewAdminSnapshotImageCleanupStatusDataSource() datasource.DataSource {
	return &AdminSnapshotImageCleanupStatusDataSource{}
}

type AdminSnapshotImageCleanupStatusDataSource struct {
	client *daytonaClient
}

type adminSnapshotImageCleanupStatusDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	ImageName  types.String `tfsdk:"image_name"`
	CanCleanup types.Bool   `tfsdk:"can_cleanup"`
}

type adminSnapshotImageCleanupStatusConfigModel struct {
	ImageName types.String `tfsdk:"image_name"`
}

func (d *AdminSnapshotImageCleanupStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_snapshot_image_cleanup_status"
}

func (d *AdminSnapshotImageCleanupStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks whether a Daytona snapshot image can be cleaned up through the Daytona admin API. This data source requires Daytona admin privileges.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"image_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Snapshot image name with tag to check.",
			},
			"can_cleanup": computedDataSourceBoolAttribute("Whether Daytona reports that the image can be cleaned up."),
		},
	}
}

func (d *AdminSnapshotImageCleanupStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminSnapshotImageCleanupStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminSnapshotImageCleanupStatusConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	imageName := strings.TrimSpace(config.ImageName.ValueString())
	if imageName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona snapshot image name",
			"Configure image_name with the Daytona snapshot image name and tag to check.",
		)
		return
	}

	canCleanup, httpResp, err := d.client.api.AdminAPI.AdminCanCleanupImage(ctx).
		ImageName(imageName).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona snapshot image cleanup status", "read snapshot image cleanup status", httpResp, err)
		return
	}

	data := adminSnapshotImageCleanupStatusDataSourceModel{}
	data.ID = types.StringValue(imageName)
	data.ImageName = types.StringValue(imageName)
	data.CanCleanup = types.BoolValue(canCleanup)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
