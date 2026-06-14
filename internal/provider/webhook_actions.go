package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &webhookAction{}
var _ action.ActionWithConfigure = &webhookAction{}

type webhookActionOperation string

const (
	webhookActionInitialize       webhookActionOperation = "initialize"
	webhookActionRefreshEndpoints webhookActionOperation = "refresh_endpoints"
)

func NewWebhookInitializeAction() action.Action {
	return &webhookAction{
		operation: webhookActionInitialize,
	}
}

func NewWebhookRefreshEndpointsAction() action.Action {
	return &webhookAction{
		operation: webhookActionRefreshEndpoints,
	}
}

type webhookAction struct {
	client    *daytonaClient
	operation webhookActionOperation
}

type webhookActionModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (a *webhookAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	switch a.operation {
	case webhookActionInitialize:
		resp.TypeName = req.ProviderTypeName + "_initialize_webhooks"
	case webhookActionRefreshEndpoints:
		resp.TypeName = req.ProviderTypeName + "_refresh_webhook_endpoints"
	default:
		resp.TypeName = req.ProviderTypeName + "_webhook_action"
	}
}

func (a *webhookAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: webhookActionDescription(a.operation),
		Attributes: map[string]actionschema.Attribute{
			"organization_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona organization ID to operate on. Defaults to the provider-level organization ID when configured.",
			},
		},
	}
}

func (a *webhookAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *webhookAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data webhookActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Daytona client",
			"The provider did not configure a Daytona API client for this action.",
		)
		return
	}

	organizationID := strings.TrimSpace(data.OrganizationID.ValueString())
	if organizationID == "" {
		organizationID = a.client.organizationID
	}
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona organization ID",
			"Configure organization_id on this action or configure organization_id on the Daytona provider.",
		)
		return
	}

	switch a.operation {
	case webhookActionInitialize:
		request := a.client.api.WebhooksAPI.
			WebhookControllerInitializeWebhooks(ctx, organizationID).
			XDaytonaOrganizationID(organizationID)

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Initializing Daytona webhooks."})
		}

		_, httpResp, err := request.Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to initialize Daytona webhooks", "initialize webhooks", httpResp, err)
			return
		}
	case webhookActionRefreshEndpoints:
		request := a.client.api.WebhooksAPI.
			WebhookControllerRefreshEndpoints(ctx, organizationID).
			XDaytonaOrganizationID(organizationID)

		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Refreshing Daytona webhook endpoints."})
		}

		httpResp, err := request.Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to refresh Daytona webhook endpoints", "refresh webhook endpoints", httpResp, err)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona webhook action",
			fmt.Sprintf("Unsupported webhook action operation %q. Please report this issue to the provider developers.", a.operation),
		)
	}
}

func webhookActionDescription(operation webhookActionOperation) string {
	switch operation {
	case webhookActionInitialize:
		return "Initializes webhooks for a Daytona organization."
	case webhookActionRefreshEndpoints:
		return "Refreshes cached webhook endpoint presence for a Daytona organization."
	default:
		return "Runs a Daytona webhook action."
	}
}
