package provider

import (
	"context"
	"fmt"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeDataSource{}
var _ datasource.DataSource = &DockerRegistryDataSource{}
var _ datasource.DataSource = &RegionDataSource{}
var _ datasource.DataSource = &SandboxDataSource{}
var _ datasource.DataSource = &SnapshotDataSource{}
var _ datasource.DataSource = &RunnerDataSource{}
var _ datasource.DataSource = &OrganizationDataSource{}

func NewVolumeDataSource() datasource.DataSource {
	return &VolumeDataSource{}
}

func NewDockerRegistryDataSource() datasource.DataSource {
	return &DockerRegistryDataSource{}
}

func NewRegionDataSource() datasource.DataSource {
	return &RegionDataSource{}
}

func NewSandboxDataSource() datasource.DataSource {
	return &SandboxDataSource{}
}

func NewSnapshotDataSource() datasource.DataSource {
	return &SnapshotDataSource{}
}

func NewRunnerDataSource() datasource.DataSource {
	return &RunnerDataSource{}
}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

type VolumeDataSource struct {
	client *daytonaClient
}

type volumeDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	State          types.String `tfsdk:"state"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	LastUsedAt     types.String `tfsdk:"last_used_at"`
	ErrorReason    types.String `tfsdk:"error_reason"`
}

func (d *VolumeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (d *VolumeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona persistent volume by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              requiredDataSourceStringAttribute("Daytona volume ID."),
			"name":            computedDataSourceStringAttribute("Volume name."),
			"organization_id": computedDataSourceStringAttribute("Daytona organization ID that owns the volume."),
			"state":           computedDataSourceStringAttribute("Current volume state."),
			"created_at":      computedDataSourceStringAttribute("Volume creation timestamp."),
			"updated_at":      computedDataSourceStringAttribute("Volume update timestamp."),
			"last_used_at":    computedDataSourceStringAttribute("Volume last-used timestamp, when available."),
			"error_reason":    computedDataSourceStringAttribute("Volume error reason, when available."),
		},
	}
}

func (d *VolumeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *VolumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, httpResp, err := d.client.api.VolumesAPI.GetVolume(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona volume", "read volume", httpResp, err)
		return
	}
	if volume == nil {
		resp.Diagnostics.AddError("Empty Daytona volume response", fmt.Sprintf("Daytona returned a successful response without volume %q.", data.ID.ValueString()))
		return
	}

	data = flattenVolumeDataSource(volume)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type DockerRegistryDataSource struct {
	client *daytonaClient
}

type dockerRegistryDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Username     types.String `tfsdk:"username"`
	Project      types.String `tfsdk:"project"`
	RegistryType types.String `tfsdk:"registry_type"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (d *DockerRegistryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_registry"
}

func (d *DockerRegistryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona Docker registry by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            requiredDataSourceStringAttribute("Daytona Docker registry ID."),
			"name":          computedDataSourceStringAttribute("Registry name."),
			"url":           computedDataSourceStringAttribute("Registry URL."),
			"username":      computedDataSourceStringAttribute("Registry username."),
			"project":       computedDataSourceStringAttribute("Registry project or namespace."),
			"registry_type": computedDataSourceStringAttribute("Registry type."),
			"created_at":    computedDataSourceStringAttribute("Registry creation timestamp."),
			"updated_at":    computedDataSourceStringAttribute("Registry update timestamp."),
		},
	}
}

