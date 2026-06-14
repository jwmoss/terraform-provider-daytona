package provider

import (
	"context"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RunnerFullDataSource{}
var _ datasource.DataSource = &RunnerForSandboxDataSource{}
var _ datasource.DataSource = &RunnersBySnapshotRefDataSource{}

func NewRunnerFullDataSource() datasource.DataSource {
	return &RunnerFullDataSource{}
}

func NewRunnerForSandboxDataSource() datasource.DataSource {
	return &RunnerForSandboxDataSource{}
}

func NewRunnersBySnapshotRefDataSource() datasource.DataSource {
	return &RunnersBySnapshotRefDataSource{}
}

type RunnerFullDataSource struct {
	client *daytonaClient
}

type RunnerForSandboxDataSource struct {
	client *daytonaClient
}

type RunnersBySnapshotRefDataSource struct {
	client *daytonaClient
}

type runnerFullDataSourceModel struct {
	ID                           types.String  `tfsdk:"id"`
	SandboxID                    types.String  `tfsdk:"sandbox_id"`
	Domain                       types.String  `tfsdk:"domain"`
	APIURL                       types.String  `tfsdk:"api_url"`
	ProxyURL                     types.String  `tfsdk:"proxy_url"`
	CPU                          types.String  `tfsdk:"cpu"`
	Memory                       types.String  `tfsdk:"memory"`
	Disk                         types.String  `tfsdk:"disk"`
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
	Name                         types.String  `tfsdk:"name"`
	State                        types.String  `tfsdk:"state"`
	LastChecked                  types.String  `tfsdk:"last_checked"`
	Unschedulable                types.Bool    `tfsdk:"unschedulable"`
	Tags                         types.List    `tfsdk:"tags"`
	CreatedAt                    types.String  `tfsdk:"created_at"`
	UpdatedAt                    types.String  `tfsdk:"updated_at"`
	Version                      types.String  `tfsdk:"version"`
	APIVersion                   types.String  `tfsdk:"api_version"`
	RunnerClass                  types.String  `tfsdk:"runner_class"`
	AppVersion                   types.String  `tfsdk:"app_version"`
	APIKey                       types.String  `tfsdk:"api_key"`
	RegionType                   types.String  `tfsdk:"region_type"`
}

type runnerFullDataSourceConfigModel struct {
	ID types.String `tfsdk:"id"`
}

type runnerForSandboxDataSourceConfigModel struct {
	SandboxID types.String `tfsdk:"sandbox_id"`
}

type runnersBySnapshotRefDataSourceModel struct {
	ID    types.String                 `tfsdk:"id"`
	Ref   types.String                 `tfsdk:"ref"`
	Items []runnerSnapshotRefItemModel `tfsdk:"items"`
}

type runnersBySnapshotRefDataSourceConfigModel struct {
	Ref types.String `tfsdk:"ref"`
}

type runnerSnapshotRefItemModel struct {
	RunnerSnapshotID types.String `tfsdk:"runner_snapshot_id"`
	RunnerID         types.String `tfsdk:"runner_id"`
	RunnerDomain     types.String `tfsdk:"runner_domain"`
}

func (d *RunnerFullDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_full"
}

func (d *RunnerFullDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads full Daytona runner details by runner ID through Daytona runner/proxy-authenticated routes.",
		Attributes:          runnerFullDataSourceAttributes("id"),
	}
}

func (d *RunnerFullDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RunnerFullDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnerFullDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runnerID := strings.TrimSpace(config.ID.ValueString())
	if runnerID == "" {
		resp.Diagnostics.AddError("Missing Daytona runner ID", "Configure the id attribute with the Daytona runner ID to read.")
		return
	}

	runner, httpResp, err := d.client.api.RunnersAPI.GetRunnerFullById(ctx, runnerID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read full Daytona runner", "read full runner", httpResp, err)
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Empty Daytona runner response", fmt.Sprintf("Daytona returned a successful response without runner %q.", runnerID))
		return
	}

	data := runnerFullDataSourceModel{ID: config.ID}
	data = flattenRunnerFullDataSource(ctx, runner, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *RunnerForSandboxDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_for_sandbox"
}

func (d *RunnerForSandboxDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads full Daytona runner details for a sandbox through Daytona runner/proxy-authenticated routes.",
		Attributes:          runnerFullDataSourceAttributes("sandbox_id"),
	}
}

