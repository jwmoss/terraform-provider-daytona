// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &SandboxRecoverAction{}
var _ action.ActionWithConfigure = &SandboxRecoverAction{}
var _ action.Action = &SandboxCreateBackupAction{}
var _ action.ActionWithConfigure = &SandboxCreateBackupAction{}
var _ action.Action = &SandboxCreateSnapshotAction{}
var _ action.ActionWithConfigure = &SandboxCreateSnapshotAction{}
var _ action.Action = &SandboxForkAction{}
var _ action.ActionWithConfigure = &SandboxForkAction{}
var _ action.Action = &SandboxUpdateLastActivityAction{}
var _ action.ActionWithConfigure = &SandboxUpdateLastActivityAction{}

func NewSandboxRecoverAction() action.Action {
	return &SandboxRecoverAction{}
}

func NewSandboxCreateBackupAction() action.Action {
	return &SandboxCreateBackupAction{}
}

func NewSandboxCreateSnapshotAction() action.Action {
	return &SandboxCreateSnapshotAction{}
}

func NewSandboxForkAction() action.Action {
	return &SandboxForkAction{}
}

func NewSandboxUpdateLastActivityAction() action.Action {
	return &SandboxUpdateLastActivityAction{}
}

type SandboxRecoverAction struct {
	client *daytonaClient
}

type sandboxRecoverActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	SkipStart       types.Bool   `tfsdk:"skip_start"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxRecoverAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recover_sandbox"
}

func (a *SandboxRecoverAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Recovers a Daytona sandbox from an error state.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": sandboxIDOrNameActionAttribute(),
			"skip_start": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to leave the sandbox stopped after recovery instead of starting it.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxRecoverAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxRecoverAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxRecoverActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to recover.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	request := a.client.api.SandboxAPI.RecoverSandbox(ctx, sandboxIDOrName)
	if skipStart := optionalBool(data.SkipStart); skipStart != nil {
		request = request.SkipStart(*skipStart)
	}
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Recovering Daytona sandbox."})
	}

	_, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to recover Daytona sandbox", "recover sandbox", httpResp, err)
	}
}

type SandboxCreateBackupAction struct {
	client *daytonaClient
}

type sandboxCreateBackupActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxCreateBackupAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_create_sandbox_backup"
}

func (a *SandboxCreateBackupAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Starts a Daytona sandbox backup.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": sandboxIDOrNameActionAttribute(),
			"organization_id":    organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxCreateBackupAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxCreateBackupAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxCreateBackupActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to back up.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	request := a.client.api.SandboxAPI.CreateBackup(ctx, sandboxIDOrName)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Starting Daytona sandbox backup."})
	}

	_, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona sandbox backup", "create sandbox backup", httpResp, err)
	}
}

type SandboxCreateSnapshotAction struct {
	client *daytonaClient
}

type sandboxCreateSnapshotActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	Name            types.String `tfsdk:"name"`
	IncludeMemory   types.Bool   `tfsdk:"include_memory"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxCreateSnapshotAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_create_sandbox_snapshot"
}

func (a *SandboxCreateSnapshotAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Starts Daytona snapshot creation from a sandbox.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": sandboxIDOrNameActionAttribute(),
			"name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name for the new Daytona snapshot.",
			},
			"include_memory": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to include VM memory in the snapshot when Daytona supports memory snapshots for the sandbox.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxCreateSnapshotAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxCreateSnapshotAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxCreateSnapshotActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to snapshot.",
		)
		return
	}
	name := strings.TrimSpace(data.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona snapshot name",
			"Configure the name attribute with the Daytona snapshot name to create.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	payload := apiclient.CreateSandboxSnapshot{Name: name}
	if includeMemory := optionalBool(data.IncludeMemory); includeMemory != nil {
		payload.IncludeMemory = includeMemory
	}

	request := a.client.api.SandboxAPI.CreateSandboxSnapshot(ctx, sandboxIDOrName).CreateSandboxSnapshot(payload)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Starting Daytona sandbox snapshot creation."})
	}

	_, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona sandbox snapshot", "create sandbox snapshot", httpResp, err)
	}
}

type SandboxForkAction struct {
	client *daytonaClient
}

type sandboxForkActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	Name            types.String `tfsdk:"name"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxForkAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fork_sandbox"
}

func (a *SandboxForkAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Starts a Daytona sandbox fork.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": sandboxIDOrNameActionAttribute(),
			"name": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional name for the forked Daytona sandbox. When omitted, Daytona generates a unique name.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxForkAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxForkAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxForkActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to fork.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	payload := apiclient.ForkSandbox{}
	if name := optionalString(data.Name); name != nil {
		payload.Name = name
	}

	request := a.client.api.SandboxAPI.ForkSandbox(ctx, sandboxIDOrName).ForkSandbox(payload)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Starting Daytona sandbox fork."})
	}

	_, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to fork Daytona sandbox", "fork sandbox", httpResp, err)
	}
}

type SandboxUpdateLastActivityAction struct {
	client *daytonaClient
}

type sandboxUpdateLastActivityActionModel struct {
	SandboxID      types.String `tfsdk:"sandbox_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (a *SandboxUpdateLastActivityAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_update_sandbox_last_activity"
}

func (a *SandboxUpdateLastActivityAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Updates a Daytona sandbox last-activity timestamp.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona sandbox ID.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxUpdateLastActivityAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxUpdateLastActivityAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxUpdateLastActivityActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxID := strings.TrimSpace(data.SandboxID.ValueString())
	if sandboxID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID",
			"Configure the sandbox_id attribute with the Daytona sandbox ID to update.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	request := a.client.api.SandboxAPI.UpdateLastActivity(ctx, sandboxID)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Updating Daytona sandbox last activity."})
	}

	httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona sandbox last activity", "update sandbox last activity", httpResp, err)
	}
}

func sandboxIDOrNameActionAttribute() actionschema.StringAttribute {
	return actionschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Daytona sandbox ID or name.",
	}
}

func organizationIDActionAttribute() actionschema.StringAttribute {
	return actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Daytona organization ID to send as `X-Daytona-Organization-ID` for this action. Defaults to the provider-level organization ID when configured.",
	}
}

func configureActionDaytonaClient(providerData any, diags *diag.Diagnostics) *daytonaClient {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*daytonaClient)
	if !ok {
		diags.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

func ensureActionClient(client *daytonaClient, diags *diag.Diagnostics) bool {
	if client != nil {
		return true
	}

	diags.AddError(
		"Unconfigured Daytona client",
		"The provider did not configure a Daytona API client for this action.",
	)
	return false
}
