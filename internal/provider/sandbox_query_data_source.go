package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SandboxQueryDataSource{}

func NewSandboxQueryDataSource() datasource.DataSource {
	return &SandboxQueryDataSource{}
}

type SandboxQueryDataSource struct {
	client *daytonaClient
}

type sandboxQueryConfigModel struct {
	RequestOrganizationID types.String  `tfsdk:"request_organization_id"`
	Cursor                types.String  `tfsdk:"cursor"`
	Limit                 types.Int64   `tfsdk:"limit"`
	IDPrefix              types.String  `tfsdk:"id_prefix"`
	NamePrefix            types.String  `tfsdk:"name_prefix"`
	LabelsJSON            types.String  `tfsdk:"labels_json"`
	IncludeErroredDeleted types.Bool    `tfsdk:"include_errored_deleted"`
	States                types.List    `tfsdk:"states"`
	Snapshots             types.List    `tfsdk:"snapshots"`
	RegionIDs             types.List    `tfsdk:"region_ids"`
	SandboxClasses        types.List    `tfsdk:"sandbox_classes"`
	MinCPU                types.Float64 `tfsdk:"min_cpu"`
	MaxCPU                types.Float64 `tfsdk:"max_cpu"`
	MinMemoryGiB          types.Float64 `tfsdk:"min_memory_gib"`
	MaxMemoryGiB          types.Float64 `tfsdk:"max_memory_gib"`
	MinDiskGiB            types.Float64 `tfsdk:"min_disk_gib"`
	MaxDiskGiB            types.Float64 `tfsdk:"max_disk_gib"`
	IsPublic              types.Bool    `tfsdk:"is_public"`
	IsRecoverable         types.Bool    `tfsdk:"is_recoverable"`
	CreatedAtAfter        types.String  `tfsdk:"created_at_after"`
	CreatedAtBefore       types.String  `tfsdk:"created_at_before"`
	LastEventAfter        types.String  `tfsdk:"last_event_after"`
	LastEventBefore       types.String  `tfsdk:"last_event_before"`
	Sort                  types.String  `tfsdk:"sort"`
	Order                 types.String  `tfsdk:"order"`
}

type sandboxQueryDataSourceModel struct {
	RequestOrganizationID types.String            `tfsdk:"request_organization_id"`
	Cursor                types.String            `tfsdk:"cursor"`
	Limit                 types.Int64             `tfsdk:"limit"`
	IDPrefix              types.String            `tfsdk:"id_prefix"`
	NamePrefix            types.String            `tfsdk:"name_prefix"`
	LabelsJSON            types.String            `tfsdk:"labels_json"`
	IncludeErroredDeleted types.Bool              `tfsdk:"include_errored_deleted"`
	States                types.List              `tfsdk:"states"`
	Snapshots             types.List              `tfsdk:"snapshots"`
	RegionIDs             types.List              `tfsdk:"region_ids"`
	SandboxClasses        types.List              `tfsdk:"sandbox_classes"`
	MinCPU                types.Float64           `tfsdk:"min_cpu"`
	MaxCPU                types.Float64           `tfsdk:"max_cpu"`
	MinMemoryGiB          types.Float64           `tfsdk:"min_memory_gib"`
	MaxMemoryGiB          types.Float64           `tfsdk:"max_memory_gib"`
	MinDiskGiB            types.Float64           `tfsdk:"min_disk_gib"`
	MaxDiskGiB            types.Float64           `tfsdk:"max_disk_gib"`
	IsPublic              types.Bool              `tfsdk:"is_public"`
	IsRecoverable         types.Bool              `tfsdk:"is_recoverable"`
	CreatedAtAfter        types.String            `tfsdk:"created_at_after"`
	CreatedAtBefore       types.String            `tfsdk:"created_at_before"`
	LastEventAfter        types.String            `tfsdk:"last_event_after"`
	LastEventBefore       types.String            `tfsdk:"last_event_before"`
	Sort                  types.String            `tfsdk:"sort"`
	Order                 types.String            `tfsdk:"order"`
	ID                    types.String            `tfsdk:"id"`
	NextCursor            types.String            `tfsdk:"next_cursor"`
	Items                 []sandboxQueryItemModel `tfsdk:"items"`
}

