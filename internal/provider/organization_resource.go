// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

type OrganizationResource struct {
	client *daytonaClient
}

type organizationResourceModel struct {
	ID                                  types.String  `tfsdk:"id"`
	Name                                types.String  `tfsdk:"name"`
	DefaultRegionID                     types.String  `tfsdk:"default_region_id"`
	CreatedBy                           types.String  `tfsdk:"created_by"`
	Personal                            types.Bool    `tfsdk:"personal"`
	Suspended                           types.Bool    `tfsdk:"suspended"`
	SuspendedAt                         types.String  `tfsdk:"suspended_at"`
	SuspensionReason                    types.String  `tfsdk:"suspension_reason"`
	SuspendedUntil                      types.String  `tfsdk:"suspended_until"`
	SuspensionCleanupGracePeriodHours   types.Float64 `tfsdk:"suspension_cleanup_grace_period_hours"`
	MaxCPUPerSandbox                    types.Float64 `tfsdk:"max_cpu_per_sandbox"`
	MaxMemoryPerSandbox                 types.Float64 `tfsdk:"max_memory_per_sandbox"`
	MaxDiskPerSandbox                   types.Float64 `tfsdk:"max_disk_per_sandbox"`
	SnapshotDeactivationTimeoutMinutes  types.Float64 `tfsdk:"snapshot_deactivation_timeout_minutes"`
	SandboxLimitedNetworkEgress         types.Bool    `tfsdk:"sandbox_limited_network_egress"`
	AuthenticatedRateLimit              types.Float64 `tfsdk:"authenticated_rate_limit"`
	SandboxCreateRateLimit              types.Float64 `tfsdk:"sandbox_create_rate_limit"`
	SandboxLifecycleRateLimit           types.Float64 `tfsdk:"sandbox_lifecycle_rate_limit"`
	AuthenticatedRateLimitTTLSeconds    types.Float64 `tfsdk:"authenticated_rate_limit_ttl_seconds"`
	SandboxCreateRateLimitTTLSeconds    types.Float64 `tfsdk:"sandbox_create_rate_limit_ttl_seconds"`
	SandboxLifecycleRateLimitTTLSeconds types.Float64 `tfsdk:"sandbox_lifecycle_rate_limit_ttl_seconds"`
	CreatedAt                           types.String  `tfsdk:"created_at"`
	UpdatedAt                           types.String  `tfsdk:"updated_at"`
}

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_region_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Default Daytona region ID for the organization.",
			},
			"created_by":                               computedStringAttribute("User ID of the organization creator."),
			"personal":                                 computedBoolAttribute("Whether this is a personal organization."),
			"suspended":                                computedBoolAttribute("Whether the organization is suspended."),
			"suspended_at":                             computedStringAttribute("Organization suspension timestamp, when available."),
			"suspension_reason":                        computedStringAttribute("Organization suspension reason, when available."),
			"suspended_until":                          computedStringAttribute("Suspension end timestamp, when available."),
			"suspension_cleanup_grace_period_hours":    computedFloat64Attribute("Suspension cleanup grace period in hours."),
			"max_cpu_per_sandbox":                      computedFloat64Attribute("Maximum CPU per sandbox."),
			"max_memory_per_sandbox":                   computedFloat64Attribute("Maximum memory per sandbox."),
			"max_disk_per_sandbox":                     computedFloat64Attribute("Maximum disk per sandbox."),
			"snapshot_deactivation_timeout_minutes":    computedFloat64Attribute("Snapshot deactivation timeout in minutes."),
			"sandbox_limited_network_egress":           computedBoolAttribute("Default limited network egress setting for new sandboxes."),
			"authenticated_rate_limit":                 computedFloat64Attribute("Authenticated request rate limit per minute."),
			"sandbox_create_rate_limit":                computedFloat64Attribute("Sandbox create rate limit per minute."),
			"sandbox_lifecycle_rate_limit":             computedFloat64Attribute("Sandbox lifecycle rate limit per minute."),
			"authenticated_rate_limit_ttl_seconds":     computedFloat64Attribute("Authenticated request rate-limit TTL in seconds."),
			"sandbox_create_rate_limit_ttl_seconds":    computedFloat64Attribute("Sandbox create rate-limit TTL in seconds."),
			"sandbox_lifecycle_rate_limit_ttl_seconds": computedFloat64Attribute("Sandbox lifecycle rate-limit TTL in seconds."),
			"created_at":                               computedStringAttribute("Organization creation timestamp."),
			"updated_at":                               computedStringAttribute("Organization update timestamp."),
		},
	}
}

