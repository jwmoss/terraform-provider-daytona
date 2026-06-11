// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strconv"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &UserLinkedAccountAction{}
var _ action.ActionWithConfigure = &UserLinkedAccountAction{}
var _ action.Action = &UserSmsMFAEnrollmentAction{}
var _ action.ActionWithConfigure = &UserSmsMFAEnrollmentAction{}

type userLinkedAccountActionOperation string

const (
	userLinkedAccountActionLink   userLinkedAccountActionOperation = "link"
	userLinkedAccountActionUnlink userLinkedAccountActionOperation = "unlink"
)

func NewUserLinkAccountAction() action.Action {
	return &UserLinkedAccountAction{operation: userLinkedAccountActionLink}
}

func NewUserUnlinkAccountAction() action.Action {
	return &UserLinkedAccountAction{operation: userLinkedAccountActionUnlink}
}

func NewUserSmsMFAEnrollmentAction() action.Action {
	return &UserSmsMFAEnrollmentAction{}
}

type UserLinkedAccountAction struct {
	client    *daytonaClient
	operation userLinkedAccountActionOperation
}

type userLinkedAccountActionModel struct {
	AccountProvider types.String `tfsdk:"account_provider"`
	ProviderUserID  types.String `tfsdk:"provider_user_id"`
}

func (a *UserLinkedAccountAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + string(a.operation) + "_account"
}

func (a *UserLinkedAccountAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: userLinkedAccountActionDescription(a.operation),
		Attributes: map[string]actionschema.Attribute{
			"account_provider": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona account provider name for the secondary account, such as a value returned by the `daytona_account_providers` data source.",
			},
			"provider_user_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Provider-specific user ID for the secondary account.",
			},
		},
	}
}

func (a *UserLinkedAccountAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *UserLinkedAccountAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data userLinkedAccountActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountProvider := strings.TrimSpace(data.AccountProvider.ValueString())
	if accountProvider == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona account provider",
			"Configure the account_provider attribute with the Daytona account provider name.",
		)
		return
	}

	providerUserID := strings.TrimSpace(data.ProviderUserID.ValueString())
	if providerUserID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona provider user ID",
			"Configure the provider_user_id attribute with the provider-specific user ID for the secondary account.",
		)
		return
	}

	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	switch a.operation {
	case userLinkedAccountActionLink:
		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Linking Daytona account."})
		}

		payload := apiclient.NewCreateLinkedAccount(accountProvider, providerUserID)
		httpResp, err := a.client.api.UsersAPI.LinkAccount(ctx).CreateLinkedAccount(*payload).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to link Daytona account", "link account", httpResp, err)
		}
	case userLinkedAccountActionUnlink:
		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{Message: "Unlinking Daytona account."})
		}

		httpResp, err := a.client.api.UsersAPI.UnlinkAccount(ctx, accountProvider, providerUserID).Execute()
		if err != nil {
			addAPIError(&resp.Diagnostics, "Unable to unlink Daytona account", "unlink account", httpResp, err)
		}
	default:
		resp.Diagnostics.AddError(
			"Unsupported Daytona account action",
			"Unsupported account action operation. Please report this issue to the provider developers.",
		)
	}
}

func userLinkedAccountActionDescription(operation userLinkedAccountActionOperation) string {
	switch operation {
	case userLinkedAccountActionLink:
		return "Links a secondary account to the authenticated Daytona user."
	case userLinkedAccountActionUnlink:
		return "Unlinks a secondary account from the authenticated Daytona user."
	default:
		return "Runs a Daytona linked-account action."
	}
}

type UserSmsMFAEnrollmentAction struct {
	client *daytonaClient
}

func (a *UserSmsMFAEnrollmentAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enroll_sms_mfa"
}

func (a *UserSmsMFAEnrollmentAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Starts SMS MFA enrollment for the authenticated Daytona user. Daytona returns an enrollment URL during action progress; treat Terraform action output and logs as sensitive.",
	}
}

func (a *UserSmsMFAEnrollmentAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *UserSmsMFAEnrollmentAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Starting Daytona SMS MFA enrollment."})
	}

	enrollmentURL, httpResp, err := a.client.api.UsersAPI.EnrollInSmsMfa(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to start Daytona SMS MFA enrollment", "enroll in SMS MFA", httpResp, err)
		return
	}

	enrollmentURL = normalizedUserActionString(enrollmentURL)
	if resp.SendProgress != nil && enrollmentURL != "" {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Daytona SMS MFA enrollment URL: " + enrollmentURL})
	}
}

func normalizedUserActionString(value string) string {
	value = strings.TrimSpace(value)
	if unquoted, err := strconv.Unquote(value); err == nil {
		return strings.TrimSpace(unquoted)
	}

	return value
}
