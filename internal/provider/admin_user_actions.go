package provider

import (
	"context"
	"encoding/json"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &AdminCreateUserAction{}
var _ action.ActionWithConfigure = &AdminCreateUserAction{}
var _ action.Action = &AdminRegenerateUserKeyPairAction{}
var _ action.ActionWithConfigure = &AdminRegenerateUserKeyPairAction{}

func NewAdminCreateUserAction() action.Action {
	return &AdminCreateUserAction{}
}

func NewAdminRegenerateUserKeyPairAction() action.Action {
	return &AdminRegenerateUserKeyPairAction{}
}

type AdminCreateUserAction struct {
	client *daytonaClient
}

type AdminRegenerateUserKeyPairAction struct {
	client *daytonaClient
}

type adminCreateUserActionModel struct {
	UserID                              types.String `tfsdk:"user_id"`
	Name                                types.String `tfsdk:"name"`
	Email                               types.String `tfsdk:"email"`
	PersonalOrganizationQuotaJSON       types.String `tfsdk:"personal_organization_quota_json"`
	PersonalOrganizationDefaultRegionID types.String `tfsdk:"personal_organization_default_region_id"`
	Role                                types.String `tfsdk:"role"`
	EmailVerified                       types.Bool   `tfsdk:"email_verified"`
}

type adminRegenerateUserKeyPairActionModel struct {
	UserID types.String `tfsdk:"user_id"`
}

func (a *AdminCreateUserAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_create_user"
}

func (a *AdminCreateUserAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Creates a Daytona user using Daytona admin APIs. Daytona exposes no matching admin delete/update user lifecycle endpoint, so this is a provider-defined action rather than a Terraform resource.",
		Attributes: map[string]actionschema.Attribute{
			"user_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona user ID to create.",
			},
			"name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "User display name.",
			},
			"email": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "User email address.",
			},
			"personal_organization_quota_json": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional personal organization quota payload as a JSON object matching Daytona's CreateOrganizationQuota shape.",
			},
			"personal_organization_default_region_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional default region ID for the user's personal organization.",
			},
			"role": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional Daytona system role for the user. Valid values are `admin` and `user`.",
			},
			"email_verified": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether Daytona should mark the user's email as verified.",
			},
		},
	}
}

func (a *AdminCreateUserAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *AdminCreateUserAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminCreateUserActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := strings.TrimSpace(data.UserID.ValueString())
	if userID == "" {
		resp.Diagnostics.AddError("Missing Daytona user ID", "Configure the user_id attribute with the Daytona user ID to create.")
		return
	}

	name := strings.TrimSpace(data.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddError("Missing Daytona user name", "Configure the name attribute with the Daytona user display name.")
		return
	}

	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	createUser := apiclient.NewCreateUser(userID, name)
	if !data.Email.IsNull() {
		createUser.SetEmail(strings.TrimSpace(data.Email.ValueString()))
	}
	if !data.PersonalOrganizationDefaultRegionID.IsNull() {
		createUser.SetPersonalOrganizationDefaultRegionId(strings.TrimSpace(data.PersonalOrganizationDefaultRegionID.ValueString()))
	}
	if !data.EmailVerified.IsNull() {
		createUser.SetEmailVerified(data.EmailVerified.ValueBool())
	}

	if !data.Role.IsNull() {
		role := strings.TrimSpace(data.Role.ValueString())
		switch role {
		case "admin", "user":
			createUser.SetRole(role)
		default:
			resp.Diagnostics.AddError("Invalid Daytona user role", "The role attribute must be either `admin` or `user`.")
			return
		}
	}

	if !data.PersonalOrganizationQuotaJSON.IsNull() {
		var quota apiclient.CreateOrganizationQuota
		if err := json.Unmarshal([]byte(data.PersonalOrganizationQuotaJSON.ValueString()), &quota); err != nil {
			resp.Diagnostics.AddError("Invalid Daytona personal organization quota JSON", "The personal_organization_quota_json attribute must be a valid JSON object matching Daytona's CreateOrganizationQuota shape: "+err.Error())
			return
		}
		var quotaObject map[string]interface{}
		if err := json.Unmarshal([]byte(data.PersonalOrganizationQuotaJSON.ValueString()), &quotaObject); err != nil || quotaObject == nil {
			resp.Diagnostics.AddError("Invalid Daytona personal organization quota JSON", "The personal_organization_quota_json attribute must be a JSON object.")
			return
		}
		createUser.SetPersonalOrganizationQuota(quota)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Creating Daytona admin user."})
	}

	httpResp, err := a.client.api.AdminAPI.AdminCreateUser(ctx).CreateUser(*createUser).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona admin user", "create admin user", httpResp, err)
	}
}

func (a *AdminRegenerateUserKeyPairAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_regenerate_user_key_pair"
}

func (a *AdminRegenerateUserKeyPairAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Regenerates a Daytona user's key pair using Daytona admin APIs.",
		Attributes: map[string]actionschema.Attribute{
			"user_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona user ID whose key pair should be regenerated.",
			},
		},
	}
}

func (a *AdminRegenerateUserKeyPairAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *AdminRegenerateUserKeyPairAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminRegenerateUserKeyPairActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := strings.TrimSpace(data.UserID.ValueString())
	if userID == "" {
		resp.Diagnostics.AddError("Missing Daytona user ID", "Configure the user_id attribute with the Daytona user ID whose key pair should be regenerated.")
		return
	}

	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Regenerating Daytona user key pair."})
	}

	httpResp, err := a.client.api.AdminAPI.AdminRegenerateKeyPair(ctx, userID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to regenerate Daytona admin user key pair", "regenerate admin user key pair", httpResp, err)
	}
}
