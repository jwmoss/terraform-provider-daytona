package provider

import (
	"context"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &SandboxUpdateStateAction{}
var _ action.ActionWithConfigure = &SandboxUpdateStateAction{}

func NewSandboxUpdateStateAction() action.Action {
	return &SandboxUpdateStateAction{}
}

type SandboxUpdateStateAction struct {
	client *daytonaClient
}

type sandboxUpdateStateActionModel struct {
	SandboxID      types.String `tfsdk:"sandbox_id"`
	State          types.String `tfsdk:"state"`
	ErrorReason    types.String `tfsdk:"error_reason"`
	Recoverable    types.Bool   `tfsdk:"recoverable"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (a *SandboxUpdateStateAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_update_sandbox_state"
}

func (a *SandboxUpdateStateAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Updates a Daytona sandbox state directly. This action is intended for runner/control-plane integrations; prefer the start, stop, archive, and recover actions for normal lifecycle operations.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona sandbox ID.",
			},
			"state": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("New Daytona sandbox state. Supported values are: %s.", strings.Join(sandboxStateValues(), ", ")),
			},
			"error_reason": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Optional error reason to attach when setting an error state.",
			},
			"recoverable": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether Daytona should mark the sandbox state as recoverable.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *SandboxUpdateStateAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *SandboxUpdateStateAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxUpdateStateActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	sandboxID := strings.TrimSpace(data.SandboxID.ValueString())
	if sandboxID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("sandbox_id"), "Missing Daytona sandbox ID", "Configure sandbox_id with the Daytona sandbox ID to update.")
		return
	}

	state := apiclient.SandboxState(strings.TrimSpace(data.State.ValueString()))
	if !state.IsValid() || state == apiclient.SANDBOXSTATE_UNKNOWN_DEFAULT_OPEN_API {
		resp.Diagnostics.AddAttributeError(
			path.Root("state"),
			"Invalid Daytona sandbox state",
			fmt.Sprintf("State must be one of %s.", strings.Join(sandboxStateValues(), ", ")),
		)
		return
	}

	payload := *apiclient.NewUpdateSandboxStateDto(string(state))
	if errorReason := optionalString(data.ErrorReason); errorReason != nil {
		payload.SetErrorReason(*errorReason)
	}
	if recoverable := optionalBool(data.Recoverable); recoverable != nil {
		payload.SetRecoverable(*recoverable)
	}

	request := a.client.api.SandboxAPI.UpdateSandboxState(ctx, sandboxID).
		UpdateSandboxStateDto(payload)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Updating Daytona sandbox state."})
	}

	httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona sandbox state", "update sandbox state", httpResp, err)
	}
}
