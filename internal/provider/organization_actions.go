package provider

import (
	"context"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &OrganizationInvitationAction{}
var _ action.ActionWithConfigure = &OrganizationInvitationAction{}
var _ action.Action = &OrganizationLifecycleAction{}
var _ action.ActionWithConfigure = &OrganizationLifecycleAction{}

type organizationInvitationActionOperation string

const (
	organizationInvitationActionAccept  organizationInvitationActionOperation = "accept"
	organizationInvitationActionDecline organizationInvitationActionOperation = "decline"
)

func NewOrganizationInvitationAcceptAction() action.Action {
	return &OrganizationInvitationAction{operation: organizationInvitationActionAccept}
}

func NewOrganizationInvitationDeclineAction() action.Action {
	return &OrganizationInvitationAction{operation: organizationInvitationActionDecline}
}

type OrganizationInvitationAction struct {
	client    *daytonaClient
	operation organizationInvitationActionOperation
}

type organizationInvitationActionModel struct {
	InvitationID types.String `tfsdk:"invitation_id"`
}

func (a *OrganizationInvitationAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + string(a.operation) + "_organization_invitation"
}

func (a *OrganizationInvitationAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: organizationInvitationActionDescription(a.operation),
		Attributes: map[string]actionschema.Attribute{
			"invitation_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona organization invitation ID.",
			},
		},
	}
}

func (a *OrganizationInvitationAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *OrganizationInvitationAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data organizationInvitationActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitationID := strings.TrimSpace(data.InvitationID.ValueString())
	if invitationID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization invitation ID",
			"Configure the invitation_id attribute with the Daytona organization invitation ID.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	switch a.operation {
	case organizationInvitationActionAccept:
		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Accepting Daytona organization invitation."})
		}
		_, httpResp, err := a.client.api.OrganizationsAPI.AcceptOrganizationInvitation(ctx, invitationID).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to accept Daytona organization invitation", "accept organization invitation", httpResp, err)
		}
	case organizationInvitationActionDecline:
		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Declining Daytona organization invitation."})
		}
		httpResp, err := a.client.api.OrganizationsAPI.DeclineOrganizationInvitation(ctx, invitationID).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to decline Daytona organization invitation", "decline organization invitation", httpResp, err)
		}
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona organization invitation action",
			"Unsupported organization invitation action operation. Please report this issue to the provider developers.",
		)
	}
}

func organizationInvitationActionDescription(operation organizationInvitationActionOperation) string {
	switch operation {
	case organizationInvitationActionAccept:
		return "Accepts a Daytona organization invitation."
	case organizationInvitationActionDecline:
		return "Declines a Daytona organization invitation."
	default:
		return "Runs a Daytona organization invitation action."
	}
}

type organizationLifecycleActionOperation string

const (
	organizationLifecycleActionLeave     organizationLifecycleActionOperation = "leave"
	organizationLifecycleActionSuspend   organizationLifecycleActionOperation = "suspend"
	organizationLifecycleActionUnsuspend organizationLifecycleActionOperation = "unsuspend"
)

func NewOrganizationLeaveAction() action.Action {
	return &OrganizationLifecycleAction{operation: organizationLifecycleActionLeave}
}

func NewOrganizationSuspendAction() action.Action {
	return &OrganizationLifecycleAction{operation: organizationLifecycleActionSuspend}
}

func NewOrganizationUnsuspendAction() action.Action {
	return &OrganizationLifecycleAction{operation: organizationLifecycleActionUnsuspend}
}

type OrganizationLifecycleAction struct {
	client    *daytonaClient
	operation organizationLifecycleActionOperation
}

type organizationLifecycleActionModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
}

type organizationSuspendActionModel struct {
	OrganizationID                    types.String  `tfsdk:"organization_id"`
	Reason                            types.String  `tfsdk:"reason"`
	Until                             types.String  `tfsdk:"until"`
	SuspensionCleanupGracePeriodHours types.Float64 `tfsdk:"suspension_cleanup_grace_period_hours"`
}

func (a *OrganizationLifecycleAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + string(a.operation) + "_organization"
}

