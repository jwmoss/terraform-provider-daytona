// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &snapshotAction{}
var _ action.ActionWithConfigure = &snapshotAction{}

type snapshotActionOperation string

const (
	snapshotActionActivate   snapshotActionOperation = "activate"
	snapshotActionDeactivate snapshotActionOperation = "deactivate"
)

func NewSnapshotActivateAction() action.Action {
	return &snapshotAction{
		operation: snapshotActionActivate,
	}
}

func NewSnapshotDeactivateAction() action.Action {
	return &snapshotAction{
		operation: snapshotActionDeactivate,
	}
}

type snapshotAction struct {
	client    *daytonaClient
	operation snapshotActionOperation
}

type snapshotActionModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (a *snapshotAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_snapshot", req.ProviderTypeName, a.operation)
}

func (a *snapshotAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: fmt.Sprintf("%s a Daytona snapshot.", snapshotActionDescription(a.operation)),
		Attributes: map[string]actionschema.Attribute{
			"id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona snapshot ID.",
			},
			"organization_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona organization ID to send as `X-Daytona-Organization-ID` for this action. Defaults to the provider-level organization ID when configured.",
			},
		},
	}
}

func (a *snapshotAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*daytonaClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.client = client
}

func (a *snapshotAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data snapshotActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotID := strings.TrimSpace(data.ID.ValueString())
	if snapshotID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona snapshot ID",
			"Configure the id attribute with the Daytona snapshot ID to operate on.",
		)
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Daytona client",
			"The provider did not configure a Daytona API client for this action.",
		)
		return
	}

	switch a.operation {
	case snapshotActionActivate:
		request := a.client.api.SnapshotsAPI.ActivateSnapshot(ctx, snapshotID)
		if organizationID := optionalString(data.OrganizationID); organizationID != nil {
			request = request.XDaytonaOrganizationID(*organizationID)
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Activating Daytona snapshot."})
		}

		_, httpResp, err := request.Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to activate Daytona snapshot", "activate snapshot", httpResp, err)
			return
		}
	case snapshotActionDeactivate:
		request := a.client.api.SnapshotsAPI.DeactivateSnapshot(ctx, snapshotID)
		if organizationID := optionalString(data.OrganizationID); organizationID != nil {
			request = request.XDaytonaOrganizationID(*organizationID)
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Deactivating Daytona snapshot."})
		}

		httpResp, err := request.Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to deactivate Daytona snapshot", "deactivate snapshot", httpResp, err)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona snapshot action",
			fmt.Sprintf("Unsupported snapshot action operation %q. Please report this issue to the provider developers.", a.operation),
		)
	}
}

func snapshotActionDescription(operation snapshotActionOperation) string {
	switch operation {
	case snapshotActionActivate:
		return "Activates"
	case snapshotActionDeactivate:
		return "Deactivates"
	default:
		return "Runs"
	}
}
