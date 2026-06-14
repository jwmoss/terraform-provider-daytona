package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AdminUserDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminUserDataSource{}
var _ datasource.DataSource = &AdminUsersDataSource{}
var _ datasource.DataSourceWithConfigure = &AdminUsersDataSource{}

func NewAdminUserDataSource() datasource.DataSource {
	return &AdminUserDataSource{}
}

func NewAdminUsersDataSource() datasource.DataSource {
	return &AdminUsersDataSource{}
}

type AdminUserDataSource struct {
	client *daytonaClient
}

type AdminUsersDataSource struct {
	client *daytonaClient
}

type adminUserDataSourceConfigModel struct {
	UserID types.String `tfsdk:"user_id"`
}

type adminUserDataSourceModel struct {
	UserID     types.String              `tfsdk:"user_id"`
	ID         types.String              `tfsdk:"id"`
	Name       types.String              `tfsdk:"name"`
	Email      types.String              `tfsdk:"email"`
	PublicKeys []adminUserPublicKeyModel `tfsdk:"public_keys"`
	CreatedAt  types.String              `tfsdk:"created_at"`
}

type adminUsersDataSourceModel struct {
	ID    types.String         `tfsdk:"id"`
	Items []adminUserItemModel `tfsdk:"items"`
}

type adminUserItemModel struct {
	ID         types.String              `tfsdk:"id"`
	Name       types.String              `tfsdk:"name"`
	Email      types.String              `tfsdk:"email"`
	PublicKeys []adminUserPublicKeyModel `tfsdk:"public_keys"`
	CreatedAt  types.String              `tfsdk:"created_at"`
}

type adminUserPublicKeyModel struct {
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (d *AdminUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_user"
}

func (d *AdminUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Daytona user by ID using Daytona admin APIs.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona user ID to read.",
			},
			"id":         computedDataSourceStringAttribute("Daytona user ID."),
			"name":       computedDataSourceStringAttribute("User display name."),
			"email":      computedDataSourceStringAttribute("User email address."),
			"created_at": computedDataSourceStringAttribute("User creation timestamp."),
			"public_keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "User public SSH keys.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": computedDataSourceStringAttribute("Public key name."),
						"key":  computedDataSourceStringAttribute("Public key value."),
					},
				},
			},
		},
	}
}

func (d *AdminUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config adminUserDataSourceConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := strings.TrimSpace(config.UserID.ValueString())
	if userID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona user ID",
			"Configure the user_id attribute with the Daytona user ID to read.",
		)
		return
	}

	user, httpResp, err := d.client.api.AdminAPI.AdminGetUser(ctx, userID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona admin user", "read admin user", httpResp, err)
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Empty Daytona admin user response", fmt.Sprintf("Daytona returned a successful response without user %q.", userID))
		return
	}

	data := flattenAdminUser(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *AdminUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_users"
}

func (d *AdminUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Daytona users using Daytona admin APIs.",
		Attributes: map[string]schema.Attribute{
			"id": computedDataSourceStringAttribute("Data source identifier."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona users.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: adminUserNestedAttributes(),
				},
			},
		},
	}
}

func (d *AdminUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *AdminUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	httpResp, err := d.client.api.AdminAPI.AdminListUsers(ctx).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to list Daytona admin users", "list admin users", httpResp, err)
		return
	}
	if httpResp == nil || httpResp.Body == nil {
		resp.Diagnostics.AddError("Empty Daytona admin users response", "Daytona returned a successful response without a response body.")
		return
	}

	body, readErr := io.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	httpResp.Body = io.NopCloser(bytes.NewBuffer(body))
	if readErr != nil {
		resp.Diagnostics.AddError("Unable to read Daytona admin users response", readErr.Error())
		return
	}

	var users []apiclient.User
	if err := json.Unmarshal(body, &users); err != nil {
		resp.Diagnostics.AddError("Unable to decode Daytona admin users response", err.Error())
		return
	}

	items := make([]adminUserItemModel, 0, len(users))
	for i := range users {
		items = append(items, flattenAdminUserItem(&users[i]))
	}

	data := adminUsersDataSourceModel{
		ID:    types.StringValue("admin_users"),
		Items: items,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func adminUserNestedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":         computedDataSourceStringAttribute("Daytona user ID."),
		"name":       computedDataSourceStringAttribute("User display name."),
		"email":      computedDataSourceStringAttribute("User email address."),
		"created_at": computedDataSourceStringAttribute("User creation timestamp."),
		"public_keys": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "User public SSH keys.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name": computedDataSourceStringAttribute("Public key name."),
					"key":  computedDataSourceStringAttribute("Public key value."),
				},
			},
		},
	}
}

func flattenAdminUser(user *apiclient.User) adminUserDataSourceModel {
	item := flattenAdminUserItem(user)

	return adminUserDataSourceModel{
		UserID:     types.StringValue(user.Id),
		ID:         item.ID,
		Name:       item.Name,
		Email:      item.Email,
		PublicKeys: item.PublicKeys,
		CreatedAt:  item.CreatedAt,
	}
}

func flattenAdminUserItem(user *apiclient.User) adminUserItemModel {
	publicKeys := make([]adminUserPublicKeyModel, 0, len(user.PublicKeys))
	for _, publicKey := range user.PublicKeys {
		publicKeys = append(publicKeys, adminUserPublicKeyModel{
			Name: types.StringValue(publicKey.Name),
			Key:  types.StringValue(publicKey.Key),
		})
	}

	return adminUserItemModel{
		ID:         types.StringValue(user.Id),
		Name:       types.StringValue(user.Name),
		Email:      types.StringValue(user.Email),
		PublicKeys: publicKeys,
		CreatedAt:  terraformTimeString(user.CreatedAt),
	}
}
