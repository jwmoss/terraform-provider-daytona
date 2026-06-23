package provider

import (
	"context"
	"fmt"
	"time"

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
		item.Value = types.StringValue(apiKey.Value)
		item.Permissions = setStringValue(ctx, apiKey.Permissions)
		item.UserID = types.StringValue(apiKey.UserId)
		item.CreatedAt = terraformTimeString(apiKey.CreatedAt)
		if value, ok := apiKey.GetLastUsedAtOk(); ok && value != nil {
			item.LastUsedAt = terraformTimeString(*value)
		}
		if value, ok := apiKey.GetExpiresAtOk(); ok && value != nil {
			item.ExpiresAt = terraformTimeString(*value)
		}
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
		item.OrganizationID = types.StringValue(volume.OrganizationId)
		item.State = types.StringValue(string(volume.State))
		item.CreatedAt = types.StringValue(volume.CreatedAt)
		item.UpdatedAt = types.StringValue(volume.UpdatedAt)
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
		item.Region = types.StringValue(runner.Region)
		item.CreatedAt = types.StringValue(runner.CreatedAt)
		item.UpdatedAt = types.StringValue(runner.UpdatedAt)
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
			item.OrganizationID = types.StringValue(sandbox.OrganizationId)
			item.Target = types.StringValue(sandbox.Target)
			item.Public = types.BoolValue(sandbox.Public)
			item.CreatedAt = pointerStringValue(sandbox.CreatedAt)
			item.UpdatedAt = pointerStringValue(sandbox.UpdatedAt)
			if sandbox.State != nil {
				item.State = types.StringValue(string(*sandbox.State))
			}
			if sandbox.RunnerId != nil {
				item.RunnerID = types.StringValue(*sandbox.RunnerId)
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
			item.CreatedAt = types.StringValue(snapshot.CreatedAt.Format(time.RFC3339))
			item.UpdatedAt = types.StringValue(snapshot.UpdatedAt.Format(time.RFC3339))
			if snapshot.OrganizationId != nil {
				item.OrganizationID = types.StringValue(*snapshot.OrganizationId)
			}
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
		item.URL = types.StringValue(registry.Url)
		item.Username = types.StringValue(registry.Username)
		item.Project = types.StringValue(registry.Project)
		item.Type = types.StringValue(registry.RegistryType)
		item.CreatedAt = types.StringValue(registry.CreatedAt.Format(time.RFC3339))
		item.UpdatedAt = types.StringValue(registry.UpdatedAt.Format(time.RFC3339))
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
		item.CreatedBy = types.StringValue(organization.CreatedBy)
		item.Personal = types.BoolValue(organization.Personal)
		item.Suspended = types.BoolValue(organization.Suspended)
		item.CreatedAt = terraformTimeString(organization.CreatedAt)
		item.UpdatedAt = terraformTimeString(organization.UpdatedAt)
		if value, ok := organization.GetDefaultRegionIdOk(); ok && value != nil {
			item.DefaultRegionID = types.StringValue(*value)
		}
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
		item.OrganizationID = organizationID
		item.Name = types.StringValue(role.Name)
		item.Description = types.StringValue(role.Description)
		item.Permissions = setStringValue(ctx, role.Permissions)
		item.IsGlobal = types.BoolValue(role.IsGlobal)
		item.CreatedAt = terraformTimeString(role.CreatedAt)
		item.UpdatedAt = terraformTimeString(role.UpdatedAt)
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
		item.OrganizationID = types.StringValue(member.OrganizationId)
		item.Name = types.StringValue(member.Name)
		item.Email = types.StringValue(member.Email)
		item.Role = types.StringValue(member.Role)
		item.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(member.AssignedRoles))
		item.CreatedAt = terraformTimeString(member.CreatedAt)
		item.UpdatedAt = terraformTimeString(member.UpdatedAt)
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
		item.OrganizationID = types.StringValue(invitation.OrganizationId)
		item.OrganizationName = types.StringValue(invitation.OrganizationName)
		item.Email = types.StringValue(invitation.Email)
		item.InvitedBy = types.StringValue(invitation.InvitedBy)
		item.Role = types.StringValue(invitation.Role)
		item.AssignedRoleIDs = setStringValue(ctx, organizationRoleIDs(invitation.AssignedRoles))
		item.State = types.StringValue(invitation.Status)
		item.ExpiresAt = terraformTimeString(invitation.ExpiresAt)
		item.CreatedAt = terraformTimeString(invitation.CreatedAt)
		item.UpdatedAt = terraformTimeString(invitation.UpdatedAt)
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
		item.Type = types.StringValue(string(region.GetRegionType()))
		item.CreatedAt = types.StringValue(region.GetCreatedAt())
		item.UpdatedAt = types.StringValue(region.GetUpdatedAt())
		if value, ok := region.GetOrganizationIdOk(); ok && value != nil {
			item.OrganizationID = types.StringValue(*value)
		}
		items = append(items, item)
	}

	return items
}