func (d *DockerRegistryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *DockerRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dockerRegistryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, httpResp, err := d.client.api.DockerRegistryAPI.GetRegistry(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona Docker registry", "read Docker registry", httpResp, err)
		return
	}
	if registry == nil {
		resp.Diagnostics.AddError("Empty Daytona Docker registry response", fmt.Sprintf("Daytona returned a successful response without Docker registry %q.", data.ID.ValueString()))
		return
	}

	data = flattenDockerRegistryDataSource(registry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type RegionDataSource struct {
	client *daytonaClient
}

type regionDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	RegionType         types.String `tfsdk:"region_type"`
	ProxyURL           types.String `tfsdk:"proxy_url"`
	SSHGatewayURL      types.String `tfsdk:"ssh_gateway_url"`
	SnapshotManagerURL types.String `tfsdk:"snapshot_manager_url"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (d *RegionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region"
}

func (d *RegionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona region by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                   requiredDataSourceStringAttribute("Daytona region ID."),
			"name":                 computedDataSourceStringAttribute("Region name."),
			"organization_id":      computedDataSourceStringAttribute("Daytona organization ID that owns the region."),
			"region_type":          computedDataSourceStringAttribute("Region type."),
			"proxy_url":            computedDataSourceStringAttribute("Proxy URL for the region."),
			"ssh_gateway_url":      computedDataSourceStringAttribute("SSH gateway URL for the region."),
			"snapshot_manager_url": computedDataSourceStringAttribute("Snapshot manager URL for the region."),
			"created_at":           computedDataSourceStringAttribute("Region creation timestamp."),
			"updated_at":           computedDataSourceStringAttribute("Region update timestamp."),
		},
	}
}

func (d *RegionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data regionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	region, httpResp, err := d.client.api.OrganizationsAPI.GetRegionById(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona region", "read region", httpResp, err)
		return
	}
	if region == nil {
		resp.Diagnostics.AddError("Empty Daytona region response", fmt.Sprintf("Daytona returned a successful response without region %q.", data.ID.ValueString()))
		return
	}

	data = flattenRegionDataSource(region)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxDataSource struct {
	client *daytonaClient
}

type sandboxDataSourceModel struct {
	SandboxIDOrName     types.String `tfsdk:"sandbox_id_or_name"`
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	Snapshot            types.String `tfsdk:"snapshot"`
	User                types.String `tfsdk:"user"`
	Env                 types.Map    `tfsdk:"env"`
	Labels              types.Map    `tfsdk:"labels"`
	Public              types.Bool   `tfsdk:"public"`
	NetworkBlockAll     types.Bool   `tfsdk:"network_block_all"`
	NetworkAllowList    types.String `tfsdk:"network_allow_list"`
	Target              types.String `tfsdk:"target"`
	CPU                 types.Int64  `tfsdk:"cpu"`
	GPU                 types.Int64  `tfsdk:"gpu"`
	Memory              types.Int64  `tfsdk:"memory"`
	Disk                types.Int64  `tfsdk:"disk"`
	AutoStopInterval    types.Int64  `tfsdk:"auto_stop_interval"`
	AutoArchiveInterval types.Int64  `tfsdk:"auto_archive_interval"`
	AutoDeleteInterval  types.Int64  `tfsdk:"auto_delete_interval"`
	State               types.String `tfsdk:"state"`
	RunnerID            types.String `tfsdk:"runner_id"`
	ToolboxProxyURL     types.String `tfsdk:"toolbox_proxy_url"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	ErrorReason         types.String `tfsdk:"error_reason"`
}

func (d *SandboxDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox"
}

func (d *SandboxDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona sandbox by ID or name.",
		Attributes: map[string]schema.Attribute{
			"sandbox_id_or_name":    requiredDataSourceStringAttribute("Daytona sandbox ID or name."),
			"id":                    computedDataSourceStringAttribute("Daytona sandbox ID."),
			"name":                  computedDataSourceStringAttribute("Sandbox name."),
			"organization_id":       computedDataSourceStringAttribute("Daytona organization ID that owns the sandbox."),
			"snapshot":              computedDataSourceStringAttribute("Snapshot ID or name used to create the sandbox."),
			"user":                  computedDataSourceStringAttribute("User associated with the sandbox project."),
			"env":                   sensitiveComputedDataSourceStringMapAttribute("Environment variables for the sandbox."),
			"labels":                computedDataSourceStringMapAttribute("Labels for the sandbox."),
			"public":                computedDataSourceBoolAttribute("Whether HTTP previews are publicly accessible."),
			"network_block_all":     computedDataSourceBoolAttribute("Whether all sandbox network access is blocked."),
			"network_allow_list":    computedDataSourceStringAttribute("Comma-separated list of allowed CIDR network addresses."),
			"target":                computedDataSourceStringAttribute("Target region where the sandbox is created."),
			"cpu":                   computedDataSourceInt64Attribute("CPU cores allocated to the sandbox."),
			"gpu":                   computedDataSourceInt64Attribute("GPU units allocated to the sandbox."),
			"memory":                computedDataSourceInt64Attribute("Memory allocated to the sandbox in GB."),
			"disk":                  computedDataSourceInt64Attribute("Disk allocated to the sandbox in GB."),
			"auto_stop_interval":    computedDataSourceInt64Attribute("Auto-stop interval in minutes."),
			"auto_archive_interval": computedDataSourceInt64Attribute("Auto-archive interval in minutes."),
			"auto_delete_interval":  computedDataSourceInt64Attribute("Auto-delete interval in minutes."),
			"state":                 computedDataSourceStringAttribute("Current sandbox state."),
			"runner_id":             computedDataSourceStringAttribute("Runner ID hosting the sandbox, when assigned."),
			"toolbox_proxy_url":     computedDataSourceStringAttribute("Toolbox proxy URL for the sandbox."),
			"created_at":            computedDataSourceStringAttribute("Sandbox creation timestamp."),
			"updated_at":            computedDataSourceStringAttribute("Sandbox update timestamp."),
			"error_reason":          computedDataSourceStringAttribute("Sandbox error reason, when available."),
		},
	}
}

