package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &SandboxExpireSignedPortPreviewURLAction{}
var _ action.ActionWithConfigure = &SandboxExpireSignedPortPreviewURLAction{}
var _ action.Action = &SandboxRevokeSSHAccessAction{}
var _ action.ActionWithConfigure = &SandboxRevokeSSHAccessAction{}

func NewSandboxExpireSignedPortPreviewURLAction() action.Action {
	return &SandboxExpireSignedPortPreviewURLAction{}
}

func NewSandboxRevokeSSHAccessAction() action.Action {
	return &SandboxRevokeSSHAccessAction{}
}

type SandboxExpireSignedPortPreviewURLAction struct {
	client *daytonaClient
}

type sandboxExpireSignedPortPreviewURLActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	Port            types.Int64  `tfsdk:"port"`
	Token           types.String `tfsdk:"token"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxExpireSignedPortPreviewURLAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_expire_sandbox_signed_port_preview_url"
}

func (a *SandboxExpireSignedPortPreviewURLAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Expires a Daytona sandbox signed port preview URL token.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona sandbox ID or name.",
			},
			"port": actionschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Sandbox port number for the signed preview URL.",
			},
			"token": actionschema.StringAttribute{
				Required:            true,
				WriteOnly:           true,
				MarkdownDescription: "Signed preview URL token to expire. This value is write-only and is not returned by the provider.",
			},
			"organization_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona organization ID to send as `X-Daytona-Organization-ID` for this action. Defaults to the provider-level organization ID when configured.",
			},
		},
	}
}

func (a *SandboxExpireSignedPortPreviewURLAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *SandboxExpireSignedPortPreviewURLAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxExpireSignedPortPreviewURLActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to operate on.",
		)
		return
	}

	port, ok := int32Port(data.Port.ValueInt64())
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Daytona sandbox port",
			"Configure port with an integer from 1 to 65535.",
		)
		return
	}

	token := strings.TrimSpace(data.Token.ValueString())
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona signed preview URL token",
			"Configure the token attribute with the signed preview URL token to expire.",
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

	request := a.client.api.SandboxAPI.ExpireSignedPortPreviewUrl(ctx, sandboxIDOrName, port, token)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Expiring Daytona sandbox signed port preview URL."})
	}

	httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to expire Daytona sandbox signed port preview URL", "expire sandbox signed port preview URL", httpResp, err)
	}
}

type SandboxRevokeSSHAccessAction struct {
	client *daytonaClient
}

type sandboxRevokeSSHAccessActionModel struct {
	SandboxIDOrName types.String `tfsdk:"sandbox_id_or_name"`
	Token           types.String `tfsdk:"token"`
	OrganizationID  types.String `tfsdk:"organization_id"`
}

func (a *SandboxRevokeSSHAccessAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_revoke_sandbox_ssh_access"
}

func (a *SandboxRevokeSSHAccessAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Revokes Daytona sandbox SSH access.",
		Attributes: map[string]actionschema.Attribute{
			"sandbox_id_or_name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona sandbox ID or name.",
			},
			"token": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Optional SSH access token to revoke. When omitted, Daytona revokes all SSH access for the sandbox. This value is write-only and is not returned by the provider.",
			},
			"organization_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Daytona organization ID to send as `X-Daytona-Organization-ID` for this action. Defaults to the provider-level organization ID when configured.",
			},
		},
	}
}

func (a *SandboxRevokeSSHAccessAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *SandboxRevokeSSHAccessAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data sandboxRevokeSSHAccessActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sandboxIDOrName := strings.TrimSpace(data.SandboxIDOrName.ValueString())
	if sandboxIDOrName == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona sandbox ID or name",
			"Configure the sandbox_id_or_name attribute with the Daytona sandbox ID or name to operate on.",
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

	request := a.client.api.SandboxAPI.RevokeSshAccess(ctx, sandboxIDOrName)
	if token := strings.TrimSpace(data.Token.ValueString()); token != "" {
		request = request.Token(token)
	}
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Revoking Daytona sandbox SSH access."})
	}

	_, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to revoke Daytona sandbox SSH access", "revoke sandbox SSH access", httpResp, err)
	}
}

func int32Port(port int64) (int32, bool) {
	if port < 1 || port > 65535 {
		return 0, false
	}

	return int32(port), true
}
