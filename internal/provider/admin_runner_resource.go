package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AdminRunnerResource{}
var _ resource.ResourceWithImportState = &AdminRunnerResource{}

func NewAdminRunnerResource() resource.Resource {
	return &AdminRunnerResource{}
}

type AdminRunnerResource struct {
	client *daytonaClient
}

type adminRunnerResourceModel struct {
	ID                           types.String  `tfsdk:"id"`
	RegionID                     types.String  `tfsdk:"region_id"`
	Name                         types.String  `tfsdk:"name"`
	Tags                         types.List    `tfsdk:"tags"`
	APIKey                       types.String  `tfsdk:"api_key"`
	APIVersion                   types.String  `tfsdk:"api_version"`
	Domain                       types.String  `tfsdk:"domain"`
	APIURL                       types.String  `tfsdk:"api_url"`
	ProxyURL                     types.String  `tfsdk:"proxy_url"`
	CPU                          types.Float64 `tfsdk:"cpu"`
	MemoryGiB                    types.Float64 `tfsdk:"memory_gib"`
	DiskGiB                      types.Float64 `tfsdk:"disk_gib"`
	GPU                          types.String  `tfsdk:"gpu"`
	GPUType                      types.String  `tfsdk:"gpu_type"`
	SandboxClass                 types.String  `tfsdk:"sandbox_class"`
	CurrentCPUUsagePercentage    types.Float64 `tfsdk:"current_cpu_usage_percentage"`
	CurrentMemoryUsagePercentage types.Float64 `tfsdk:"current_memory_usage_percentage"`
	CurrentDiskUsagePercentage   types.Float64 `tfsdk:"current_disk_usage_percentage"`
	CurrentAllocatedCPU          types.Float64 `tfsdk:"current_allocated_cpu"`
	CurrentAllocatedMemoryGiB    types.Float64 `tfsdk:"current_allocated_memory_gib"`
	CurrentAllocatedDiskGiB      types.Float64 `tfsdk:"current_allocated_disk_gib"`
	CurrentSnapshotCount         types.Float64 `tfsdk:"current_snapshot_count"`
	CurrentStartedSandboxes      types.Float64 `tfsdk:"current_started_sandboxes"`
	AvailabilityScore            types.Float64 `tfsdk:"availability_score"`
	Region                       types.String  `tfsdk:"region"`
	State                        types.String  `tfsdk:"state"`
	LastChecked                  types.String  `tfsdk:"last_checked"`
	Unschedulable                types.Bool    `tfsdk:"unschedulable"`
	CreatedAt                    types.String  `tfsdk:"created_at"`
	UpdatedAt                    types.String  `tfsdk:"updated_at"`
	Version                      types.String  `tfsdk:"version"`
	RunnerClass                  types.String  `tfsdk:"runner_class"`
	AppVersion                   types.String  `tfsdk:"app_version"`
	RegionType                   types.String  `tfsdk:"region_type"`
}

func (r *AdminRunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_runner"
}

func (r *AdminRunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona runner using Daytona admin APIs.",
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
				MarkdownDescription: "Daytona region ID where the runner is registered.",
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
			"api_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Runner API key to register. Daytona returns this value to admin reads, and Terraform stores it as sensitive state.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Runner API version. Daytona currently accepts `0` and `2`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain":                          optionalComputedReplaceStringAttribute("Runner domain."),
			"api_url":                         optionalComputedReplaceStringAttribute("Runner API URL."),
			"proxy_url":                       optionalComputedReplaceStringAttribute("Runner proxy URL."),
			"cpu":                             optionalComputedReplaceFloat64Attribute("Runner CPU capacity."),
			"memory_gib":                      optionalComputedReplaceFloat64Attribute("Runner memory capacity in GiB."),
			"disk_gib":                        optionalComputedReplaceFloat64Attribute("Runner disk capacity in GiB."),
			"gpu":                             computedStringAttribute("Runner GPU capacity, when available."),
			"gpu_type":                        computedStringAttribute("Runner GPU type, when available."),
			"sandbox_class":                   computedStringAttribute("Sandbox class supported by the runner."),
			"current_cpu_usage_percentage":    computedFloat64Attribute("Current runner CPU usage percentage."),
			"current_memory_usage_percentage": computedFloat64Attribute("Current runner memory usage percentage."),
			"current_disk_usage_percentage":   computedFloat64Attribute("Current runner disk usage percentage."),
			"current_allocated_cpu":           computedFloat64Attribute("Current allocated CPU."),
			"current_allocated_memory_gib":    computedFloat64Attribute("Current allocated memory in GiB."),
			"current_allocated_disk_gib":      computedFloat64Attribute("Current allocated disk in GiB."),
			"current_snapshot_count":          computedFloat64Attribute("Current snapshot count."),
			"current_started_sandboxes":       computedFloat64Attribute("Current number of started sandboxes."),
			"availability_score":              computedFloat64Attribute("Runner availability score."),
			"region":                          computedStringAttribute("Runner region name."),
			"state":                           computedStringAttribute("Current runner state."),
			"last_checked":                    computedStringAttribute("Last runner health-check timestamp."),
			"created_at":                      computedStringAttribute("Runner creation timestamp."),
			"updated_at":                      computedStringAttribute("Runner update timestamp."),
			"version":                         computedStringAttribute("Runner version."),
			"runner_class":                    computedStringAttribute("Runner class."),
			"app_version":                     computedStringAttribute("Runner app version."),
			"region_type":                     computedStringAttribute("Runner region type."),
			"unschedulable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Daytona should stop scheduling new work on the runner.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func optionalComputedReplaceFloat64Attribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.Float64{
			float64planmodifier.UseStateForUnknown(),
			float64planmodifier.RequiresReplace(),
		},
	}
}

