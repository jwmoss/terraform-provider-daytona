package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &AdminRecoverSandboxAction{}
var _ action.ActionWithConfigure = &AdminRecoverSandboxAction{}

func NewAdminRecoverSandboxAction() action.Action {
	return &AdminRecoverSandboxAction{}
}

type AdminRecoverSandboxAction struct {
	client *daytonaClient
}

type adminRecoverSandboxActionModel struct {
	SandboxID types.String `tfsdk:"sandbox_id"`
}

func (a *AdminRecoverSandboxAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_recover_sandbox"
}

func (a *AdminRecoverSandboxAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Recovers a Daytona sandbox from an error state using Daytona admin APIs.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona sandbox ID to recover.",
			},
		},
	}
}

func (a *AdminRecoverSandboxAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *AdminRecoverSandboxAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminRecoverSandboxActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxID := strings.TrimSpace(data.SandboxID.ValueString())
	if sandboxID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID",
			"Configure the sandbox_id attribute with the Daytona sandbox ID to recover as an admin.",
		)
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Recovering Daytona sandbox as admin."})
	}

	_, httpResp, err := a.client.api.AdminAPI.AdminRecoverSandbox(ctx, sandboxID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to recover Daytona sandbox as admin", "admin recover sandbox", httpResp, err)
	}
}