func computedFloat64Attribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Computed:            true,
		MarkdownDescription: description,
	}
}

func requiredStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:            true,
		MarkdownDescription: description,
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, httpResp, err := r.client.api.OrganizationsAPI.CreateOrganization(ctx).
		CreateOrganization(*apiclient.NewCreateOrganization(data.Name.ValueString(), data.DefaultRegionID.ValueString())).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona organization", "create organization", httpResp, err)
		return
	}

	data = flattenOrganization(created, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization, httpResp, err := r.client.api.OrganizationsAPI.GetOrganization(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization", "read organization", httpResp, err)
		return
	}

	data = flattenOrganization(organization, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationResourceModel
	var state organizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.DefaultRegionID.Equal(state.DefaultRegionID) {
		httpResp, err := r.client.api.OrganizationsAPI.SetOrganizationDefaultRegion(ctx, state.ID.ValueString()).
			UpdateOrganizationDefaultRegion(*apiclient.NewUpdateOrganizationDefaultRegion(plan.DefaultRegionID.ValueString())).
			Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to update Daytona organization default region", "update organization default region", httpResp, err)
			return
		}
	}

	organization, httpResp, err := r.client.api.OrganizationsAPI.GetOrganization(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization", "read organization", httpResp, err)
		return
	}

	plan = flattenOrganization(organization, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.OrganizationsAPI.DeleteOrganization(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona organization", "delete organization", httpResp, err)
		return
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenOrganization(organization *apiclient.Organization, prior organizationResourceModel) organizationResourceModel {
	if organization == nil {
		return prior
	}

	prior.ID = types.StringValue(organization.Id)
	prior.Name = types.StringValue(organization.Name)
	prior.CreatedBy = types.StringValue(organization.CreatedBy)
	prior.Personal = types.BoolValue(organization.Personal)
	prior.Suspended = types.BoolValue(organization.Suspended)
	prior.SuspendedAt = terraformTimeString(organization.SuspendedAt)
	prior.SuspensionReason = nullablePlainString(organization.SuspensionReason)
	prior.SuspendedUntil = terraformTimeString(organization.SuspendedUntil)
	prior.SuspensionCleanupGracePeriodHours = types.Float64Value(float64(organization.SuspensionCleanupGracePeriodHours))
	prior.MaxCPUPerSandbox = types.Float64Value(float64(organization.MaxCpuPerSandbox))
	prior.MaxMemoryPerSandbox = types.Float64Value(float64(organization.MaxMemoryPerSandbox))
	prior.MaxDiskPerSandbox = types.Float64Value(float64(organization.MaxDiskPerSandbox))
	prior.SnapshotDeactivationTimeoutMinutes = types.Float64Value(float64(organization.SnapshotDeactivationTimeoutMinutes))
	prior.SandboxLimitedNetworkEgress = types.BoolValue(organization.SandboxLimitedNetworkEgress)
	prior.AuthenticatedRateLimit = nullableFloat32(organization.AuthenticatedRateLimit)
	prior.SandboxCreateRateLimit = nullableFloat32(organization.SandboxCreateRateLimit)
	prior.SandboxLifecycleRateLimit = nullableFloat32(organization.SandboxLifecycleRateLimit)
	prior.AuthenticatedRateLimitTTLSeconds = nullableFloat32(organization.AuthenticatedRateLimitTtlSeconds)
	prior.SandboxCreateRateLimitTTLSeconds = nullableFloat32(organization.SandboxCreateRateLimitTtlSeconds)
	prior.SandboxLifecycleRateLimitTTLSeconds = nullableFloat32(organization.SandboxLifecycleRateLimitTtlSeconds)
	prior.CreatedAt = terraformTimeString(organization.CreatedAt)
	prior.UpdatedAt = terraformTimeString(organization.UpdatedAt)

	if value, ok := organization.GetDefaultRegionIdOk(); ok && value != nil {
		prior.DefaultRegionID = types.StringValue(*value)
	} else {
		prior.DefaultRegionID = types.StringNull()
	}

	return prior
}

func nullablePlainString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func nullableFloat32(value apiclient.NullableFloat32) types.Float64 {
	if !value.IsSet() || value.Get() == nil {
		return types.Float64Null()
	}
	return types.Float64Value(float64(*value.Get()))
}