func (d *RunnerForSandboxDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RunnerForSandboxDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnerForSandboxDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxID := strings.TrimSpace(config.SandboxID.ValueString())
	if sandboxID == "" {
		resp.Diagnostics.AddError("Missing Daytona sandbox ID", "Configure the sandbox_id attribute with the Daytona sandbox ID.")
		return
	}

	runner, httpResp, err := d.client.api.RunnersAPI.GetRunnerBySandboxId(ctx, sandboxID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona runner for sandbox", "read runner for sandbox", httpResp, err)
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Empty Daytona runner response", fmt.Sprintf("Daytona returned a successful response without runner data for sandbox %q.", sandboxID))
		return
	}

	data := runnerFullDataSourceModel{SandboxID: config.SandboxID}
	data = flattenRunnerFullDataSource(ctx, runner, data)
	data.SandboxID = types.StringValue(sandboxID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *RunnersBySnapshotRefDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runners_by_snapshot_ref"
}

func (d *RunnersBySnapshotRefDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Daytona runner snapshot mappings by snapshot ref through Daytona runner/proxy-authenticated routes.",
		Attributes: map[string]schema.Attribute{
			"id":  computedDataSourceStringAttribute("Data source identifier."),
			"ref": requiredDataSourceStringAttribute("Snapshot ref."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Runner snapshot mappings for the snapshot ref.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"runner_snapshot_id": computedDataSourceStringAttribute("Runner snapshot ID."),
						"runner_id":          computedDataSourceStringAttribute("Runner ID."),
						"runner_domain":      computedDataSourceStringAttribute("Runner domain, when available."),
					},
				},
			},
		},
	}
}

func (d *RunnersBySnapshotRefDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RunnersBySnapshotRefDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnersBySnapshotRefDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ref := strings.TrimSpace(config.Ref.ValueString())
	if ref == "" {
		resp.Diagnostics.AddError("Missing Daytona snapshot ref", "Configure the ref attribute with the Daytona snapshot ref to read.")
		return
	}

	runners, httpResp, err := d.client.api.RunnersAPI.GetRunnersBySnapshotRef(ctx).Ref(ref).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona runners by snapshot ref", "list runners by snapshot ref", httpResp, err)
		return
	}

	data := runnersBySnapshotRefDataSourceModel{
		ID:    types.StringValue(ref),
		Ref:   types.StringValue(ref),
		Items: flattenRunnerSnapshotRefItems(runners),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func runnerFullDataSourceAttributes(input string) map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{
		"id":                              computedDataSourceStringAttribute("Daytona runner ID."),
		"sandbox_id":                      computedDataSourceStringAttribute("Daytona sandbox ID used to locate the runner, when applicable."),
		"domain":                          computedDataSourceStringAttribute("Runner domain."),
		"api_url":                         computedDataSourceStringAttribute("Runner API URL."),
		"proxy_url":                       computedDataSourceStringAttribute("Runner proxy URL."),
		"cpu":                             computedDataSourceStringAttribute("Runner CPU capacity."),
		"memory":                          computedDataSourceStringAttribute("Runner memory capacity in GiB."),
		"disk":                            computedDataSourceStringAttribute("Runner disk capacity in GiB."),
		"gpu":                             computedDataSourceStringAttribute("Runner GPU capacity, when available."),
		"gpu_type":                        computedDataSourceStringAttribute("Runner GPU type, when available."),
		"sandbox_class":                   computedDataSourceStringAttribute("Sandbox class supported by the runner."),
		"current_cpu_usage_percentage":    computedDataSourceFloat64Attribute("Current runner CPU usage percentage."),
		"current_memory_usage_percentage": computedDataSourceFloat64Attribute("Current runner memory usage percentage."),
		"current_disk_usage_percentage":   computedDataSourceFloat64Attribute("Current runner disk usage percentage."),
		"current_allocated_cpu":           computedDataSourceFloat64Attribute("Current allocated CPU."),
		"current_allocated_memory_gib":    computedDataSourceFloat64Attribute("Current allocated memory in GiB."),
		"current_allocated_disk_gib":      computedDataSourceFloat64Attribute("Current allocated disk in GiB."),
		"current_snapshot_count":          computedDataSourceFloat64Attribute("Current snapshot count."),
		"current_started_sandboxes":       computedDataSourceFloat64Attribute("Current number of started sandboxes."),
		"availability_score":              computedDataSourceFloat64Attribute("Runner availability score."),
		"region":                          computedDataSourceStringAttribute("Runner region name."),
		"name":                            computedDataSourceStringAttribute("Runner name."),
		"state":                           computedDataSourceStringAttribute("Current runner state."),
		"last_checked":                    computedDataSourceStringAttribute("Last runner health-check timestamp."),
		"unschedulable":                   computedDataSourceBoolAttribute("Whether the runner is unschedulable."),
		"tags":                            computedDataSourceStringListAttribute("Tags associated with the runner."),
		"created_at":                      computedDataSourceStringAttribute("Runner creation timestamp."),
		"updated_at":                      computedDataSourceStringAttribute("Runner update timestamp."),
		"version":                         computedDataSourceStringAttribute("Runner version."),
		"api_version":                     computedDataSourceStringAttribute("Runner API version."),
		"runner_class":                    computedDataSourceStringAttribute("Runner class."),
		"app_version":                     computedDataSourceStringAttribute("Runner app version."),
		"api_key":                         sensitiveComputedDataSourceStringAttribute("Runner API key."),
		"region_type":                     computedDataSourceStringAttribute("Runner region type."),
	}

	switch input {
	case "id":
		attributes["id"] = requiredDataSourceStringAttribute("Daytona runner ID.")
	case "sandbox_id":
		attributes["sandbox_id"] = requiredDataSourceStringAttribute("Daytona sandbox ID used to locate the runner.")
	}

	return attributes
}