type sandboxQueryItemModel struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	OrganizationID      types.String  `tfsdk:"organization_id"`
	Target              types.String  `tfsdk:"target"`
	RunnerID            types.String  `tfsdk:"runner_id"`
	SandboxClass        types.String  `tfsdk:"sandbox_class"`
	State               types.String  `tfsdk:"state"`
	DesiredState        types.String  `tfsdk:"desired_state"`
	Snapshot            types.String  `tfsdk:"snapshot"`
	User                types.String  `tfsdk:"user"`
	ErrorReason         types.String  `tfsdk:"error_reason"`
	Recoverable         types.Bool    `tfsdk:"recoverable"`
	Public              types.Bool    `tfsdk:"public"`
	CPU                 types.Float64 `tfsdk:"cpu"`
	GPU                 types.Float64 `tfsdk:"gpu"`
	GPUType             types.String  `tfsdk:"gpu_type"`
	Memory              types.Float64 `tfsdk:"memory"`
	Disk                types.Float64 `tfsdk:"disk"`
	Labels              types.Map     `tfsdk:"labels"`
	BackupState         types.String  `tfsdk:"backup_state"`
	AutoStopInterval    types.Float64 `tfsdk:"auto_stop_interval"`
	AutoArchiveInterval types.Float64 `tfsdk:"auto_archive_interval"`
	AutoDeleteInterval  types.Float64 `tfsdk:"auto_delete_interval"`
	CreatedAt           types.String  `tfsdk:"created_at"`
	UpdatedAt           types.String  `tfsdk:"updated_at"`
	LastActivityAt      types.String  `tfsdk:"last_activity_at"`
	DaemonVersion       types.String  `tfsdk:"daemon_version"`
	ToolboxProxyURL     types.String  `tfsdk:"toolbox_proxy_url"`
}

func (d *SandboxQueryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_query"
}

func (d *SandboxQueryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries Daytona sandboxes with server-side filtering, sorting, and cursor pagination.",
		Attributes: map[string]schema.Attribute{
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"cursor": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Pagination cursor from a previous response.",
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of sandboxes to return. Daytona defaults to 100.",
			},
			"id_prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Case-insensitive sandbox ID prefix filter.",
			},
			"name_prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Case-insensitive sandbox name prefix filter.",
			},
			"labels_json": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "JSON object string of labels to filter by.",
			},
			"include_errored_deleted": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether results with errored state and deleted desired state should be included.",
			},
			"states":          optionalDataSourceStringListAttribute("Sandbox states to include."),
			"snapshots":       optionalDataSourceStringListAttribute("Snapshot names or IDs to include."),
			"region_ids":      optionalDataSourceStringListAttribute("Region IDs to include."),
			"sandbox_classes": optionalDataSourceStringListAttribute("Sandbox classes to include."),
			"min_cpu":         optionalFloat64DataSourceAttribute("Minimum CPU cores."),
			"max_cpu":         optionalFloat64DataSourceAttribute("Maximum CPU cores."),
			"min_memory_gib":  optionalFloat64DataSourceAttribute("Minimum memory in GiB."),
			"max_memory_gib":  optionalFloat64DataSourceAttribute("Maximum memory in GiB."),
			"min_disk_gib":    optionalFloat64DataSourceAttribute("Minimum disk in GiB."),
			"max_disk_gib":    optionalFloat64DataSourceAttribute("Maximum disk in GiB."),
			"is_public": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by sandbox public preview status.",
			},
			"is_recoverable": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by sandbox recoverable status.",
			},
			"created_at_after":  optionalRFC3339DataSourceStringAttribute("Include sandboxes created after this RFC3339 timestamp."),
			"created_at_before": optionalRFC3339DataSourceStringAttribute("Include sandboxes created before this RFC3339 timestamp."),
			"last_event_after":  optionalRFC3339DataSourceStringAttribute("Include sandboxes with a last event after this RFC3339 timestamp."),
			"last_event_before": optionalRFC3339DataSourceStringAttribute("Include sandboxes with a last event before this RFC3339 timestamp."),
			"sort": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Sort field. Supported values: `name`, `cpu`, `memoryGib`, `diskGib`, `lastActivityAt`, `createdAt`.",
			},
			"order": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Sort direction. Supported values: `asc`, `desc`.",
			},
			"id":          computedDataSourceStringAttribute("Data source identifier."),
			"next_cursor": computedDataSourceStringAttribute("Cursor for the next page of results, when available."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned Daytona sandboxes.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxQueryItemAttributes(),
				},
			},
		},
	}
}

