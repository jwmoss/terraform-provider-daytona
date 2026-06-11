// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RunnerResource{}
var _ resource.ResourceWithImportState = &RunnerResource{}

func NewRunnerResource() resource.Resource {
	return &RunnerResource{}
}

type RunnerResource struct {
	client *daytonaClient
}

type runnerResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RegionID      types.String `tfsdk:"region_id"`
	Name          types.String `tfsdk:"name"`
	Tags          types.List   `tfsdk:"tags"`
	APIKey        types.String `tfsdk:"api_key"`
	Region        types.String `tfsdk:"region"`
	State         types.String `tfsdk:"state"`
	Unschedulable types.Bool   `tfsdk:"unschedulable"`
	Draining      types.Bool   `tfsdk:"draining"`
	CPU           types.String `tfsdk:"cpu"`
	Memory        types.String `tfsdk:"memory"`
	Disk          types.String `tfsdk:"disk"`
	GPU           types.String `tfsdk:"gpu"`
	GPUType       types.String `tfsdk:"gpu_type"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *RunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner"
}

func (r *RunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona custom runner registration. Daytona exposes custom runner create/list/update/delete only when organization infrastructure is enabled for the organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona runner ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Custom region ID where the runner is registered.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Runner name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Tags associated with the runner.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"api_key": sensitiveComputedStringAttribute("Runner API key returned when the runner is created."),
			"region":  computedStringAttribute("Runner region name."),
			"state":   computedStringAttribute("Current runner state."),
			"unschedulable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Daytona should stop scheduling new work on the runner.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"draining": schema.BoolAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Sets Daytona draining mode for the runner. Daytona accepts this value through the API but does not return it in runner reads, so Terraform treats it as write-only.",
			},
			"cpu":        computedStringAttribute("Runner CPU capacity."),
			"memory":     computedStringAttribute("Runner memory capacity in GiB."),
			"disk":       computedStringAttribute("Runner disk capacity in GiB."),
			"gpu":        computedStringAttribute("Runner GPU capacity, when available."),
			"gpu_type":   computedStringAttribute("Runner GPU type, when available."),
			"created_at": computedStringAttribute("Runner creation timestamp."),
			"updated_at": computedStringAttribute("Runner update timestamp."),
		},
	}
}

func (r *RunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data runnerResourceModel
	var config runnerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRunner := apiclient.NewCreateRunner(data.RegionID.ValueString(), data.Name.ValueString())
	tags, diags := stringList(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tags) > 0 {
		createRunner.SetTags(tags)
	}

	created, httpResp, err := r.client.api.RunnersAPI.CreateRunner(ctx).
		CreateRunner(*createRunner).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona runner", "create runner", httpResp, err)
		return
	}

	data.ID = types.StringValue(created.Id)
	data.APIKey = types.StringValue(created.ApiKey)

	runner, httpResp, err := r.client.api.RunnersAPI.GetRunnerById(ctx, created.Id).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read created Daytona runner", "read runner", httpResp, err)
		return
	}

	httpResp, err = r.applyRunnerOperationalSettings(ctx, created.Id, config, runner)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona runner operational settings", "update runner operational settings", httpResp, err)
		return
	}

	runner, httpResp, err = r.client.api.RunnersAPI.GetRunnerById(ctx, created.Id).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read created Daytona runner", "read runner", httpResp, err)
		return
	}

	data = flattenRunner(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data runnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner, httpResp, err := r.client.api.RunnersAPI.GetRunnerById(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona runner", "read runner", httpResp, err)
		return
	}

	data = flattenRunner(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config runnerResourceModel
	var plan runnerResourceModel
	var state runnerResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.applyRunnerOperationalSettings(ctx, state.ID.ValueString(), config, nil)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona runner operational settings", "update runner operational settings", httpResp, err)
		return
	}

	runner, httpResp, err := r.client.api.RunnersAPI.GetRunnerById(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona runner", "read runner", httpResp, err)
		return
	}

	plan = flattenRunner(ctx, runner, plan)
	plan.APIKey = state.APIKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data runnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.RunnersAPI.DeleteRunner(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona runner", "delete runner", httpResp, err)
		return
	}
}

func (r *RunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenRunner(ctx context.Context, runner *apiclient.Runner, prior runnerResourceModel) runnerResourceModel {
	if runner == nil {
		return prior
	}

	prior.ID = types.StringValue(runner.Id)
	prior.Name = types.StringValue(runner.Name)
	prior.Region = types.StringValue(runner.Region)
	prior.State = types.StringValue(string(runner.State))
	prior.Unschedulable = types.BoolValue(runner.Unschedulable)
	prior.CPU = types.StringValue(fmt.Sprintf("%g", runner.Cpu))
	prior.Memory = types.StringValue(fmt.Sprintf("%g", runner.Memory))
	prior.Disk = types.StringValue(fmt.Sprintf("%g", runner.Disk))
	prior.Tags = listStringValue(ctx, runner.Tags)
	prior.CreatedAt = types.StringValue(runner.CreatedAt)
	prior.UpdatedAt = types.StringValue(runner.UpdatedAt)

	if runner.Gpu != nil {
		prior.GPU = types.StringValue(fmt.Sprintf("%g", *runner.Gpu))
	} else {
		prior.GPU = types.StringNull()
	}
	if runner.GpuType != nil {
		prior.GPUType = types.StringValue(*runner.GpuType)
	} else {
		prior.GPUType = types.StringNull()
	}

	return prior
}

func (r *RunnerResource) applyRunnerOperationalSettings(ctx context.Context, runnerID string, config runnerResourceModel, current *apiclient.Runner) (*http.Response, error) {
	var httpResp *http.Response

	if configuredBool(config.Unschedulable) {
		unschedulable := config.Unschedulable.ValueBool()
		if current == nil || current.Unschedulable != unschedulable {
			resp, err := r.updateRunnerScheduling(ctx, runnerID, unschedulable)
			httpResp = resp
			if err != nil {
				return httpResp, err
			}
		}
	}

	if configuredBool(config.Draining) {
		resp, err := r.updateRunnerDraining(ctx, runnerID, config.Draining.ValueBool())
		httpResp = resp
		if err != nil {
			return httpResp, err
		}
	}

	return httpResp, nil
}

func (r *RunnerResource) updateRunnerScheduling(ctx context.Context, runnerID string, unschedulable bool) (*http.Response, error) {
	httpResp, err := r.client.patchJSON(
		ctx,
		fmt.Sprintf("/runners/%s/scheduling", url.PathEscape(runnerID)),
		map[string]bool{"unschedulable": unschedulable},
		nil,
	)
	if err != nil {
		return httpResp, err
	}
	return httpResp, nil
}

func (r *RunnerResource) updateRunnerDraining(ctx context.Context, runnerID string, draining bool) (*http.Response, error) {
	httpResp, err := r.client.patchJSON(
		ctx,
		fmt.Sprintf("/runners/%s/draining", url.PathEscape(runnerID)),
		map[string]bool{"draining": draining},
		nil,
	)
	if err != nil {
		return httpResp, err
	}
	return httpResp, nil
}

func configuredBool(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown()
}
