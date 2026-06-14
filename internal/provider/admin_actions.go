package provider

import (
	"context"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &AdminSetDefaultDockerRegistryAction{}
var _ action.ActionWithConfigure = &AdminSetDefaultDockerRegistryAction{}
var _ action.Action = &AdminSetSnapshotGeneralStatusAction{}
var _ action.ActionWithConfigure = &AdminSetSnapshotGeneralStatusAction{}

func NewAdminSetDefaultDockerRegistryAction() action.Action {
	return &AdminSetDefaultDockerRegistryAction{}
}

func NewAdminSetSnapshotGeneralStatusAction() action.Action {
	return &AdminSetSnapshotGeneralStatusAction{}
}

type AdminSetDefaultDockerRegistryAction struct {
	client *daytonaClient
}

type AdminSetSnapshotGeneralStatusAction struct {
	client *daytonaClient
}

type adminSetDefaultDockerRegistryActionModel struct {
	RegistryID types.String `tfsdk:"registry_id"`
}

type adminSetSnapshotGeneralStatusActionModel struct {
	SnapshotID types.String `tfsdk:"snapshot_id"`
	General    types.Bool   `tfsdk:"general"`
}

func (a *AdminSetDefaultDockerRegistryAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_set_default_docker_registry"
}

func (a *AdminSetDefaultDockerRegistryAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Sets a Daytona Docker registry as the admin default registry. This action requires Daytona admin privileges.",
		Attributes: map[string]actionschema.Attribute{
			"registry_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona Docker registry ID to make the default registry.",
			},
		},
	}
}

func (a *AdminSetDefaultDockerRegistryAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *AdminSetDefaultDockerRegistryAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminSetDefaultDockerRegistryActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registryID := strings.TrimSpace(data.RegistryID.ValueString())
	if registryID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona Docker registry ID",
			"Configure registry_id with the Daytona Docker registry ID to set as the default registry.",
		)
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError("Unconfigured Daytona client", "The provider did not configure a Daytona API client for this action.")
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Setting Daytona default Docker registry."})
	}

	_, httpResp, err := a.client.api.AdminAPI.AdminSetDefaultRegistry(ctx, registryID).Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to set Daytona default Docker registry", "set default Docker registry", httpResp, err)
	}
}

func (a *AdminSetSnapshotGeneralStatusAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_set_snapshot_general_status"
}

func (a *AdminSetSnapshotGeneralStatusAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Sets Daytona snapshot general status through the Daytona admin API. This action requires Daytona admin privileges.",
		Attributes: map[string]actionschema.Attribute{
			"snapshot_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona snapshot ID.",
			},
			"general": actionschema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the snapshot should be marked as general.",
			},
		},
	}
}

func (a *AdminSetSnapshotGeneralStatusAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *AdminSetSnapshotGeneralStatusAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data adminSetSnapshotGeneralStatusActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotID := strings.TrimSpace(data.SnapshotID.ValueString())
	if snapshotID == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona snapshot ID",
			"Configure snapshot_id with the Daytona snapshot ID to update.",
		)
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError("Unconfigured Daytona client", "The provider did not configure a Daytona API client for this action.")
		return
	}

	payload := *apiclient.NewSetSnapshotGeneralStatusDto(data.General.ValueBool())

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Setting Daytona snapshot general status."})
	}

	_, httpResp, err := a.client.api.AdminAPI.AdminSetSnapshotGeneralStatus(ctx, snapshotID).
		SetSnapshotGeneralStatusDto(payload).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to set Daytona snapshot general status", "set snapshot general status", httpResp, err)
	}
}
