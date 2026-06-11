// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SandboxResource{}
var _ resource.ResourceWithImportState = &SandboxResource{}

func NewSandboxResource() resource.Resource {
	return &SandboxResource{}
}

type SandboxResource struct {
	client *daytonaClient
}

type sandboxResourceModel struct {
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
	LinkedSandbox       types.String `tfsdk:"linked_sandbox"`
	DesiredState        types.String `tfsdk:"desired_state"`
	State               types.String `tfsdk:"state"`
	RunnerID            types.String `tfsdk:"runner_id"`
	ToolboxProxyURL     types.String `tfsdk:"toolbox_proxy_url"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	ErrorReason         types.String `tfsdk:"error_reason"`
}

func (r *SandboxResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox"
}

func (r *SandboxResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona sandbox ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": replaceStringAttribute("Sandbox name. If omitted, Daytona uses the sandbox ID as the name."),
			"organization_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona organization ID that owns the sandbox.",
			},
			"snapshot": replaceStringAttribute("Snapshot ID or name used to create the sandbox."),
			"user":     replaceStringAttribute("User associated with the sandbox project."),
			"env": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Environment variables for the sandbox.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Labels for the sandbox.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"public": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether HTTP previews are publicly accessible.",
			},
			"network_block_all": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to block all sandbox network access.",
			},
			"network_allow_list": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comma-separated list of allowed CIDR network addresses.",
			},
			"target":                replaceStringAttribute("Target region where the sandbox is created."),
			"cpu":                   replaceInt64Attribute("CPU cores allocated to the sandbox."),
			"gpu":                   replaceInt64Attribute("GPU units allocated to the sandbox."),
			"memory":                replaceInt64Attribute("Memory allocated to the sandbox in GB."),
			"disk":                  replaceInt64Attribute("Disk allocated to the sandbox in GB."),
			"auto_stop_interval":    replaceInt64Attribute("Auto-stop interval in minutes. Use 0 to disable."),
			"auto_archive_interval": replaceInt64Attribute("Auto-archive interval in minutes."),
			"auto_delete_interval":  replaceInt64Attribute("Auto-delete interval in minutes. Negative values disable auto-delete."),
			"linked_sandbox":        replaceStringAttribute("Existing sandbox ID or name to link the new sandbox to."),
			"desired_state": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional desired lifecycle state: `started`, `stopped`, or `archived`.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current sandbox state.",
			},
			"runner_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runner ID hosting the sandbox, when assigned.",
			},
			"toolbox_proxy_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Toolbox proxy URL for the sandbox.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sandbox creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sandbox update timestamp.",
			},
			"error_reason": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sandbox error reason, when available.",
			},
		},
	}
}

func replaceStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func replaceInt64Attribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	}
}

func (r *SandboxResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SandboxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data sandboxResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createSandbox, err := expandCreateSandbox(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Daytona sandbox configuration", err.Error())
		return
	}

	sandbox, httpResp, err := r.client.api.SandboxAPI.CreateSandbox(ctx).
		CreateSandbox(createSandbox).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona sandbox", "create sandbox", httpResp, err)
		return
	}

	data = flattenSandbox(ctx, sandbox, data)
	if !data.DesiredState.IsNull() && data.DesiredState.ValueString() != "" {
		sandbox, httpResp, err = r.applyDesiredState(ctx, data.ID.ValueString(), data.DesiredState.ValueString())
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to set Daytona sandbox state", "set sandbox state", httpResp, err)
			return
		}
		data = flattenSandbox(ctx, sandbox, data)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SandboxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data sandboxResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandbox, httpResp, err := r.client.api.SandboxAPI.GetSandbox(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox", "read sandbox", httpResp, err)
		return
	}

	data = flattenSandbox(ctx, sandbox, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SandboxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sandboxResourceModel
	var state sandboxResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sandbox *apiclient.Sandbox
	var httpRespErr error
	var httpResp any

	if !plan.Public.Equal(state.Public) {
		sandbox, response, err := r.client.api.SandboxAPI.UpdatePublicStatus(ctx, state.ID.ValueString(), plan.Public.ValueBool()).Execute()
		httpResp = response
		httpRespErr = err
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to update Daytona sandbox public status", "update sandbox public status", response, err)
			return
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if !plan.NetworkBlockAll.Equal(state.NetworkBlockAll) || !plan.NetworkAllowList.Equal(state.NetworkAllowList) {
		updateNetworkSettings := apiclient.NewUpdateSandboxNetworkSettings()
		if !plan.NetworkBlockAll.IsNull() && !plan.NetworkBlockAll.IsUnknown() {
			updateNetworkSettings.SetNetworkBlockAll(plan.NetworkBlockAll.ValueBool())
		}
		if value := optionalString(plan.NetworkAllowList); value != nil {
			updateNetworkSettings.SetNetworkAllowList(*value)
		}

		sandbox, response, err := r.client.api.SandboxAPI.UpdateNetworkSettings(ctx, state.ID.ValueString()).
			UpdateSandboxNetworkSettings(*updateNetworkSettings).
			Execute()
		httpResp = response
		httpRespErr = err
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to update Daytona sandbox network settings", "update sandbox network settings", response, err)
			return
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if !plan.DesiredState.Equal(state.DesiredState) && !plan.DesiredState.IsNull() && plan.DesiredState.ValueString() != "" {
		sandbox, response, err := r.applyDesiredState(ctx, state.ID.ValueString(), plan.DesiredState.ValueString())
		httpResp = response
		httpRespErr = err
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to set Daytona sandbox state", "set sandbox state", response, err)
			return
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if sandbox == nil && httpResp == nil && httpRespErr == nil {
		current, response, err := r.client.api.SandboxAPI.GetSandbox(ctx, state.ID.ValueString()).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox", "read sandbox", response, err)
			return
		}
		plan = flattenSandbox(ctx, current, plan)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SandboxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data sandboxResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.api.SandboxAPI.DeleteSandbox(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona sandbox", "delete sandbox", httpResp, err)
		return
	}
}

func (r *SandboxResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func expandCreateSandbox(ctx context.Context, data sandboxResourceModel) (apiclient.CreateSandbox, error) {
	createSandbox := apiclient.NewCreateSandbox()

	if value := optionalString(data.Name); value != nil {
		createSandbox.SetName(*value)
	}
	if value := optionalString(data.Snapshot); value != nil {
		createSandbox.SetSnapshot(*value)
	}
	if value := optionalString(data.User); value != nil {
		createSandbox.SetUser(*value)
	}
	if value := optionalBool(data.Public); value != nil {
		createSandbox.SetPublic(*value)
	}
	if value := optionalBool(data.NetworkBlockAll); value != nil {
		createSandbox.SetNetworkBlockAll(*value)
	}
	if value := optionalString(data.NetworkAllowList); value != nil {
		createSandbox.SetNetworkAllowList(*value)
	}
	if value := optionalString(data.Target); value != nil {
		createSandbox.SetTarget(*value)
	}
	if value := optionalInt32(data.CPU); value != nil {
		createSandbox.SetCpu(*value)
	}
	if value := optionalInt32(data.GPU); value != nil {
		createSandbox.SetGpu(*value)
	}
	if value := optionalInt32(data.Memory); value != nil {
		createSandbox.SetMemory(*value)
	}
	if value := optionalInt32(data.Disk); value != nil {
		createSandbox.SetDisk(*value)
	}
	if value := optionalInt32(data.AutoStopInterval); value != nil {
		createSandbox.SetAutoStopInterval(*value)
	}
	if value := optionalInt32(data.AutoArchiveInterval); value != nil {
		createSandbox.SetAutoArchiveInterval(*value)
	}
	if value := optionalInt32(data.AutoDeleteInterval); value != nil {
		createSandbox.SetAutoDeleteInterval(*value)
	}
	if value := optionalString(data.LinkedSandbox); value != nil {
		createSandbox.SetLinkedSandbox(*value)
	}

	env, diags := stringMap(ctx, data.Env)
	if diags.HasError() {
		return *createSandbox, fmt.Errorf("invalid env map")
	}
	if len(env) > 0 {
		createSandbox.SetEnv(env)
	}

	labels, diags := stringMap(ctx, data.Labels)
	if diags.HasError() {
		return *createSandbox, fmt.Errorf("invalid labels map")
	}
	if len(labels) > 0 {
		createSandbox.SetLabels(labels)
	}

	if desiredState := data.DesiredState.ValueString(); desiredState != "" && desiredState != "started" && desiredState != "stopped" && desiredState != "archived" {
		return *createSandbox, fmt.Errorf("desired_state must be one of started, stopped, or archived")
	}

	return *createSandbox, nil
}

func (r *SandboxResource) applyDesiredState(ctx context.Context, id string, desiredState string) (*apiclient.Sandbox, *http.Response, error) {
	switch desiredState {
	case "started":
		return r.client.api.SandboxAPI.StartSandbox(ctx, id).Execute()
	case "stopped":
		return r.client.api.SandboxAPI.StopSandbox(ctx, id).Execute()
	case "archived":
		return r.client.api.SandboxAPI.ArchiveSandbox(ctx, id).Execute()
	case "":
		return r.client.api.SandboxAPI.GetSandbox(ctx, id).Execute()
	default:
		return nil, nil, fmt.Errorf("unsupported desired_state %q", desiredState)
	}
}

func flattenSandbox(ctx context.Context, sandbox *apiclient.Sandbox, prior sandboxResourceModel) sandboxResourceModel {
	if sandbox == nil {
		return prior
	}

	prior.ID = types.StringValue(sandbox.Id)
	prior.Name = types.StringValue(sandbox.Name)
	prior.OrganizationID = types.StringValue(sandbox.OrganizationId)
	prior.User = types.StringValue(sandbox.User)
	prior.Public = types.BoolValue(sandbox.Public)
	prior.NetworkBlockAll = types.BoolValue(sandbox.NetworkBlockAll)
	prior.Target = types.StringValue(sandbox.Target)
	prior.CPU = types.Int64Value(int64(sandbox.Cpu))
	prior.GPU = types.Int64Value(int64(sandbox.Gpu))
	prior.Memory = types.Int64Value(int64(sandbox.Memory))
	prior.Disk = types.Int64Value(int64(sandbox.Disk))
	prior.ToolboxProxyURL = types.StringValue(sandbox.ToolboxProxyUrl)

	if sandbox.Snapshot != nil {
		prior.Snapshot = types.StringValue(*sandbox.Snapshot)
	} else if prior.Snapshot.IsUnknown() {
		prior.Snapshot = types.StringNull()
	}
	if sandbox.NetworkAllowList != nil {
		prior.NetworkAllowList = types.StringValue(*sandbox.NetworkAllowList)
	} else {
		prior.NetworkAllowList = types.StringNull()
	}
	if sandbox.State != nil {
		prior.State = types.StringValue(string(*sandbox.State))
	} else {
		prior.State = types.StringNull()
	}
	if sandbox.RunnerId != nil {
		prior.RunnerID = types.StringValue(*sandbox.RunnerId)
	} else {
		prior.RunnerID = types.StringNull()
	}
	if sandbox.CreatedAt != nil {
		prior.CreatedAt = types.StringValue(*sandbox.CreatedAt)
	} else {
		prior.CreatedAt = types.StringNull()
	}
	if sandbox.UpdatedAt != nil {
		prior.UpdatedAt = types.StringValue(*sandbox.UpdatedAt)
	} else {
		prior.UpdatedAt = types.StringNull()
	}
	if sandbox.ErrorReason != nil {
		prior.ErrorReason = types.StringValue(*sandbox.ErrorReason)
	} else {
		prior.ErrorReason = types.StringNull()
	}
	if sandbox.AutoStopInterval != nil {
		prior.AutoStopInterval = types.Int64Value(int64(*sandbox.AutoStopInterval))
	}
	if sandbox.AutoArchiveInterval != nil {
		prior.AutoArchiveInterval = types.Int64Value(int64(*sandbox.AutoArchiveInterval))
	}
	if sandbox.AutoDeleteInterval != nil {
		prior.AutoDeleteInterval = types.Int64Value(int64(*sandbox.AutoDeleteInterval))
	}

	env, _ := types.MapValueFrom(ctx, types.StringType, sandbox.Env)
	prior.Env = env
	labels, _ := types.MapValueFrom(ctx, types.StringType, sandbox.Labels)
	prior.Labels = labels

	return prior
}