func (d *SandboxQueryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxQueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sandboxQueryConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.ListSandboxes(ctx)
	request = configureSandboxQueryRequest(ctx, request, config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	result, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to query Daytona sandboxes", "query sandboxes", httpResp, err)
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox query response", "Daytona returned a successful response without sandbox query data.")
		return
	}

	data := sandboxQueryDataSourceModel{
		RequestOrganizationID: config.RequestOrganizationID,
		Cursor:                config.Cursor,
		Limit:                 config.Limit,
		IDPrefix:              config.IDPrefix,
		NamePrefix:            config.NamePrefix,
		LabelsJSON:            config.LabelsJSON,
		IncludeErroredDeleted: config.IncludeErroredDeleted,
		States:                config.States,
		Snapshots:             config.Snapshots,
		RegionIDs:             config.RegionIDs,
		SandboxClasses:        config.SandboxClasses,
		MinCPU:                config.MinCPU,
		MaxCPU:                config.MaxCPU,
		MinMemoryGiB:          config.MinMemoryGiB,
		MaxMemoryGiB:          config.MaxMemoryGiB,
		MinDiskGiB:            config.MinDiskGiB,
		MaxDiskGiB:            config.MaxDiskGiB,
		IsPublic:              config.IsPublic,
		IsRecoverable:         config.IsRecoverable,
		CreatedAtAfter:        config.CreatedAtAfter,
		CreatedAtBefore:       config.CreatedAtBefore,
		LastEventAfter:        config.LastEventAfter,
		LastEventBefore:       config.LastEventBefore,
		Sort:                  config.Sort,
		Order:                 config.Order,
		ID:                    types.StringValue("sandbox_query"),
		NextCursor:            types.StringNull(),
		Items:                 flattenSandboxQueryItems(ctx, result.Items),
	}
	if value, ok := result.GetNextCursorOk(); ok && value != nil {
		data.NextCursor = types.StringValue(*value)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func configureSandboxQueryRequest(ctx context.Context, request apiclient.SandboxAPIListSandboxesRequest, config sandboxQueryConfigModel, diags *diag.Diagnostics) apiclient.SandboxAPIListSandboxesRequest {
	if organizationID := optionalString(config.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if cursor := optionalString(config.Cursor); cursor != nil {
		request = request.Cursor(*cursor)
	}
	if dataSourceInt64Configured(config.Limit) {
		request = request.Limit(float32(config.Limit.ValueInt64()))
	}
	if idPrefix := optionalString(config.IDPrefix); idPrefix != nil {
		request = request.Id(*idPrefix)
	}
	if namePrefix := optionalString(config.NamePrefix); namePrefix != nil {
		request = request.Name(*namePrefix)
	}
	if labelsJSON := optionalString(config.LabelsJSON); labelsJSON != nil {
		if !validateSandboxQueryLabelsJSON(*labelsJSON, diags) {
			return request
		}
		request = request.Labels(*labelsJSON)
	}
	if terraformBoolConfigured(config.IncludeErroredDeleted) {
		request = request.IncludeErroredDeleted(config.IncludeErroredDeleted.ValueBool())
	}
	if states := sandboxStateFilters(ctx, config.States, diags); len(states) > 0 {
		request = request.States(states)
	}
	if snapshots := optionalStringList(ctx, "snapshots", config.Snapshots, diags); len(snapshots) > 0 {
		request = request.Snapshots(snapshots)
	}
	if regionIDs := optionalStringList(ctx, "region_ids", config.RegionIDs, diags); len(regionIDs) > 0 {
		request = request.RegionIds(regionIDs)
	}
	if classes := sandboxClassFilters(ctx, config.SandboxClasses, diags); len(classes) > 0 {
		request = request.SandboxClasses(classes)
	}
	if terraformFloat64Configured(config.MinCPU) {
		request = request.MinCpu(float32(config.MinCPU.ValueFloat64()))
	}
	if terraformFloat64Configured(config.MaxCPU) {
		request = request.MaxCpu(float32(config.MaxCPU.ValueFloat64()))
	}
	if terraformFloat64Configured(config.MinMemoryGiB) {
		request = request.MinMemoryGiB(float32(config.MinMemoryGiB.ValueFloat64()))
	}
	if terraformFloat64Configured(config.MaxMemoryGiB) {
		request = request.MaxMemoryGiB(float32(config.MaxMemoryGiB.ValueFloat64()))
	}
	if terraformFloat64Configured(config.MinDiskGiB) {
		request = request.MinDiskGiB(float32(config.MinDiskGiB.ValueFloat64()))
	}
	if terraformFloat64Configured(config.MaxDiskGiB) {
		request = request.MaxDiskGiB(float32(config.MaxDiskGiB.ValueFloat64()))
	}
	if terraformBoolConfigured(config.IsPublic) {
		request = request.IsPublic(config.IsPublic.ValueBool())
	}
	if terraformBoolConfigured(config.IsRecoverable) {
		request = request.IsRecoverable(config.IsRecoverable.ValueBool())
	}
	if createdAtAfter, ok := optionalRFC3339Time(config.CreatedAtAfter, "created_at_after", diags); ok && createdAtAfter != nil {
		request = request.CreatedAtAfter(*createdAtAfter)
	}
	if createdAtBefore, ok := optionalRFC3339Time(config.CreatedAtBefore, "created_at_before", diags); ok && createdAtBefore != nil {
		request = request.CreatedAtBefore(*createdAtBefore)
	}
	if lastEventAfter, ok := optionalRFC3339Time(config.LastEventAfter, "last_event_after", diags); ok && lastEventAfter != nil {
		request = request.LastEventAfter(*lastEventAfter)
	}
	if lastEventBefore, ok := optionalRFC3339Time(config.LastEventBefore, "last_event_before", diags); ok && lastEventBefore != nil {
		request = request.LastEventBefore(*lastEventBefore)
	}
	if sort, ok := sandboxSortField(config.Sort, diags); ok && sort != nil {
		request = request.Sort(*sort)
	}
	if order, ok := sandboxSortDirection(config.Order, diags); ok && order != nil {
		request = request.Order(*order)
	}

	return request
}

func sandboxQueryItemAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                    computedDataSourceStringAttribute("Daytona sandbox ID."),
		"name":                  computedDataSourceStringAttribute("Sandbox name."),
		"organization_id":       computedDataSourceStringAttribute("Daytona organization ID that owns the sandbox."),
		"target":                computedDataSourceStringAttribute("Target region where the sandbox is created."),
		"runner_id":             computedDataSourceStringAttribute("Runner ID hosting the sandbox, when assigned."),
		"sandbox_class":         computedDataSourceStringAttribute("Sandbox class."),
		"state":                 computedDataSourceStringAttribute("Current sandbox state."),
		"desired_state":         computedDataSourceStringAttribute("Desired sandbox state."),
		"snapshot":              computedDataSourceStringAttribute("Snapshot ID or name used to create the sandbox."),
		"user":                  computedDataSourceStringAttribute("User associated with the sandbox project."),
		"error_reason":          computedDataSourceStringAttribute("Sandbox error reason, when available."),
		"recoverable":           computedDataSourceBoolAttribute("Whether the sandbox can be recovered from its current error state."),
		"public":                computedDataSourceBoolAttribute("Whether HTTP previews are publicly accessible."),
		"cpu":                   computedDataSourceFloat64Attribute("CPU cores allocated to the sandbox."),
		"gpu":                   computedDataSourceFloat64Attribute("GPU units allocated to the sandbox."),
		"gpu_type":              computedDataSourceStringAttribute("GPU type assigned to the sandbox, when available."),
		"memory":                computedDataSourceFloat64Attribute("Memory allocated to the sandbox in GiB."),
		"disk":                  computedDataSourceFloat64Attribute("Disk allocated to the sandbox in GiB."),
		"labels":                computedDataSourceStringMapAttribute("Labels for the sandbox."),
		"backup_state":          computedDataSourceStringAttribute("Sandbox backup state, when available."),
		"auto_stop_interval":    computedDataSourceFloat64Attribute("Auto-stop interval in minutes."),
		"auto_archive_interval": computedDataSourceFloat64Attribute("Auto-archive interval in minutes."),
		"auto_delete_interval":  computedDataSourceFloat64Attribute("Auto-delete interval in minutes."),
		"created_at":            computedDataSourceStringAttribute("Sandbox creation timestamp."),
		"updated_at":            computedDataSourceStringAttribute("Sandbox update timestamp."),
		"last_activity_at":      computedDataSourceStringAttribute("Sandbox last activity timestamp, when available."),
		"daemon_version":        computedDataSourceStringAttribute("Sandbox daemon version, when available."),
		"toolbox_proxy_url":     computedDataSourceStringAttribute("Toolbox proxy URL for the sandbox."),
	}
}

func flattenSandboxQueryItems(ctx context.Context, sandboxes []apiclient.SandboxListItem) []sandboxQueryItemModel {
	items := make([]sandboxQueryItemModel, 0, len(sandboxes))
	for i := range sandboxes {
		items = append(items, flattenSandboxQueryItem(ctx, &sandboxes[i]))
	}
	return items
}

func flattenSandboxQueryItem(ctx context.Context, sandbox *apiclient.SandboxListItem) sandboxQueryItemModel {
	item := sandboxQueryItemModel{
		ID:                  types.StringValue(sandbox.Id),
		Name:                types.StringValue(sandbox.Name),
		OrganizationID:      types.StringValue(sandbox.OrganizationId),
		Target:              types.StringValue(sandbox.Target),
		RunnerID:            pointerStringValue(sandbox.RunnerId),
		SandboxClass:        types.StringNull(),
		State:               types.StringNull(),
		DesiredState:        types.StringNull(),
		Snapshot:            pointerStringValue(sandbox.Snapshot),
		User:                types.StringValue(sandbox.User),
		ErrorReason:         pointerStringValue(sandbox.ErrorReason),
		Recoverable:         pointerBoolValue(sandbox.Recoverable),
		Public:              types.BoolValue(sandbox.Public),
		CPU:                 types.Float64Value(float64(sandbox.Cpu)),
		GPU:                 types.Float64Value(float64(sandbox.Gpu)),
		GPUType:             types.StringNull(),
		Memory:              types.Float64Value(float64(sandbox.Memory)),
		Disk:                types.Float64Value(float64(sandbox.Disk)),
		Labels:              stringMapValue(ctx, sandbox.Labels),
		BackupState:         pointerStringValue(sandbox.BackupState),
		AutoStopInterval:    pointerFloat32Value(sandbox.AutoStopInterval),
		AutoArchiveInterval: pointerFloat32Value(sandbox.AutoArchiveInterval),
		AutoDeleteInterval:  pointerFloat32Value(sandbox.AutoDeleteInterval),
		CreatedAt:           pointerStringValue(sandbox.CreatedAt),
		UpdatedAt:           pointerStringValue(sandbox.UpdatedAt),
		LastActivityAt:      pointerStringValue(sandbox.LastActivityAt),
		DaemonVersion:       pointerStringValue(sandbox.DaemonVersion),
		ToolboxProxyURL:     types.StringValue(sandbox.ToolboxProxyUrl),
	}
	if sandbox.SandboxClass != nil {
		item.SandboxClass = types.StringValue(string(*sandbox.SandboxClass))
	}
	if sandbox.State != nil {
		item.State = types.StringValue(string(*sandbox.State))
	}
	if sandbox.DesiredState != nil {
		item.DesiredState = types.StringValue(string(*sandbox.DesiredState))
	}
	if sandbox.GpuType != nil {
		item.GPUType = types.StringValue(string(*sandbox.GpuType))
	}
	return item
}

func optionalStringList(ctx context.Context, attribute string, value types.List, diags *diag.Diagnostics) []string {
	values, listDiags := stringList(ctx, value)
	diags.Append(listDiags...)
	if listDiags.HasError() {
		diags.AddAttributeError(path.Root(attribute), "Invalid string list", "The value must be a list of strings.")
		return nil
	}
	return values
}

func sandboxStateFilters(ctx context.Context, value types.List, diags *diag.Diagnostics) []apiclient.SandboxState {
	values := optionalStringList(ctx, "states", value, diags)
	states := make([]apiclient.SandboxState, 0, len(values))
	for _, value := range values {
		state := apiclient.SandboxState(value)
		if !state.IsValid() || state == apiclient.SANDBOXSTATE_UNKNOWN_DEFAULT_OPEN_API {
			diags.AddAttributeError(path.Root("states"), "Invalid sandbox state", fmt.Sprintf("Unsupported sandbox state %q. Supported values are: %s.", value, strings.Join(sandboxStateValues(), ", ")))
			return nil
		}
		states = append(states, state)
	}
	return states
}

func sandboxClassFilters(ctx context.Context, value types.List, diags *diag.Diagnostics) []apiclient.SandboxClass {
	values := optionalStringList(ctx, "sandbox_classes", value, diags)
	classes := make([]apiclient.SandboxClass, 0, len(values))
	for _, value := range values {
		class := apiclient.SandboxClass(value)
		if !class.IsValid() || class == apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
			diags.AddAttributeError(path.Root("sandbox_classes"), "Invalid sandbox class", fmt.Sprintf("Unsupported sandbox class %q. Supported values are: %s.", value, strings.Join(sandboxClassValues(), ", ")))
			return nil
		}
		classes = append(classes, class)
	}
	return classes
}