func (d *SandboxDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandbox, httpResp, err := d.client.api.SandboxAPI.GetSandbox(ctx, data.SandboxIDOrName.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox", "read sandbox", httpResp, err)
		return
	}
	if sandbox == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox response", fmt.Sprintf("Daytona returned a successful response without sandbox %q.", data.SandboxIDOrName.ValueString()))
		return
	}

	data = flattenSandboxDataSource(ctx, data.SandboxIDOrName.ValueString(), sandbox)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SnapshotDataSource struct {
	client *daytonaClient
}

type snapshotDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ImageName      types.String `tfsdk:"image_name"`
	Entrypoint     types.List   `tfsdk:"entrypoint"`
	CPU            types.Int64  `tfsdk:"cpu"`
	GPU            types.Int64  `tfsdk:"gpu"`
	Memory         types.Int64  `tfsdk:"memory"`
	Disk           types.Int64  `tfsdk:"disk"`
	RegionIDs      types.List   `tfsdk:"region_ids"`
	SandboxClass   types.String `tfsdk:"sandbox_class"`
	State          types.String `tfsdk:"state"`
	Ref            types.String `tfsdk:"ref"`
	General        types.Bool   `tfsdk:"general"`
	Size           types.String `tfsdk:"size"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ErrorReason    types.String `tfsdk:"error_reason"`
}

func (d *SnapshotDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (d *SnapshotDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona snapshot by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              requiredDataSourceStringAttribute("Daytona snapshot ID."),
			"name":            computedDataSourceStringAttribute("Snapshot name."),
			"organization_id": computedDataSourceStringAttribute("Daytona organization ID that owns the snapshot."),
			"image_name":      computedDataSourceStringAttribute("Image name used to build the snapshot."),
			"entrypoint":      computedDataSourceStringListAttribute("Entrypoint command for the snapshot."),
			"cpu":             computedDataSourceInt64Attribute("CPU cores allocated to sandboxes created from this snapshot."),
			"gpu":             computedDataSourceInt64Attribute("GPU units allocated to sandboxes created from this snapshot."),
			"memory":          computedDataSourceInt64Attribute("Memory allocated to sandboxes created from this snapshot in GB."),
			"disk":            computedDataSourceInt64Attribute("Disk allocated to sandboxes created from this snapshot in GB."),
			"region_ids":      computedDataSourceStringListAttribute("Region IDs where the snapshot is available."),
			"sandbox_class":   computedDataSourceStringAttribute("Sandbox class for sandboxes created from this snapshot."),
			"state":           computedDataSourceStringAttribute("Current snapshot state."),
			"ref":             computedDataSourceStringAttribute("Snapshot reference."),
			"general":         computedDataSourceBoolAttribute("Whether this is a general Daytona snapshot."),
			"size":            computedDataSourceStringAttribute("Snapshot size, when available."),
			"created_at":      computedDataSourceStringAttribute("Snapshot creation timestamp."),
			"updated_at":      computedDataSourceStringAttribute("Snapshot update timestamp."),
			"error_reason":    computedDataSourceStringAttribute("Snapshot error reason, when available."),
		},
	}
}

func (d *SnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data snapshotDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, httpResp, err := d.client.api.SnapshotsAPI.GetSnapshot(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona snapshot", "read snapshot", httpResp, err)
		return
	}
	if snapshot == nil {
		resp.Diagnostics.AddError("Empty Daytona snapshot response", fmt.Sprintf("Daytona returned a successful response without snapshot %q.", data.ID.ValueString()))
		return
	}

	data = flattenSnapshotDataSource(ctx, snapshot)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type RunnerDataSource struct {
	client *daytonaClient
}

type runnerDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Tags          types.List   `tfsdk:"tags"`
	Region        types.String `tfsdk:"region"`
	State         types.String `tfsdk:"state"`
	Unschedulable types.Bool   `tfsdk:"unschedulable"`
	CPU           types.String `tfsdk:"cpu"`
	Memory        types.String `tfsdk:"memory"`
	Disk          types.String `tfsdk:"disk"`
	GPU           types.String `tfsdk:"gpu"`
	GPUType       types.String `tfsdk:"gpu_type"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (d *RunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner"
}

