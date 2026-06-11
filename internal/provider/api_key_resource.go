// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &APIKeyResource{}
var _ resource.ResourceWithImportState = &APIKeyResource{}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

type APIKeyResource struct {
	client *daytonaClient
}

type apiKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Permissions types.Set    `tfsdk:"permissions"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	Value       types.String `tfsdk:"value"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
	UserID      types.String `tfsdk:"user_id"`
}

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API key name, used as the resource ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "API key name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Daytona organization resource permissions assigned to the API key.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional RFC3339 timestamp when the API key expires.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "API key value. Daytona returns this only when the key is created.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API key creation timestamp.",
			},
			"last_used_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API key last-used timestamp, when available.",
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the user who owns the API key.",
			},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*daytonaClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *daytonaClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data apiKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := make([]string, 0)
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createAPIKey := apiclient.NewCreateApiKey(data.Name.ValueString(), permissions)
	if !data.ExpiresAt.IsNull() && data.ExpiresAt.ValueString() != "" {
		expiresAt, err := time.Parse(time.RFC3339, data.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("expires_at"),
				"Invalid expires_at timestamp",
				"expires_at must be formatted as RFC3339, for example 2026-12-31T23:59:59Z.",
			)
			return
		}
		createAPIKey.SetExpiresAt(expiresAt)
	}

	created, httpResp, err := r.client.api.ApiKeysAPI.CreateApiKey(ctx).
		CreateApiKey(*createAPIKey).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona API key", "create API key", httpResp, err)
		return
	}

	data = flattenAPIKeyResponse(created, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data apiKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, httpResp, err := r.client.api.ApiKeysAPI.GetApiKey(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona API key", "read API key", httpResp, err)
		return
	}

	data = flattenAPIKeyList(apiKey, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Daytona API key cannot be updated",
		"Daytona API key attributes are immutable through the API. Terraform should have planned replacement for configurable changes.",
	)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data apiKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.ApiKeysAPI.DeleteApiKey(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona API key", "delete API key", httpResp, err)
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenAPIKeyResponse(apiKey *apiclient.ApiKeyResponse, prior apiKeyResourceModel) apiKeyResourceModel {
	if apiKey == nil {
		return prior
	}

	prior.ID = types.StringValue(apiKey.Name)
	prior.Name = types.StringValue(apiKey.Name)
	prior.Value = types.StringValue(apiKey.Value)
	prior.CreatedAt = types.StringValue(apiKey.CreatedAt.Format(time.RFC3339))

	if expiresAt, ok := apiKey.GetExpiresAtOk(); ok && expiresAt != nil {
		prior.ExpiresAt = types.StringValue(expiresAt.Format(time.RFC3339))
	} else if prior.ExpiresAt.IsUnknown() {
		prior.ExpiresAt = types.StringNull()
	}

	permissions, _ := types.SetValueFrom(context.Background(), types.StringType, apiKey.Permissions)
	prior.Permissions = permissions

	return prior
}

func flattenAPIKeyList(apiKey *apiclient.ApiKeyList, prior apiKeyResourceModel) apiKeyResourceModel {
	if apiKey == nil {
		return prior
	}

	prior.ID = types.StringValue(apiKey.Name)
	prior.Name = types.StringValue(apiKey.Name)
	if prior.Value.IsNull() || prior.Value.IsUnknown() {
		prior.Value = types.StringValue(apiKey.Value)
	}
	prior.CreatedAt = types.StringValue(apiKey.CreatedAt.Format(time.RFC3339))
	prior.UserID = types.StringValue(apiKey.UserId)

	if lastUsedAt, ok := apiKey.GetLastUsedAtOk(); ok && lastUsedAt != nil {
		prior.LastUsedAt = types.StringValue(lastUsedAt.Format(time.RFC3339))
	} else {
		prior.LastUsedAt = types.StringNull()
	}

	if expiresAt, ok := apiKey.GetExpiresAtOk(); ok && expiresAt != nil {
		prior.ExpiresAt = types.StringValue(expiresAt.Format(time.RFC3339))
	} else if prior.ExpiresAt.IsUnknown() {
		prior.ExpiresAt = types.StringNull()
	}

	permissions, _ := types.SetValueFrom(context.Background(), types.StringType, apiKey.Permissions)
	prior.Permissions = permissions

	return prior
}