func (a *OrganizationLifecycleAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attributes := map[string]actionschema.Attribute{
		"organization_id": actionschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Daytona organization ID.",
		},
	}

	if a.operation == organizationLifecycleActionSuspend {
		attributes["reason"] = actionschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Reason for suspending the Daytona organization.",
		}
		attributes["until"] = actionschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "RFC3339 timestamp for when the Daytona organization suspension ends.",
		}
		attributes["suspension_cleanup_grace_period_hours"] = actionschema.Float64Attribute{
			Optional:            true,
			MarkdownDescription: "Optional suspension cleanup grace period in hours.",
		}
	}

	resp.Schema = actionschema.Schema{
		MarkdownDescription: organizationLifecycleActionDescription(a.operation),
		Attributes:          attributes,
	}
}

func (a *OrganizationLifecycleAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *OrganizationLifecycleAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	switch a.operation {
	case organizationLifecycleActionLeave:
		organizationID, ok := organizationLifecycleActionOrganizationID(ctx, req, resp)
		if !ok || !ensureActionClient(a.client, &resp.Diagnostics) {
			return
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Leaving Daytona organization."})
		}
		httpResp, err := a.client.api.OrganizationsAPI.LeaveOrganization(ctx, organizationID).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to leave Daytona organization", "leave organization", httpResp, err)
		}
	case organizationLifecycleActionSuspend:
		var data organizationSuspendActionModel

		resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		organizationID, ok := organizationIDValue(data.OrganizationID, resp)
		if !ok || !ensureActionClient(a.client, &resp.Diagnostics) {
			return
		}

		payload, ok := organizationSuspensionPayload(data, resp)
		if !ok {
			return
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Suspending Daytona organization."})
		}
		httpResp, err := a.client.api.OrganizationsAPI.SuspendOrganization(ctx, organizationID).OrganizationSuspension(payload).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to suspend Daytona organization", "suspend organization", httpResp, err)
		}
	case organizationLifecycleActionUnsuspend:
		organizationID, ok := organizationLifecycleActionOrganizationID(ctx, req, resp)
		if !ok || !ensureActionClient(a.client, &resp.Diagnostics) {
			return
		}

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Unsuspending Daytona organization."})
		}
		httpResp, err := a.client.api.OrganizationsAPI.UnsuspendOrganization(ctx, organizationID).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to unsuspend Daytona organization", "unsuspend organization", httpResp, err)
		}
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona organization action",
			"Unsupported organization action operation. Please report this issue to the provider developers.",
		)
	}
}

func organizationLifecycleActionOrganizationID(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) (string, bool) {
	var data organizationLifecycleActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return "", false
	}

	return organizationIDValue(data.OrganizationID, resp)
}

func organizationIDValue(value types.String, resp *action.InvokeResponse) (string, bool) {
	organizationID := strings.TrimSpace(value.ValueString())
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization ID",
			"Configure the organization_id attribute with the Daytona organization ID.",
		)
		return "", false
	}

	return organizationID, true
}

func organizationSuspensionPayload(data organizationSuspendActionModel, resp *action.InvokeResponse) (apiclient.OrganizationSuspension, bool) {
	reason := strings.TrimSpace(data.Reason.ValueString())
	if reason == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization suspension reason",
			"Configure the reason attribute with the reason for suspending the Daytona organization.",
		)
		return apiclient.OrganizationSuspension{}, false
	}

	until, err := time.Parse(time.RFC3339, strings.TrimSpace(data.Until.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Daytona organization suspension timestamp",
			"until must be formatted as RFC3339, for example 2026-12-31T23:59:59Z.",
		)
		return apiclient.OrganizationSuspension{}, false
	}

	payload := apiclient.OrganizationSuspension{
		Reason: reason,
		Until:  until,
	}

	if !data.SuspensionCleanupGracePeriodHours.IsNull() && !data.SuspensionCleanupGracePeriodHours.IsUnknown() {
		gracePeriodHours := float32(data.SuspensionCleanupGracePeriodHours.ValueFloat64())
		payload.SuspensionCleanupGracePeriodHours = &gracePeriodHours
	}

	return payload, true
}

func organizationLifecycleActionDescription(operation organizationLifecycleActionOperation) string {
	switch operation {
	case organizationLifecycleActionLeave:
		return "Leaves a Daytona organization."
	case organizationLifecycleActionSuspend:
		return "Suspends a Daytona organization."
	case organizationLifecycleActionUnsuspend:
		return "Unsuspends a Daytona organization."
	default:
		return "Runs a Daytona organization action."
	}
}