func flattenRunnerFullDataSource(ctx context.Context, runner *apiclient.RunnerFull, prior runnerFullDataSourceModel) runnerFullDataSourceModel {
	prior.ID = types.StringValue(runner.Id)
	prior.Domain = stringPointerValue(runner.Domain)
	prior.APIURL = stringPointerValue(runner.ApiUrl)
	prior.ProxyURL = stringPointerValue(runner.ProxyUrl)
	prior.CPU = types.StringValue(fmt.Sprintf("%g", runner.Cpu))
	prior.Memory = types.StringValue(fmt.Sprintf("%g", runner.Memory))
	prior.Disk = types.StringValue(fmt.Sprintf("%g", runner.Disk))
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
	prior.Name = types.StringValue(runner.Name)
	prior.State = types.StringValue(string(runner.State))
	prior.LastChecked = stringPointerValue(runner.LastChecked)
	prior.Unschedulable = types.BoolValue(runner.Unschedulable)
	prior.Tags = listStringValue(ctx, runner.Tags)
	prior.CreatedAt = types.StringValue(runner.CreatedAt)
	prior.UpdatedAt = types.StringValue(runner.UpdatedAt)
	prior.Version = types.StringValue(runner.Version)
	prior.APIVersion = types.StringValue(runner.ApiVersion)
	prior.RunnerClass = types.StringValue(string(runner.RunnerClass))
	prior.AppVersion = stringPointerValue(runner.AppVersion)
	prior.APIKey = types.StringValue(runner.ApiKey)
	prior.RegionType = regionTypePointerValue(runner.RegionType)

	return prior
}

func flattenRunnerSnapshotRefItems(runners []apiclient.RunnerSnapshotDto) []runnerSnapshotRefItemModel {
	items := make([]runnerSnapshotRefItemModel, 0, len(runners))
	for _, runner := range runners {
		items = append(items, runnerSnapshotRefItemModel{
			RunnerSnapshotID: types.StringValue(runner.RunnerSnapshotId),
			RunnerID:         types.StringValue(runner.RunnerId),
			RunnerDomain:     stringPointerValue(runner.RunnerDomain),
		})
	}
	return items
}

func stringPointerValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func float32PointerStringValue(value *float32) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(fmt.Sprintf("%g", *value))
}

func sandboxClassPointerValue(value *apiclient.SandboxClass) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}

func regionTypePointerValue(value *apiclient.RegionType) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}