func sandboxSortField(value types.String, diags *diag.Diagnostics) (*apiclient.SandboxListSortField, bool) {
	if optionalString(value) == nil {
		return nil, true
	}
	sort := apiclient.SandboxListSortField(value.ValueString())
	if !sort.IsValid() || sort == apiclient.SANDBOXLISTSORTFIELD_UNKNOWN_DEFAULT_OPEN_API {
		diags.AddAttributeError(path.Root("sort"), "Invalid sandbox sort field", fmt.Sprintf("Unsupported sort field %q. Supported values are: %s.", value.ValueString(), strings.Join(sandboxSortFieldValues(), ", ")))
		return nil, false
	}
	return &sort, true
}

func sandboxSortDirection(value types.String, diags *diag.Diagnostics) (*apiclient.SandboxListSortDirection, bool) {
	if optionalString(value) == nil {
		return nil, true
	}
	order := apiclient.SandboxListSortDirection(value.ValueString())
	if !order.IsValid() || order == apiclient.SANDBOXLISTSORTDIRECTION_UNKNOWN_DEFAULT_OPEN_API {
		diags.AddAttributeError(path.Root("order"), "Invalid sandbox sort direction", fmt.Sprintf("Unsupported sort direction %q. Supported values are: asc, desc.", value.ValueString()))
		return nil, false
	}
	return &order, true
}

