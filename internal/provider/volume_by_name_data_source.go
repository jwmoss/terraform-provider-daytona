package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeByNameDataSource{}

func NewVolumeByNameDataSource() datasource.DataSource {
	return &VolumeByNameDataSource{}
}

type VolumeByNameDataSource struct {
	client *daytonaClient
}

type volumeByNameConfigModel struct {
	Name                  types.String `tfsdk:"name"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
}

type volumeByNameDataSourceModel struct {
	Name                  types.String `tfsdk:"name"`
	RequestOrganizationID types.String `tfsdk:"request_organization_id"`
	ID                    types.String `tfsdk:"id"`
	OrganizationID        types.String `tfsdk:"organization_id"`
	State                 types.String `tfsdk:"state"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	LastUsedAt            types.String `tfsdk:"last_used_at"`
	ErrorReason           types.String `tfsdk:"error_reason"`
}

func (d *VolumeByNameDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_by_name"
}

func (d *VolumeByNameDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona persistent volume by name.",
		Attributes: map[string]schema.Attribute{
			"name":                    requiredDataSourceStringAttribute("Daytona volume name."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"id":                      computedDataSourceStringAttribute("Daytona volume ID."),
			"organization_id":         computedDataSourceStringAttribute("Daytona organization ID that owns the volume."),
			"state":                   computedDataSourceStringAttribute("Current volume state."),
			"created_at":              computedDataSourceStringAttribute("Volume creation timestamp."),
			"updated_at":              computedDataSourceStringAttribute("Volume update timestamp."),
			"last_used_at":            computedDataSourceStringAttribute("Volume last-used timestamp, when available."),
			"error_reason":            computedDataSourceStringAttribute("Volume error reason, when available."),
		},
	}
}

func (d *VolumeByNameDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *VolumeByNameDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config volumeByNameConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.VolumesAPI.GetVolumeByName(ctx, config.Name.ValueString())
	if organizationID := optionalString(config.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	volume, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona volume by name", "read volume by name", httpResp, err)
		return
	}
	if volume == nil {
		resp.Diagnostics.AddError("Empty Daytona volume response", fmt.Sprintf("Daytona returned a successful response without volume %q.", config.Name.ValueString()))
		return
	}

	flattened := flattenVolumeDataSource(volume)
	data := volumeByNameDataSourceModel{
		Name:                  flattened.Name,
		RequestOrganizationID: config.RequestOrganizationID,
		ID:                    flattened.ID,
		OrganizationID:        flattened.OrganizationID,
		State:                 flattened.State,
		CreatedAt:             flattened.CreatedAt,
		UpdatedAt:             flattened.UpdatedAt,
		LastUsedAt:            flattened.LastUsedAt,
		ErrorReason:           flattened.ErrorReason,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