func (r *AdminRunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client := configureResourceDaytonaClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	r.client = client
}

func (r *AdminRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data adminRunnerResourceModel
	var config adminRunnerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, httpResp, err := r.createAdminRunner(ctx, data)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona admin runner", "create admin runner", httpResp, err)
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Empty Daytona admin runner response", "Daytona returned a successful runner create response without a response body.")
		return
	}

	data.ID = types.StringValue(created.Id)
	if strings.TrimSpace(created.ApiKey) != "" {
		data.APIKey = types.StringValue(created.ApiKey)
	}

	// Persist the runner before follow-up calls so a failure cannot orphan it.
	nullUnknownModelValues(ctx, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner, httpResp, err := r.client.api.AdminAPI.AdminGetRunnerById(ctx, created.Id).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read created Daytona admin runner", "read admin runner", httpResp, err)
		return
	}

	httpResp, err = r.applyAdminRunnerOperationalSettings(ctx, created.Id, config, runner)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona admin runner operational settings", "update admin runner operational settings", httpResp, err)
		return
	}

	runner, httpResp, err = r.client.api.AdminAPI.AdminGetRunnerById(ctx, created.Id).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read created Daytona admin runner", "read admin runner", httpResp, err)
		return
	}

	data = flattenAdminRunner(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AdminRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data adminRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner, httpResp, err := r.client.api.AdminAPI.AdminGetRunnerById(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin runner", "read admin runner", httpResp, err)
		return
	}

	data = flattenAdminRunner(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AdminRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config adminRunnerResourceModel
	var plan adminRunnerResourceModel
	var state adminRunnerResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.applyAdminRunnerOperationalSettings(ctx, state.ID.ValueString(), config, nil)
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona admin runner operational settings", "update admin runner operational settings", httpResp, err)
		return
	}

	runner, httpResp, err := r.client.api.AdminAPI.AdminGetRunnerById(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin runner", "read admin runner", httpResp, err)
		return
	}

	plan = flattenAdminRunner(ctx, runner, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AdminRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data adminRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.AdminAPI.AdminDeleteRunner(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona admin runner", "delete admin runner", httpResp, err)
		return
	}
}

func (r *AdminRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AdminRunnerResource) createAdminRunner(ctx context.Context, data adminRunnerResourceModel) (*apiclient.CreateRunnerResponse, *http.Response, error) {
	createRunner, err := adminCreateRunnerPayload(ctx, data)
	if err != nil {
		return nil, nil, err
	}

	return r.client.api.AdminAPI.AdminCreateRunner(ctx).AdminCreateRunner(*createRunner).Execute()
}

func adminCreateRunnerPayload(ctx context.Context, data adminRunnerResourceModel) (*apiclient.AdminCreateRunner, error) {
	apiVersion := strings.TrimSpace(data.APIVersion.ValueString())
	switch apiVersion {
	case "0", "2":
	default:
		return nil, fmt.Errorf("api_version must be either 0 or 2")
	}

	createRunner := apiclient.NewAdminCreateRunner(
		strings.TrimSpace(data.RegionID.ValueString()),
		strings.TrimSpace(data.Name.ValueString()),
		data.APIKey.ValueString(),
		apiVersion,
	)

	tags, diags := stringList(ctx, data.Tags)
	if diags.HasError() {
		return nil, fmt.Errorf("invalid runner tags")
	}
	if len(tags) > 0 {
		createRunner.SetTags(tags)
	}
	if configuredString(data.Domain) {
		createRunner.SetDomain(strings.TrimSpace(data.Domain.ValueString()))
	}
	if configuredString(data.APIURL) {
		createRunner.SetApiUrl(strings.TrimSpace(data.APIURL.ValueString()))
	}
	if configuredString(data.ProxyURL) {
		createRunner.SetProxyUrl(strings.TrimSpace(data.ProxyURL.ValueString()))
	}
	if configuredFloat64(data.CPU) {
		createRunner.SetCpu(float32(data.CPU.ValueFloat64()))
	}
	if configuredFloat64(data.MemoryGiB) {
		createRunner.SetMemoryGiB(float32(data.MemoryGiB.ValueFloat64()))
	}
	if configuredFloat64(data.DiskGiB) {
		createRunner.SetDiskGiB(float32(data.DiskGiB.ValueFloat64()))
	}

	return createRunner, nil
}

func (r *AdminRunnerResource) applyAdminRunnerOperationalSettings(ctx context.Context, runnerID string, config adminRunnerResourceModel, current *apiclient.RunnerFull) (*http.Response, error) {
	if configuredBool(config.Unschedulable) {
		unschedulable := config.Unschedulable.ValueBool()
		if current == nil || current.Unschedulable != unschedulable {
			return r.updateAdminRunnerScheduling(ctx, runnerID, unschedulable)
		}
	}

	return nil, nil
}

func (r *AdminRunnerResource) updateAdminRunnerScheduling(ctx context.Context, runnerID string, unschedulable bool) (*http.Response, error) {
	return r.client.patchJSON(
		ctx,
		fmt.Sprintf("/admin/runners/%s/scheduling", url.PathEscape(runnerID)),
		map[string]bool{"unschedulable": unschedulable},
	)
}

func flattenAdminRunner(ctx context.Context, runner *apiclient.RunnerFull, prior adminRunnerResourceModel) adminRunnerResourceModel {
	if runner == nil {
		return prior
	}

	prior.ID = types.StringValue(runner.Id)
	prior.Name = types.StringValue(runner.Name)
	if strings.TrimSpace(runner.ApiKey) != "" {
		prior.APIKey = types.StringValue(runner.ApiKey)
	}
	prior.APIVersion = types.StringValue(runner.ApiVersion)
	prior.Domain = stringPointerValue(runner.Domain)
	prior.APIURL = stringPointerValue(runner.ApiUrl)
	prior.ProxyURL = stringPointerValue(runner.ProxyUrl)
	prior.CPU = types.Float64Value(float64(runner.Cpu))
	prior.MemoryGiB = types.Float64Value(float64(runner.Memory))
	prior.DiskGiB = types.Float64Value(float64(runner.Disk))
	prior.GPU = float32PointerStringValue(runner.Gpu)
	prior.GPUType = stringPointerValue(runner.GpuType)
	prior.SandboxClass = sandboxClassPointerValue(runner.SandboxClass)
	prior.CurrentCPUUsagePercentage = nullableFloat32Pointer(runner.CurrentCpuUsagePercentage)
	prior.CurrentMemoryUsagePercentage = nullableFloat32Pointer(runner.CurrentMemoryUsagePercentage)
	prior.CurrentDiskUsagePercentage = nullableFloat32Pointer(runner.CurrentDiskUsagePercentage)
	prior.CurrentAllocatedCPU = nullableFloat32Pointer(runner.CurrentAllocatedCpu)
	prior.CurrentAllocatedMemoryGiB = nullableFloat32Pointer(runner.CurrentAllocatedMemoryGiB)
	prior.CurrentAllocatedDiskGiB = nullableFloat32Pointer(runner.CurrentAllocatedDiskGiB)
	prior.CurrentSnapshotCount = nullableFloat32Pointer(runner.CurrentSnapshotCount)
	prior.CurrentStartedSandboxes = nullableFloat32Pointer(runner.CurrentStartedSandboxes)
	prior.AvailabilityScore = nullableFloat32Pointer(runner.AvailabilityScore)
	prior.Region = types.StringValue(runner.Region)
	prior.State = types.StringValue(string(runner.State))
	prior.LastChecked = stringPointerValue(runner.LastChecked)
	prior.Unschedulable = types.BoolValue(runner.Unschedulable)
	prior.Tags = listStringValue(ctx, runner.Tags)
	prior.CreatedAt = types.StringValue(runner.CreatedAt)
	prior.UpdatedAt = types.StringValue(runner.UpdatedAt)
	prior.Version = types.StringValue(runner.Version)
	prior.RunnerClass = types.StringValue(string(runner.RunnerClass))
	prior.AppVersion = stringPointerValue(runner.AppVersion)
	prior.RegionType = regionTypePointerValue(runner.RegionType)

	return prior
}

func configuredString(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) != ""
}

func configuredFloat64(value types.Float64) bool {
	return !value.IsNull() && !value.IsUnknown()
}
