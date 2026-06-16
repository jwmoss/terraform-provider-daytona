package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RegionResource{}
var _ resource.ResourceWithImportState = &RegionResource{}

func NewRegionResource() resource.Resource {
	return &RegionResource{}
}

type RegionResource struct {
	client *daytonaClient
}

type regionResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	OrganizationID            types.String `tfsdk:"organization_id"`
	RegionType                types.String `tfsdk:"region_type"`
	ProxyURL                  types.String `tfsdk:"proxy_url"`
	SSHGatewayURL             types.String `tfsdk:"ssh_gateway_url"`
	SnapshotManagerURL        types.String `tfsdk:"snapshot_manager_url"`
	ProxyAPIKey               types.String `tfsdk:"proxy_api_key"`
	SSHGatewayAPIKey          types.String `tfsdk:"ssh_gateway_api_key"`
	SnapshotManagerUsername   types.String `tfsdk:"snapshot_manager_username"`
	SnapshotManagerPassword   types.String `tfsdk:"snapshot_manager_password"`
	ProxyAPIKeyRotationID     types.String `tfsdk:"proxy_api_key_rotation_id"`
	SSHGatewayRotationID      types.String `tfsdk:"ssh_gateway_api_key_rotation_id"`
	SnapshotManagerRotationID types.String `tfsdk:"snapshot_manager_credentials_rotation_id"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
}

func (r *RegionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region"
}

func (r *RegionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona customer region. **Experimental / self-hosted only:** the region create/update/delete endpoints are not served by the managed Daytona cloud (`app.daytona.io` returns HTTP 404) and only work against self-hosted Daytona deployments where region management is enabled.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona region ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Region name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id":      computedStringAttribute("Daytona organization ID that owns the region."),
			"region_type":          computedStringAttribute("Region type."),
			"proxy_url":            optionalStringAttribute("Proxy URL for the region."),
			"ssh_gateway_url":      optionalStringAttribute("SSH gateway URL for the region."),
			"snapshot_manager_url": optionalStringAttribute("Snapshot manager URL for the region."),
			"proxy_api_key_rotation_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Arbitrary rotation trigger for the region proxy API key. Change this value to regenerate the proxy API key and store the returned key in `proxy_api_key`.",
			},
			"ssh_gateway_api_key_rotation_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Arbitrary rotation trigger for the region SSH gateway API key. Change this value to regenerate the SSH gateway API key and store the returned key in `ssh_gateway_api_key`.",
			},
			"snapshot_manager_credentials_rotation_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Arbitrary rotation trigger for the region snapshot manager credentials. Change this value to regenerate the snapshot manager username and password.",
			},
			"proxy_api_key":             sensitiveComputedStringAttribute("Proxy API key returned when the region is created."),
			"ssh_gateway_api_key":       sensitiveComputedStringAttribute("SSH gateway API key returned when the region is created."),
			"snapshot_manager_username": sensitiveComputedStringAttribute("Snapshot manager username returned when the region is created."),
			"snapshot_manager_password": sensitiveComputedStringAttribute("Snapshot manager password returned when the region is created."),
			"created_at":                computedStringAttribute("Region creation timestamp."),
			"updated_at":                computedStringAttribute("Region update timestamp."),
		},
	}
}

func optionalStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: description,
	}
}

func sensitiveComputedStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:            true,
		Sensitive:           true,
		MarkdownDescription: description,
	}
}

func (r *RegionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RegionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data regionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRegion := apiclient.NewCreateRegion(data.Name.ValueString())
	if value := optionalString(data.ProxyURL); value != nil {
		createRegion.SetProxyUrl(*value)
	}
	if value := optionalString(data.SSHGatewayURL); value != nil {
		createRegion.SetSshGatewayUrl(*value)
	}
	if value := optionalString(data.SnapshotManagerURL); value != nil {
		createRegion.SetSnapshotManagerUrl(*value)
	}

	created, httpResp, err := r.client.api.OrganizationsAPI.CreateRegion(ctx).
		CreateRegion(*createRegion).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona region", "create region", httpResp, err)
		return
	}

	data.ID = types.StringValue(created.Id)
	data = flattenRegionCreateResponse(created, data)
	region, httpResp, err := r.client.api.OrganizationsAPI.GetRegionById(ctx, created.Id).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read created Daytona region", "read region", httpResp, err)
		return
	}

	data = flattenRegion(region, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data regionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	region, httpResp, err := r.client.api.OrganizationsAPI.GetRegionById(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona region", "read region", httpResp, err)
		return
	}

	data = flattenRegion(region, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data regionResourceModel
	var prior regionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ProxyAPIKey = prior.ProxyAPIKey
	data.SSHGatewayAPIKey = prior.SSHGatewayAPIKey
	data.SnapshotManagerUsername = prior.SnapshotManagerUsername
	data.SnapshotManagerPassword = prior.SnapshotManagerPassword

	updateRegion := apiclient.NewUpdateRegion()
	if value := optionalString(data.ProxyURL); value != nil {
		updateRegion.SetProxyUrl(*value)
	}
	if value := optionalString(data.SSHGatewayURL); value != nil {
		updateRegion.SetSshGatewayUrl(*value)
	}
	if value := optionalString(data.SnapshotManagerURL); value != nil {
		updateRegion.SetSnapshotManagerUrl(*value)
	}

	httpResp, err := r.client.api.OrganizationsAPI.UpdateRegion(ctx, data.ID.ValueString()).
		UpdateRegion(*updateRegion).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona region", "update region", httpResp, err)
		return
	}

	region, httpResp, err := r.client.api.OrganizationsAPI.GetRegionById(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona region", "read region", httpResp, err)
		return
	}

	data = flattenRegion(region, data)

	// Rotation IDs start at their prior values and advance one at a time as each
	// rotation succeeds and persists, so a partial failure cannot lose a freshly
	// regenerated credential or skip a pending rotation on the next apply.
	planned := data
	data.ProxyAPIKeyRotationID = prior.ProxyAPIKeyRotationID
	data.SSHGatewayRotationID = prior.SSHGatewayRotationID
	data.SnapshotManagerRotationID = prior.SnapshotManagerRotationID

	persist := func() bool {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return !resp.Diagnostics.HasError()
	}

	httpResp, err = r.applyRegionCredentialRotations(ctx, planned, prior, &data, persist)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to rotate Daytona region credentials", "rotate region credentials", httpResp, err)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data regionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.DeleteRegion(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona region", "delete region", httpResp, err)
		return
	}
}

func (r *RegionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenRegionCreateResponse(created *apiclient.CreateRegionResponse, prior regionResourceModel) regionResourceModel {
	if created == nil {
		return prior
	}

	if value, ok := created.GetProxyApiKeyOk(); ok && value != nil {
		prior.ProxyAPIKey = types.StringValue(*value)
	}
	if value, ok := created.GetSshGatewayApiKeyOk(); ok && value != nil {
		prior.SSHGatewayAPIKey = types.StringValue(*value)
	}
	if value, ok := created.GetSnapshotManagerUsernameOk(); ok && value != nil {
		prior.SnapshotManagerUsername = types.StringValue(*value)
	}
	if value, ok := created.GetSnapshotManagerPasswordOk(); ok && value != nil {
		prior.SnapshotManagerPassword = types.StringValue(*value)
	}

	return prior
}

func flattenRegion(region *apiclient.Region, prior regionResourceModel) regionResourceModel {
	if region == nil {
		return prior
	}

	prior.ID = types.StringValue(region.Id)
	prior.Name = types.StringValue(region.Name)
	prior.RegionType = types.StringValue(string(region.RegionType))
	prior.CreatedAt = types.StringValue(region.CreatedAt)
	prior.UpdatedAt = types.StringValue(region.UpdatedAt)

	if value, ok := region.GetOrganizationIdOk(); ok && value != nil {
		prior.OrganizationID = types.StringValue(*value)
	} else {
		prior.OrganizationID = types.StringNull()
	}
	if value, ok := region.GetProxyUrlOk(); ok && value != nil {
		prior.ProxyURL = types.StringValue(*value)
	} else if prior.ProxyURL.IsUnknown() {
		prior.ProxyURL = types.StringNull()
	}
	if value, ok := region.GetSshGatewayUrlOk(); ok && value != nil {
		prior.SSHGatewayURL = types.StringValue(*value)
	} else if prior.SSHGatewayURL.IsUnknown() {
		prior.SSHGatewayURL = types.StringNull()
	}
	if value, ok := region.GetSnapshotManagerUrlOk(); ok && value != nil {
		prior.SnapshotManagerURL = types.StringValue(*value)
	} else if prior.SnapshotManagerURL.IsUnknown() {
		prior.SnapshotManagerURL = types.StringNull()
	}

	return prior
}

func (r *RegionResource) applyRegionCredentialRotations(ctx context.Context, planned, prior regionResourceModel, data *regionResourceModel, persist func() bool) (*http.Response, error) {
	var httpResp *http.Response
	regionID := planned.ID.ValueString()

	if rotationIDChanged(planned.ProxyAPIKeyRotationID, prior.ProxyAPIKeyRotationID) {
		rotated, resp, err := r.client.api.OrganizationsAPI.RegenerateProxyApiKey(ctx, regionID).Execute()
		httpResp = resp
		if err != nil {
			return httpResp, err
		}
		if rotated != nil {
			data.ProxyAPIKey = types.StringValue(rotated.ApiKey)
		}
		data.ProxyAPIKeyRotationID = planned.ProxyAPIKeyRotationID
		if !persist() {
			return httpResp, nil
		}
	}

	if rotationIDChanged(planned.SSHGatewayRotationID, prior.SSHGatewayRotationID) {
		rotated, resp, err := r.client.api.OrganizationsAPI.RegenerateSshGatewayApiKey(ctx, regionID).Execute()
		httpResp = resp
		if err != nil {
			return httpResp, err
		}
		if rotated != nil {
			data.SSHGatewayAPIKey = types.StringValue(rotated.ApiKey)
		}
		data.SSHGatewayRotationID = planned.SSHGatewayRotationID
		if !persist() {
			return httpResp, nil
		}
	}

	if rotationIDChanged(planned.SnapshotManagerRotationID, prior.SnapshotManagerRotationID) {
		rotated, resp, err := r.client.api.OrganizationsAPI.RegenerateSnapshotManagerCredentials(ctx, regionID).Execute()
		httpResp = resp
		if err != nil {
			return httpResp, err
		}
		if rotated != nil {
			data.SnapshotManagerUsername = types.StringValue(rotated.Username)
			data.SnapshotManagerPassword = types.StringValue(rotated.Password)
		}
		data.SnapshotManagerRotationID = planned.SnapshotManagerRotationID
		if !persist() {
			return httpResp, nil
		}
	}

	return httpResp, nil
}

func rotationIDChanged(planned, prior types.String) bool {
	if planned.IsUnknown() {
		return false
	}
	if planned.IsNull() && prior.IsNull() {
		return false
	}
	if planned.IsNull() != prior.IsNull() {
		return true
	}
	return planned.ValueString() != prior.ValueString()
}
