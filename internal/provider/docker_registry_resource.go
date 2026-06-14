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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DockerRegistryResource{}
var _ resource.ResourceWithImportState = &DockerRegistryResource{}

func NewDockerRegistryResource() resource.Resource {
	return &DockerRegistryResource{}
}

type DockerRegistryResource struct {
	client *daytonaClient
}

type dockerRegistryResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Project      types.String `tfsdk:"project"`
	RegistryType types.String `tfsdk:"registry_type"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *DockerRegistryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_registry"
}

func (r *DockerRegistryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Daytona Docker registry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Daytona Docker registry ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Registry name.",
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Registry URL.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Registry username.",
			},
			"password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Registry password or token.",
			},
			"project": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Registry project or namespace.",
			},
			"registry_type": computedStringAttribute("Registry type."),
			"created_at":    computedStringAttribute("Registry creation timestamp."),
			"updated_at":    computedStringAttribute("Registry update timestamp."),
		},
	}
}

func (r *DockerRegistryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DockerRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dockerRegistryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRegistry := apiclient.NewCreateDockerRegistry(
		data.Name.ValueString(),
		data.URL.ValueString(),
		data.Username.ValueString(),
		data.Password.ValueString(),
	)
	if value := optionalString(data.Project); value != nil {
		createRegistry.SetProject(*value)
	}

	registry, httpResp, err := r.client.api.DockerRegistryAPI.CreateRegistry(ctx).
		CreateDockerRegistry(*createRegistry).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to create Daytona Docker registry", "create Docker registry", httpResp, err)
		return
	}

	data = flattenDockerRegistry(registry, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DockerRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dockerRegistryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, httpResp, err := r.client.api.DockerRegistryAPI.GetRegistry(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona Docker registry", "read Docker registry", httpResp, err)
		return
	}

	data = flattenDockerRegistry(registry, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DockerRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dockerRegistryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRegistry := apiclient.NewUpdateDockerRegistry(
		data.Name.ValueString(),
		data.URL.ValueString(),
		data.Username.ValueString(),
	)
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		updateRegistry.SetPassword(data.Password.ValueString())
	}
	if value := optionalString(data.Project); value != nil {
		updateRegistry.SetProject(*value)
	}

	registry, httpResp, err := r.client.api.DockerRegistryAPI.UpdateRegistry(ctx, data.ID.ValueString()).
		UpdateDockerRegistry(*updateRegistry).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona Docker registry", "update Docker registry", httpResp, err)
		return
	}

	data = flattenDockerRegistry(registry, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DockerRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dockerRegistryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.api.DockerRegistryAPI.DeleteRegistry(ctx, data.ID.ValueString()).Execute()
	if isNotFound(httpResp) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to delete Daytona Docker registry", "delete Docker registry", httpResp, err)
		return
	}
}

func (r *DockerRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenDockerRegistry(registry *apiclient.DockerRegistry, prior dockerRegistryResourceModel) dockerRegistryResourceModel {
	if registry == nil {
		return prior
	}

	prior.ID = types.StringValue(registry.Id)
	prior.Name = types.StringValue(registry.Name)
	prior.URL = types.StringValue(registry.Url)
	prior.Username = types.StringValue(registry.Username)
	prior.Project = types.StringValue(registry.Project)
	prior.RegistryType = types.StringValue(registry.RegistryType)
	prior.CreatedAt = types.StringValue(registry.CreatedAt.Format(time.RFC3339))
	prior.UpdatedAt = types.StringValue(registry.UpdatedAt.Format(time.RFC3339))

	return prior
}