func (d *RunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona runner by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            requiredDataSourceStringAttribute("Daytona runner ID."),
			"name":          computedDataSourceStringAttribute("Runner name."),
			"tags":          computedDataSourceStringListAttribute("Tags associated with the runner."),
			"region":        computedDataSourceStringAttribute("Runner region name."),
			"state":         computedDataSourceStringAttribute("Current runner state."),
			"unschedulable": computedDataSourceBoolAttribute("Whether the runner is unschedulable."),
			"cpu":           computedDataSourceStringAttribute("Runner CPU capacity."),
			"memory":        computedDataSourceStringAttribute("Runner memory capacity in GiB."),
			"disk":          computedDataSourceStringAttribute("Runner disk capacity in GiB."),
			"gpu":           computedDataSourceStringAttribute("Runner GPU capacity, when available."),
			"gpu_type":      computedDataSourceStringAttribute("Runner GPU type, when available."),
			"created_at":    computedDataSourceStringAttribute("Runner creation timestamp."),
			"updated_at":    computedDataSourceStringAttribute("Runner update timestamp."),
		},
	}
}

func (d *RunnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *RunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data runnerDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner, httpResp, err := d.client.api.RunnersAPI.GetRunnerById(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona runner", "read runner", httpResp, err)
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Empty Daytona runner response", fmt.Sprintf("Daytona returned a successful response without runner %q.", data.ID.ValueString()))
		return
	}

	data = flattenRunnerDataSource(ctx, runner)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type OrganizationDataSource struct {
	client *daytonaClient
}

func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona organization by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                                       requiredDataSourceStringAttribute("Daytona organization ID."),
			"name":                                     computedDataSourceStringAttribute("Organization name."),
			"default_region_id":                        computedDataSourceStringAttribute("Default Daytona region ID for the organization."),
			"created_by":                               computedDataSourceStringAttribute("User ID of the organization creator."),
			"personal":                                 computedDataSourceBoolAttribute("Whether this is a personal organization."),
			"suspended":                                computedDataSourceBoolAttribute("Whether the organization is suspended."),
			"suspended_at":                             computedDataSourceStringAttribute("Organization suspension timestamp, when available."),
			"suspension_reason":                        computedDataSourceStringAttribute("Organization suspension reason, when available."),
			"suspended_until":                          computedDataSourceStringAttribute("Suspension end timestamp, when available."),
			"suspension_cleanup_grace_period_hours":    computedDataSourceFloat64Attribute("Suspension cleanup grace period in hours."),
			"max_cpu_per_sandbox":                      computedDataSourceFloat64Attribute("Maximum CPU per sandbox."),
			"max_memory_per_sandbox":                   computedDataSourceFloat64Attribute("Maximum memory per sandbox."),
			"max_disk_per_sandbox":                     computedDataSourceFloat64Attribute("Maximum disk per sandbox."),
			"snapshot_deactivation_timeout_minutes":    computedDataSourceFloat64Attribute("Snapshot deactivation timeout in minutes."),
			"sandbox_limited_network_egress":           computedDataSourceBoolAttribute("Default limited network egress setting for new sandboxes."),
			"authenticated_rate_limit":                 computedDataSourceFloat64Attribute("Authenticated request rate limit per minute."),
			"sandbox_create_rate_limit":                computedDataSourceFloat64Attribute("Sandbox create rate limit per minute."),
			"sandbox_lifecycle_rate_limit":             computedDataSourceFloat64Attribute("Sandbox lifecycle rate limit per minute."),
			"authenticated_rate_limit_ttl_seconds":     computedDataSourceFloat64Attribute("Authenticated request rate-limit TTL in seconds."),
			"sandbox_create_rate_limit_ttl_seconds":    computedDataSourceFloat64Attribute("Sandbox create rate-limit TTL in seconds."),
			"sandbox_lifecycle_rate_limit_ttl_seconds": computedDataSourceFloat64Attribute("Sandbox lifecycle rate-limit TTL in seconds."),
			"experimental_config_json":                 computedDataSourceStringAttribute("Experimental organization configuration as a JSON object string."),
			"created_at":                               computedDataSourceStringAttribute("Organization creation timestamp."),
			"updated_at":                               computedDataSourceStringAttribute("Organization update timestamp."),
		},
	}
}

