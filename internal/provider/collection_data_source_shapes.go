package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var collectionShapes = map[string]collectionShape{
	"api_keys": {
		kind: "api_keys",
		read: readAPIKeyCollection,
	},
	"volumes": {
		kind: "volumes",
		read: readVolumeCollection,
	},
	"regions": {
		kind: "regions",
		read: readRegionCollection,
	},
	"shared_regions": {
		kind: "shared_regions",
		read: readSharedRegionCollection,
	},
	"runners": {
		kind:                "runners",
		markdownDescription: "Lists Daytona custom runners visible to the configured credentials. Daytona exposes this endpoint only when organization infrastructure is enabled for the organization.",
		read:                readRunnerCollection,
	},
	"sandboxes": {
		kind: "sandboxes",
		read: readSandboxCollection,
	},
	"snapshots": {
		kind: "snapshots",
		read: readSnapshotCollection,
	},
	"docker_registries": {
		kind: "docker_registries",
		read: readDockerRegistryCollection,
	},
	"organizations": {
		kind: "organizations",
		read: readOrganizationCollection,
	},
	"organization_roles": {
		kind:                   "organization_roles",
		requiresOrganizationID: true,
		read:                   readOrganizationRoleCollection,
	},
	"organization_members": {
		kind:                   "organization_members",
		requiresOrganizationID: true,
		read:                   readOrganizationMemberCollection,
	},
	"organization_invitations": {
		kind:                   "organization_invitations",
		requiresOrganizationID: true,
		read:                   readOrganizationInvitationCollection,
	},
}

func readAPIKeyCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	apiKeys, httpResp, err := client.api.ApiKeysAPI.ListApiKeys(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona API keys", "list API keys", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		item := newCollectionItem()
		item.ID = types.StringValue(apiKey.Name)
		item.Name = types.StringValue(apiKey.Name)
		items = append(items, item)
	}

	return items, nil
}

func readVolumeCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	volumes, httpResp, err := client.api.VolumesAPI.ListVolumes(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona volumes", "list volumes", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(volumes))
	for _, volume := range volumes {
		item := newCollectionItem()
		item.ID = types.StringValue(volume.Id)
		item.Name = types.StringValue(volume.Name)
		item.State = types.StringValue(string(volume.State))
		items = append(items, item)
	}

	return items, nil
}

func readRegionCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	regions, httpResp, err := client.api.OrganizationsAPI.ListAvailableRegions(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona regions", "list regions", httpResp, err)
		return nil, err
	}

	return regionCollectionItems(regions), nil
}

func readSharedRegionCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	regions, httpResp, err := client.api.RegionsAPI.ListSharedRegions(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona shared regions", "list shared regions", httpResp, err)
		return nil, err
	}

	return regionCollectionItems(regions), nil
}

func readRunnerCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	runners, httpResp, err := client.api.RunnersAPI.ListRunners(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona runners", "list runners", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(runners))
	for _, runner := range runners {
		item := newCollectionItem()
		item.ID = types.StringValue(runner.Id)
		item.Name = types.StringValue(runner.Name)
		item.State = types.StringValue(string(runner.State))
		items = append(items, item)
	}

	return items, nil
}

func readSandboxCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	var items []collectionItemModel
	cursor := ""

	for {
		request := client.api.SandboxAPI.ListSandboxes(ctx).Limit(100)
		if cursor != "" {
			request = request.Cursor(cursor)
		}
		sandboxes, httpResp, err := request.Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona sandboxes", "list sandboxes", httpResp, err)
			return nil, err
		}
		if sandboxes == nil {
			addEmptyAPIResponseError(diags, "Empty Daytona sandboxes response", "list sandboxes", httpResp)
			return nil, fmt.Errorf("empty Daytona sandboxes response")
		}

		for _, sandbox := range sandboxes.Items {
			item := newCollectionItem()
			item.ID = types.StringValue(sandbox.Id)
			item.Name = types.StringValue(sandbox.Name)
			if sandbox.State != nil {
				item.State = types.StringValue(string(*sandbox.State))
			}
			items = append(items, item)
		}

		next := sandboxes.NextCursor.Get()
		if next == nil || *next == "" || *next == cursor || len(sandboxes.Items) == 0 {
			break
		}
		cursor = *next
	}

	return items, nil
}

func readSnapshotCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	var items []collectionItemModel

	for page := float32(1); ; page++ {
		snapshots, httpResp, err := client.api.SnapshotsAPI.GetAllSnapshots(ctx).Page(page).Limit(100).Execute()
		if err != nil {
			addAPIError(diags, "Unable to list Daytona snapshots", "list snapshots", httpResp, err)
			return nil, err
		}
		if snapshots == nil {
			addEmptyAPIResponseError(diags, "Empty Daytona snapshots response", "list snapshots", httpResp)
			return nil, fmt.Errorf("empty Daytona snapshots response")
		}

		for _, snapshot := range snapshots.Items {
			item := newCollectionItem()
			item.ID = types.StringValue(snapshot.Id)
			item.Name = types.StringValue(snapshot.Name)
			item.State = types.StringValue(string(snapshot.State))
			items = append(items, item)
		}
		if page >= snapshots.TotalPages || len(snapshots.Items) == 0 {
			break
		}
	}

	return items, nil
}

func readDockerRegistryCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	registries, httpResp, err := client.api.DockerRegistryAPI.ListRegistries(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona Docker registries", "list Docker registries", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(registries))
	for _, registry := range registries {
		item := newCollectionItem()
		item.ID = types.StringValue(registry.Id)
		item.Name = types.StringValue(registry.Name)
		items = append(items, item)
	}

	return items, nil
}

func readOrganizationCollection(ctx context.Context, client *daytonaClient, _ types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	organizations, httpResp, err := client.api.OrganizationsAPI.ListOrganizations(ctx).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona organizations", "list organizations", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(organizations))
	for _, organization := range organizations {
		item := newCollectionItem()
		item.ID = types.StringValue(organization.Id)
		item.Name = types.StringValue(organization.Name)
		items = append(items, item)
	}

	return items, nil
}

func readOrganizationRoleCollection(ctx context.Context, client *daytonaClient, organizationID types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	roles, httpResp, err := client.api.OrganizationsAPI.ListOrganizationRoles(ctx, organizationID.ValueString()).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona organization roles", "list organization roles", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(roles))
	for _, role := range roles {
		item := newCollectionItem()
		item.ID = types.StringValue(role.Id)
		item.Name = types.StringValue(role.Name)
		items = append(items, item)
	}

	return items, nil
}

func readOrganizationMemberCollection(ctx context.Context, client *daytonaClient, organizationID types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	members, httpResp, err := client.api.OrganizationsAPI.ListOrganizationMembers(ctx, organizationID.ValueString()).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona organization members", "list organization members", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(members))
	for _, member := range members {
		item := newCollectionItem()
		item.ID = types.StringValue(member.UserId)
		item.Name = types.StringValue(member.Name)
		items = append(items, item)
	}

	return items, nil
}

func readOrganizationInvitationCollection(ctx context.Context, client *daytonaClient, organizationID types.String, diags *diag.Diagnostics) ([]collectionItemModel, error) {
	invitations, httpResp, err := client.api.OrganizationsAPI.ListOrganizationInvitations(ctx, organizationID.ValueString()).Execute()
	if err != nil {
		addAPIError(diags, "Unable to list Daytona organization invitations", "list organization invitations", httpResp, err)
		return nil, err
	}

	items := make([]collectionItemModel, 0, len(invitations))
	for _, invitation := range invitations {
		item := newCollectionItem()
		item.ID = types.StringValue(invitation.Id)
		item.Name = types.StringValue(invitation.Email)
		item.State = types.StringValue(invitation.Status)
		items = append(items, item)
	}

	return items, nil
}

func regionCollectionItems(regions []apiclient.Region) []collectionItemModel {
	items := make([]collectionItemModel, 0, len(regions))
	for i := range regions {
		region := &regions[i]
		item := newCollectionItem()
		item.ID = types.StringValue(region.GetId())
		item.Name = types.StringValue(region.GetName())
		items = append(items, item)
	}

	return items
}