func optionalRFC3339Time(value types.String, attribute string, diags *diag.Diagnostics) (*time.Time, bool) {
	if optionalString(value) == nil {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, value.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root(attribute), "Invalid RFC3339 timestamp", fmt.Sprintf("%s must be formatted as RFC3339, for example 2026-12-31T23:59:59Z.", attribute))
		return nil, false
	}
	return &parsed, true
}

func validateSandboxQueryLabelsJSON(value string, diags *diag.Diagnostics) bool {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		diags.AddAttributeError(path.Root("labels_json"), "Invalid labels_json", fmt.Sprintf("The value must be a JSON object string: %s", err))
		return false
	}
	if payload == nil {
		diags.AddAttributeError(path.Root("labels_json"), "Invalid labels_json", "The value must be a JSON object string, not null.")
		return false
	}
	return true
}

func sandboxStateValues() []string {
	values := make([]string, 0, len(apiclient.AllowedSandboxStateEnumValues)-1)
	for _, value := range apiclient.AllowedSandboxStateEnumValues {
		if value != apiclient.SANDBOXSTATE_UNKNOWN_DEFAULT_OPEN_API {
			values = append(values, string(value))
		}
	}
	return values
}

func sandboxClassValues() []string {
	values := make([]string, 0, len(apiclient.AllowedSandboxClassEnumValues)-1)
	for _, value := range apiclient.AllowedSandboxClassEnumValues {
		if value != apiclient.SANDBOXCLASS_UNKNOWN_DEFAULT_OPEN_API {
			values = append(values, string(value))
		}
	}
	return values
}

func sandboxSortFieldValues() []string {
	values := make([]string, 0, len(apiclient.AllowedSandboxListSortFieldEnumValues)-1)
	for _, value := range apiclient.AllowedSandboxListSortFieldEnumValues {
		if value != apiclient.SANDBOXLISTSORTFIELD_UNKNOWN_DEFAULT_OPEN_API {
			values = append(values, string(value))
		}
	}
	return values
}

func optionalFloat64DataSourceAttribute(description string) schema.Float64Attribute {
	return schema.Float64Attribute{
		Optional:            true,
		MarkdownDescription: description,
	}
}

func optionalRFC3339DataSourceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: description,
	}
}

func pointerBoolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func pointerFloat32Value(value *float32) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(float64(*value))
}