func (d *OrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization, httpResp, err := d.client.api.OrganizationsAPI.GetOrganization(ctx, data.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona organization", "read organization", httpResp, err)
		return
	}
	if organization == nil {
		resp.Diagnostics.AddError("Empty Daytona organization response", fmt.Sprintf("Daytona returned a successful response without organization %q.", data.ID.ValueString()))
		return
	}

	data = flattenOrganization(organization, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenVolumeDataSource(volume *apiclient.VolumeDto) volumeDataSourceModel {
	data := volumeDataSourceModel{
		ID:             types.StringValue(volume.Id),
		Name:           types.StringValue(volume.Name),
		OrganizationID: types.StringValue(volume.OrganizationId),
		State:          types.StringValue(string(volume.State)),
		CreatedAt:      types.StringValue(volume.CreatedAt),
		UpdatedAt:      types.StringValue(volume.UpdatedAt),
		LastUsedAt:     types.StringNull(),
		ErrorReason:    types.StringNull(),
	}

	if value, ok := volume.GetLastUsedAtOk(); ok && value != nil {
		data.LastUsedAt = types.StringValue(*value)
	}
	if value, ok := volume.GetErrorReasonOk(); ok && value != nil {
		data.ErrorReason = types.StringValue(*value)
	}

	return data
}

func flattenDockerRegistryDataSource(registry *apiclient.DockerRegistry) dockerRegistryDataSourceModel {
	return dockerRegistryDataSourceModel{
		ID:           types.StringValue(registry.Id),
		Name:         types.StringValue(registry.Name),
		URL:          types.StringValue(registry.Url),
		Username:     types.StringValue(registry.Username),
		Project:      types.StringValue(registry.Project),
		RegistryType: types.StringValue(registry.RegistryType),
		CreatedAt:    types.StringValue(registry.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:    types.StringValue(registry.UpdatedAt.Format(time.RFC3339)),
	}
}

func flattenSandboxDataSource(ctx context.Context, idOrName string, sandbox *apiclient.Sandbox) sandboxDataSourceModel {
	data := sandboxDataSourceModel{
		SandboxIDOrName:  types.StringValue(idOrName),
		ID:               types.StringValue(sandbox.Id),
		Name:             types.StringValue(sandbox.Name),
		OrganizationID:   types.StringValue(sandbox.OrganizationId),
		User:             types.StringValue(sandbox.User),
		Public:           types.BoolValue(sandbox.Public),
		NetworkBlockAll:  types.BoolValue(sandbox.NetworkBlockAll),
		Target:           types.StringValue(sandbox.Target),
		CPU:              types.Int64Value(int64(sandbox.Cpu)),
		GPU:              types.Int64Value(int64(sandbox.Gpu)),
		Memory:           types.Int64Value(int64(sandbox.Memory)),
		Disk:             types.Int64Value(int64(sandbox.Disk)),
		ToolboxProxyURL:  types.StringValue(sandbox.ToolboxProxyUrl),
		Env:              stringMapValue(ctx, sandbox.Env),
		Labels:           stringMapValue(ctx, sandbox.Labels),
		Snapshot:         types.StringNull(),
		NetworkAllowList: types.StringNull(),
		State:            types.StringNull(),
		RunnerID:         types.StringNull(),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
		ErrorReason:      types.StringNull(),
	}

	if sandbox.Snapshot != nil {
		data.Snapshot = types.StringValue(*sandbox.Snapshot)
	}
	if sandbox.NetworkAllowList != nil {
		data.NetworkAllowList = types.StringValue(*sandbox.NetworkAllowList)
	}
	if sandbox.State != nil {
		data.State = types.StringValue(string(*sandbox.State))
	}
	if sandbox.RunnerId != nil {
		data.RunnerID = types.StringValue(*sandbox.RunnerId)
	}
	if sandbox.CreatedAt != nil {
		data.CreatedAt = types.StringValue(*sandbox.CreatedAt)
	}
	if sandbox.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(*sandbox.UpdatedAt)
	}
	if sandbox.ErrorReason != nil {
		data.ErrorReason = types.StringValue(*sandbox.ErrorReason)
	}
	if sandbox.AutoStopInterval != nil {
		data.AutoStopInterval = types.Int64Value(int64(*sandbox.AutoStopInterval))
	} else {
		data.AutoStopInterval = types.Int64Null()
	}
	if sandbox.AutoArchiveInterval != nil {
		data.AutoArchiveInterval = types.Int64Value(int64(*sandbox.AutoArchiveInterval))
	} else {
		data.AutoArchiveInterval = types.Int64Null()
	}
	if sandbox.AutoDeleteInterval != nil {
		data.AutoDeleteInterval = types.Int64Value(int64(*sandbox.AutoDeleteInterval))
	} else {
		data.AutoDeleteInterval = types.Int64Null()
	}

	return data
}

func flattenSnapshotDataSource(ctx context.Context, snapshot *apiclient.SnapshotDto) snapshotDataSourceModel {
	data := snapshotDataSourceModel{
		ID:           types.StringValue(snapshot.Id),
		Name:         types.StringValue(snapshot.Name),
		General:      types.BoolValue(snapshot.General),
		State:        types.StringValue(string(snapshot.State)),
		CPU:          types.Int64Value(int64(snapshot.Cpu)),
		GPU:          types.Int64Value(int64(snapshot.Gpu)),
		Memory:       types.Int64Value(int64(snapshot.Mem)),
		Disk:         types.Int64Value(int64(snapshot.Disk)),
		CreatedAt:    types.StringValue(snapshot.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:    types.StringValue(snapshot.UpdatedAt.Format(time.RFC3339)),
		Entrypoint:   listStringValue(ctx, snapshot.Entrypoint),
		RegionIDs:    listStringValue(ctx, snapshot.RegionIds),
		ImageName:    types.StringNull(),
		Ref:          types.StringNull(),
		SandboxClass: types.StringNull(),
		Size:         types.StringNull(),
		ErrorReason:  types.StringNull(),
	}

	if snapshot.OrganizationId != nil {
		data.OrganizationID = types.StringValue(*snapshot.OrganizationId)
	} else {
		data.OrganizationID = types.StringNull()
	}
	if snapshot.ImageName != nil {
		data.ImageName = types.StringValue(*snapshot.ImageName)
	}
	if snapshot.Ref != nil {
		data.Ref = types.StringValue(*snapshot.Ref)
	}
	if snapshot.SandboxClass != nil {
		data.SandboxClass = types.StringValue(*snapshot.SandboxClass)
	}
	if size, ok := snapshot.GetSizeOk(); ok && size != nil {
		data.Size = types.StringValue(fmt.Sprintf("%g", *size))
	}
	if errorReason, ok := snapshot.GetErrorReasonOk(); ok && errorReason != nil {
		data.ErrorReason = types.StringValue(*errorReason)
	}

	return data
}

func flattenRunnerDataSource(ctx context.Context, runner *apiclient.Runner) runnerDataSourceModel {
	data := runnerDataSourceModel{
		ID:            types.StringValue(runner.Id),
		Name:          types.StringValue(runner.Name),
		Tags:          listStringValue(ctx, runner.Tags),
		Region:        types.StringValue(runner.Region),
		State:         types.StringValue(string(runner.State)),
		Unschedulable: types.BoolValue(runner.Unschedulable),
		CPU:           types.StringValue(fmt.Sprintf("%g", runner.Cpu)),
		Memory:        types.StringValue(fmt.Sprintf("%g", runner.Memory)),
		Disk:          types.StringValue(fmt.Sprintf("%g", runner.Disk)),
		CreatedAt:     types.StringValue(runner.CreatedAt),
		UpdatedAt:     types.StringValue(runner.UpdatedAt),
		GPU:           types.StringNull(),
		GPUType:       types.StringNull(),
	}

	if runner.Gpu != nil {
		data.GPU = types.StringValue(fmt.Sprintf("%g", *runner.Gpu))
	}
	if runner.GpuType != nil {
		data.GPUType = types.StringValue(*runner.GpuType)
	}

	return data
}

func computedDataSourceStringMapAttribute(description string) schema.MapAttribute {
	return schema.MapAttribute{
		ElementType:         types.StringType,
		Computed:            true,
		MarkdownDescription: description,
	}
}

func sensitiveComputedDataSourceStringMapAttribute(description string) schema.MapAttribute {
	return schema.MapAttribute{
		ElementType:         types.StringType,
		Computed:            true,
		Sensitive:           true,
		MarkdownDescription: description,
	}
}

func flattenRegionDataSource(region *apiclient.Region) regionDataSourceModel {
	return regionDataSourceModel{
		ID:                 types.StringValue(region.Id),
		Name:               types.StringValue(region.Name),
		OrganizationID:     nullableStringValue(region.OrganizationId),
		RegionType:         types.StringValue(string(region.RegionType)),
		ProxyURL:           nullableStringValue(region.ProxyUrl),
		SSHGatewayURL:      nullableStringValue(region.SshGatewayUrl),
		SnapshotManagerURL: nullableStringValue(region.SnapshotManagerUrl),
		CreatedAt:          types.StringValue(region.CreatedAt),
		UpdatedAt:          types.StringValue(region.UpdatedAt),
	}
}
