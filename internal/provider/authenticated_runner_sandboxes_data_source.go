package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AuthenticatedRunnerSandboxesDataSource{}

func NewAuthenticatedRunnerSandboxesDataSource() datasource.DataSource {
	return &AuthenticatedRunnerSandboxesDataSource{}
}

type AuthenticatedRunnerSandboxesDataSource struct {
	client *daytonaClient
}

type authenticatedRunnerSandboxesConfigModel struct {
	RequestOrganizationID    types.String `tfsdk:"request_organization_id"`
	States                   types.String `tfsdk:"states"`
	SkipReconcilingSandboxes types.Bool   `tfsdk:"skip_reconciling_sandboxes"`
}

type authenticatedRunnerSandboxesDataSourceModel struct {
	ID                       types.String                   `tfsdk:"id"`
	RequestOrganizationID    types.String                   `tfsdk:"request_organization_id"`
	States                   types.String                   `tfsdk:"states"`
	SkipReconcilingSandboxes types.Bool                     `tfsdk:"skip_reconciling_sandboxes"`
	Items                    []sandboxRelationshipItemModel `tfsdk:"items"`
}

func (d *AuthenticatedRunnerSandboxesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authenticated_runner_sandboxes"
}

func (d *AuthenticatedRunnerSandboxesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads sandboxes assigned to the authenticated Daytona runner.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"states": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comma-separated list of sandbox states to include.",
			},
			"skip_reconciling_sandboxes": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether sandboxes whose current state differs from desired state should be skipped.",
			},
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned Daytona sandboxes assigned to the authenticated runner.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxRelationshipComputedAttributes(),
				},
			},
		},
	}
}

func (d *AuthenticatedRunnerSandboxesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AuthenticatedRunnerSandboxesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config authenticatedRunnerSandboxesConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxesForRunner(ctx)
	if organizationID := optionalString(config.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if states := optionalString(config.States); states != nil {
		request = request.States(*states)
	}
	if terraformBoolConfigured(config.SkipReconcilingSandboxes) {
		request = request.SkipReconcilingSandboxes(config.SkipReconcilingSandboxes.ValueBool())
	}

	sandboxes, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona authenticated runner sandboxes", "read authenticated runner sandboxes", httpResp, err)
		return
	}

	data := authenticatedRunnerSandboxesDataSourceModel{
		ID:                       types.StringValue("authenticated_runner_sandboxes"),
		RequestOrganizationID:    config.RequestOrganizationID,
		States:                   config.States,
		SkipReconcilingSandboxes: config.SkipReconcilingSandboxes,
		Items:                    flattenSandboxRelationshipItems(ctx, sandboxes),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
