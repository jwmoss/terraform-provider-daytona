// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &adminWebhookAction{}
var _ action.ActionWithConfigure = &adminWebhookAction{}

type adminWebhookActionOperation string

const (
	adminWebhookActionInitialize adminWebhookActionOperation = "initialize"
	adminWebhookActionSend       adminWebhookActionOperation = "send"
)

func NewAdminInitializeWebhooksAction() action.Action {
	return &adminWebhookAction{
		operation: adminWebhookActionInitialize,
	}
}

func NewAdminSendWebhookAction() action.Action {
	return &adminWebhookAction{
		operation: adminWebhookActionSend,
	}
}

type adminWebhookAction struct {
	client    *daytonaClient
	operation adminWebhookActionOperation
}

type adminWebhookInitializeActionModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
}

type adminWebhookSendActionModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	EventType      types.String `tfsdk:"event_type"`
	PayloadJSON    types.String `tfsdk:"payload_json"`
	EventID        types.String `tfsdk:"event_id"`
}

func (a *adminWebhookAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	switch a.operation {
	case adminWebhookActionInitialize:
		resp.TypeName = req.ProviderTypeName + "_admin_initialize_webhooks"
	case adminWebhookActionSend:
		resp.TypeName = req.ProviderTypeName + "_admin_send_webhook"
	default:
		resp.TypeName = req.ProviderTypeName + "_admin_webhook_action"
	}
}

func (a *adminWebhookAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attributes := map[string]actionschema.Attribute{
		"organization_id": actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Daytona organization ID to operate on. Defaults to the provider-level organization ID when configured.",
		},
	}

	if a.operation == adminWebhookActionSend {
		attributes["event_type"] = actionschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Daytona webhook event type to send.",
		}
		attributes["payload_json"] = actionschema.StringAttribute{
			Required:            true,
			WriteOnly:           true,
			MarkdownDescription: "Webhook payload as a JSON object.",
		}
		attributes["event_id"] = actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Optional webhook event ID for idempotency.",
		}
	}

	resp.Schema = actionschema.Schema{
		MarkdownDescription: adminWebhookActionDescription(a.operation),
		Attributes:          attributes,
	}
}

func (a *adminWebhookAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *adminWebhookAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if a.client == nil {
		resp.Diagnostics.AddError("Unconfigured Daytona client", "The provider did not configure a Daytona API client for this action.")
		return
	}

	switch a.operation {
	case adminWebhookActionInitialize:
		a.invokeInitialize(ctx, req, resp)
	case adminWebhookActionSend:
		a.invokeSend(ctx, req, resp)
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona admin webhook action",
			fmt.Sprintf("Unsupported admin webhook action operation %q. Please report this issue to the provider developers.", a.operation),
		)
	}
}

func (a *adminWebhookAction) invokeInitialize(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminWebhookInitializeActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := a.organizationID(data.OrganizationID)
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization ID",
			"Configure organization_id on this action or configure organization_id on the Daytona provider.",
		)
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Initializing Daytona admin webhooks."})
	}

	httpResp, err := a.client.api.AdminAPI.AdminInitializeWebhooks(ctx, organizationID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to initialize Daytona admin webhooks", "initialize admin webhooks", httpResp, err)
	}
}

func (a *adminWebhookAction) invokeSend(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminWebhookSendActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := a.organizationID(data.OrganizationID)
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization ID",
			"Configure organization_id on this action or configure organization_id on the Daytona provider.",
		)
		return
	}

	eventType := strings.TrimSpace(data.EventType.ValueString())
	if !isKnownWebhookEvent(eventType) {
		resp.Diagnostics.AddError(
			"Invalid Daytona webhook event type",
			fmt.Sprintf("Configure event_type with one of: %s.", strings.Join(knownWebhookEvents(), ", ")),
		)
		return
	}

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(data.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddError("Invalid Daytona webhook payload", fmt.Sprintf("payload_json must be a JSON object: %s", err))
		return
	}
	if payload == nil {
		resp.Diagnostics.AddError("Invalid Daytona webhook payload", "payload_json must be a JSON object, not null.")
		return
	}

	body := *apiclient.NewSendWebhookDto(apiclient.WebhookEvent(eventType), payload)
	if eventID := strings.TrimSpace(data.EventID.ValueString()); eventID != "" {
		body.SetEventId(eventID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Sending Daytona admin webhook."})
	}

	httpResp, err := a.client.api.AdminAPI.AdminSendWebhook(ctx, organizationID).
		SendWebhookDto(body).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to send Daytona admin webhook", "send admin webhook", httpResp, err)
	}
}

func (a *adminWebhookAction) organizationID(value types.String) string {
	organizationID := strings.TrimSpace(value.ValueString())
	if organizationID == "" && a.client != nil {
		organizationID = a.client.organizationID
	}
	return organizationID
}

func adminWebhookActionDescription(operation adminWebhookActionOperation) string {
	switch operation {
	case adminWebhookActionInitialize:
		return "Initializes Daytona webhooks for an organization through the Daytona admin API. This action requires Daytona admin privileges."
	case adminWebhookActionSend:
		return "Sends a Daytona webhook event to an organization through the Daytona admin API. This action requires Daytona admin privileges."
	default:
		return "Runs a Daytona admin webhook action."
	}
}

func knownWebhookEvents() []string {
	return []string{
		string(apiclient.WEBHOOKEVENT_SANDBOX_CREATED),
		string(apiclient.WEBHOOKEVENT_SANDBOX_STATE_UPDATED),
		string(apiclient.WEBHOOKEVENT_SNAPSHOT_CREATED),
		string(apiclient.WEBHOOKEVENT_SNAPSHOT_STATE_UPDATED),
		string(apiclient.WEBHOOKEVENT_SNAPSHOT_REMOVED),
		string(apiclient.WEBHOOKEVENT_VOLUME_CREATED),
		string(apiclient.WEBHOOKEVENT_VOLUME_STATE_UPDATED),
	}
}

func isKnownWebhookEvent(value string) bool {
	for _, allowed := range knownWebhookEvents() {
		if value == allowed {
			return true
		}
	}
	return false
}
