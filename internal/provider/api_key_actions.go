package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &APIKeyForUserRevokeAction{}
var _ action.ActionWithConfigure = &APIKeyForUserRevokeAction{}

func NewAPIKeyForUserRevokeAction() action.Action {
	return &APIKeyForUserRevokeAction{}
}

type APIKeyForUserRevokeAction struct {
	client *daytonaClient
}

type apiKeyForUserRevokeActionModel struct {
	UserID         types.String `tfsdk:"user_id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (a *APIKeyForUserRevokeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_revoke_api_key_for_user"
}

func (a *APIKeyForUserRevokeAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Revokes a Daytona API key by name for a specific organization user. Daytona allows the authenticated user to revoke their own key; revoking another user's key requires organization owner access.",
		Attributes: map[string]actionschema.Attribute{
			"user_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona user ID that owns the API key.",
			},
			"name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "API key name to revoke.",
			},
			"organization_id": organizationIDActionAttribute(),
		},
	}
}

func (a *APIKeyForUserRevokeAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *APIKeyForUserRevokeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data apiKeyForUserRevokeActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := strings.TrimSpace(data.UserID.ValueString())
	if userID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona user ID",
			"Configure the user_id attribute with the Daytona user ID that owns the API key.",
		)
		return
	}

	name := strings.TrimSpace(data.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona API key name",
			"Configure the name attribute with the Daytona API key name to revoke.",
		)
		return
	}

	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	request := a.client.api.ApiKeysAPI.DeleteApiKeyForUser(ctx, userID, name)
	if organizationID := optionalString(data.OrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Revoking Daytona API key for user."})
	}

	httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to revoke Daytona API key for user", "revoke API key for user", httpResp, err)
	}
}
