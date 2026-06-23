package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &RunnerHealthcheckAction{}
var _ action.ActionWithConfigure = &RunnerHealthcheckAction{}

func NewRunnerHealthcheckAction() action.Action {
	return &RunnerHealthcheckAction{}
}

type RunnerHealthcheckAction struct {
	client *daytonaClient
}

type runnerHealthcheckActionModel struct {
	AppVersion    types.String `tfsdk:"app_version"`
	Domain        types.String `tfsdk:"domain"`
	ProxyURL      types.String `tfsdk:"proxy_url"`
	APIURL        types.String `tfsdk:"api_url"`
	MetricsJSON   types.String `tfsdk:"metrics_json"`
	ServiceHealth types.String `tfsdk:"service_health_json"`
}

func (a *RunnerHealthcheckAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_healthcheck"
}

func (a *RunnerHealthcheckAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Sends a Daytona runner healthcheck. This action is intended for runner-agent integrations using runner credentials.",
		Attributes: map[string]actionschema.Attribute{
			"app_version": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Runner app version to report.",
			},
			"domain": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Runner domain to report.",
			},
			"proxy_url": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Runner proxy URL to report.",
			},
			"api_url": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Runner API URL to report.",
			},
			"metrics_json": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Runner metrics JSON matching Daytona's runner health metrics payload.",
			},
			"service_health_json": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "JSON array of runner service-health objects matching Daytona's serviceHealth payload.",
			},
		},
	}
}

func (a *RunnerHealthcheckAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *RunnerHealthcheckAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data runnerHealthcheckActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	appVersion := strings.TrimSpace(data.AppVersion.ValueString())
	if appVersion == "" {
		resp.Diagnostics.AddAttributeError(path.Root("app_version"), "Missing Daytona runner app version", "Configure app_version with the runner app version to report.")
		return
	}

	payload := *apiclient.NewRunnerHealthcheck(appVersion)
	if domain := optionalString(data.Domain); domain != nil {
		payload.SetDomain(*domain)
	}
	if proxyURL := optionalString(data.ProxyURL); proxyURL != nil {
		payload.SetProxyUrl(*proxyURL)
	}
	if apiURL := optionalString(data.APIURL); apiURL != nil {
		payload.SetApiUrl(*apiURL)
	}
	if !decodeOptionalJSON(data.MetricsJSON, "metrics_json", &payload.Metrics, &resp.Diagnostics) {
		return
	}
	if !decodeOptionalJSON(data.ServiceHealth, "service_health_json", &payload.ServiceHealth, &resp.Diagnostics) {
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Sending Daytona runner healthcheck."})
	}

	httpResp, err := a.client.api.RunnersAPI.RunnerHealthcheck(ctx).
		RunnerHealthcheck(payload).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to send Daytona runner healthcheck", "send runner healthcheck", httpResp, err)
	}
}

func decodeOptionalJSON(value types.String, attribute string, target any, diags *diag.Diagnostics) bool {
	raw := strings.TrimSpace(value.ValueString())
	if value.IsNull() || value.IsUnknown() || raw == "" {
		return true
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		diags.AddAttributeError(
			path.Root(attribute),
			"Invalid Daytona runner healthcheck JSON",
			fmt.Sprintf("%s must contain valid JSON for Daytona's runner healthcheck payload: %s", attribute, err),
		)
		return false
	}
	return true
}
