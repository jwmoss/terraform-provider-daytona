package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationOtelConfigResource{}
var _ resource.ResourceWithImportState = &OrganizationOtelConfigResource{}

func NewOrganizationOtelConfigResource() resource.Resource {
	return &OrganizationOtelConfigResource{}
}

type OrganizationOtelConfigResource struct {
	client *daytonaClient
}

type organizationOtelConfigResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Endpoint       types.String `tfsdk:"endpoint"`
	Headers        types.Map    `tfsdk:"headers"`
}

func (r *OrganizationOtelConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_otel_config"
}

func (r *OrganizationOtelConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages OpenTelemetry export configuration for a Daytona organization. Requires the organization API (an access token / JWT); API-key auth is rejected. Header values are write-only: Daytona redacts them on read, so the configured values are kept in state and drift in header values cannot be detected.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization ID for the OpenTelemetry configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"endpoint": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OpenTelemetry collector endpoint.",
			},
			"headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional OpenTelemetry request headers.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrganizationOtelConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*daytonaClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *OrganizationOtelConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationOtelConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	otelConfig, ok := expandOrganizationOtelConfig(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.UpdateOrganizationOtelConfig(ctx, data.OrganizationID.ValueString()).
		OtelConfig(otelConfig).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona organization OpenTelemetry configuration", "update organization OpenTelemetry configuration", httpResp, err)
		return
	}

	// The write succeeded; persist the configured values. The dedicated otel-config
	// read endpoint is not authorized for an organization owner and the organization
	// object redacts header values, so there is nothing to read back here.
	data.ID = data.OrganizationID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationOtelConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationOtelConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The dedicated otel-config read endpoint is not authorized for an organization
	// owner, so read the configuration from the organization object, which carries it
	// (with header values redacted).
	organization, httpResp, err := r.client.api.OrganizationsAPI.GetOrganization(ctx, data.OrganizationID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization", "read organization", httpResp, err)
		return
	}

	config, ok := organization.GetOtelConfigOk()
	if !ok || config == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data = flattenOrganizationOtelConfig(ctx, config, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationOtelConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationOtelConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	otelConfig, ok := expandOrganizationOtelConfig(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.UpdateOrganizationOtelConfig(ctx, data.OrganizationID.ValueString()).
		OtelConfig(otelConfig).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona organization OpenTelemetry configuration", "update organization OpenTelemetry configuration", httpResp, err)
		return
	}

	data.ID = data.OrganizationID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationOtelConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationOtelConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.DeleteOrganizationOtelConfig(ctx, data.OrganizationID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona organization OpenTelemetry configuration", "delete organization OpenTelemetry configuration", httpResp, err)
		return
	}
}

func (r *OrganizationOtelConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), types.StringValue(req.ID))...)
}

func expandOrganizationOtelConfig(ctx context.Context, data organizationOtelConfigResourceModel, diags *diag.Diagnostics) (apiclient.OtelConfig, bool) {
	config := *apiclient.NewOtelConfig(data.Endpoint.ValueString())

	headers, headerDiags := stringMap(ctx, data.Headers)
	diags.Append(headerDiags...)
	if diags.HasError() {
		return config, false
	}
	if len(headers) > 0 {
		config.SetHeaders(headers)
	}

	return config, true
}

func flattenOrganizationOtelConfig(_ context.Context, config *apiclient.OtelConfig, prior organizationOtelConfigResourceModel) organizationOtelConfigResourceModel {
	if config == nil {
		return prior
	}

	prior.ID = prior.OrganizationID
	prior.Endpoint = types.StringValue(config.Endpoint)
	// Header values are redacted by the organization object, so the configured
	// values already in state are kept rather than overwritten with the redactions.

	return prior
}
