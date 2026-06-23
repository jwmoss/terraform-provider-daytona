package provider

import (
	"context"
	"fmt"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sandboxChangeApplication struct {
	client *daytonaClient
}

func (a sandboxChangeApplication) apply(ctx context.Context, state, plan sandboxResourceModel, diags *diag.Diagnostics) (sandboxResourceModel, bool) {
	plannedAutoStopInterval := plan.AutoStopInterval
	plannedAutoArchiveInterval := plan.AutoArchiveInterval
	plannedAutoDeleteInterval := plan.AutoDeleteInterval

	if !plan.Public.Equal(state.Public) {
		sandbox, response, err := a.client.api.SandboxAPI.UpdatePublicStatus(ctx, state.ID.ValueString(), plan.Public.ValueBool()).Execute()
		if err != nil {
			addAPIError(diags, "Unable to update Daytona sandbox public status", "update sandbox public status", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if !plan.NetworkBlockAll.Equal(state.NetworkBlockAll) || !plan.NetworkAllowList.Equal(state.NetworkAllowList) {
		updateNetworkSettings := apiclient.NewUpdateSandboxNetworkSettings()
		if !plan.NetworkBlockAll.IsNull() && !plan.NetworkBlockAll.IsUnknown() {
			updateNetworkSettings.SetNetworkBlockAll(plan.NetworkBlockAll.ValueBool())
		}
		if value := optionalString(plan.NetworkAllowList); value != nil {
			updateNetworkSettings.SetNetworkAllowList(*value)
		} else if !plan.NetworkAllowList.Equal(state.NetworkAllowList) {
			// Removing network_allow_list from config must clear it server-side,
			// otherwise the old value re-imports on refresh as a perma-diff.
			updateNetworkSettings.SetNetworkAllowList("")
		}

		sandbox, response, err := a.client.api.SandboxAPI.UpdateNetworkSettings(ctx, state.ID.ValueString()).
			UpdateSandboxNetworkSettings(*updateNetworkSettings).
			Execute()
		if err != nil {
			addAPIError(diags, "Unable to update Daytona sandbox network settings", "update sandbox network settings", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if !plan.Labels.Equal(state.Labels) {
		labels, mapDiags := stringMap(ctx, plan.Labels)
		diags.Append(mapDiags...)
		if diags.HasError() {
			return plan, false
		}

		sandboxLabels := apiclient.NewSandboxLabels(labels)
		labelResponse, response, err := a.client.api.SandboxAPI.ReplaceLabels(ctx, state.ID.ValueString()).
			SandboxLabels(*sandboxLabels).
			Execute()
		if err != nil {
			addAPIError(diags, "Unable to replace Daytona sandbox labels", "replace sandbox labels", response, err)
			return plan, false
		}
		if labelResponse != nil {
			plan.Labels = stringMapValue(ctx, labelResponse.Labels)
		}
	}

	if !plan.CPU.Equal(state.CPU) || !plan.Memory.Equal(state.Memory) || !plan.Disk.Equal(state.Disk) {
		resizeSandbox := apiclient.NewResizeSandbox()
		hasResize := false
		if !plan.CPU.Equal(state.CPU) {
			if value := optionalInt32(plan.CPU); value != nil {
				resizeSandbox.SetCpu(*value)
				hasResize = true
			}
		}
		if !plan.Memory.Equal(state.Memory) {
			if value := optionalInt32(plan.Memory); value != nil {
				resizeSandbox.SetMemory(*value)
				hasResize = true
			}
		}
		if !plan.Disk.Equal(state.Disk) {
			if value := optionalInt32(plan.Disk); value != nil {
				resizeSandbox.SetDisk(*value)
				hasResize = true
			}
		}
		if hasResize {
			sandbox, response, err := a.client.api.SandboxAPI.ResizeSandbox(ctx, state.ID.ValueString()).
				ResizeSandbox(*resizeSandbox).
				Execute()
			if err != nil {
				addAPIError(diags, "Unable to resize Daytona sandbox", "resize sandbox", response, err)
				return plan, false
			}
			plan = flattenSandbox(ctx, sandbox, plan)
		}
	}

	if value, ok := sandboxIntervalUpdateValue(plannedAutoStopInterval, state.AutoStopInterval, 0); ok {
		sandbox, response, err := a.client.api.SandboxAPI.SetAutostopInterval(ctx, state.ID.ValueString(), value).Execute()
		if err != nil {
			addAPIError(diags, "Unable to update Daytona sandbox auto-stop interval", "set sandbox auto-stop interval", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if value, ok := sandboxIntervalUpdateValue(plannedAutoArchiveInterval, state.AutoArchiveInterval, 0); ok {
		sandbox, response, err := a.client.api.SandboxAPI.SetAutoArchiveInterval(ctx, state.ID.ValueString(), value).Execute()
		if err != nil {
			addAPIError(diags, "Unable to update Daytona sandbox auto-archive interval", "set sandbox auto-archive interval", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if value, ok := sandboxIntervalUpdateValue(plannedAutoDeleteInterval, state.AutoDeleteInterval, -1); ok {
		sandbox, response, err := a.client.api.SandboxAPI.SetAutoDeleteInterval(ctx, state.ID.ValueString(), value).Execute()
		if err != nil {
			addAPIError(diags, "Unable to update Daytona sandbox auto-delete interval", "set sandbox auto-delete interval", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	if !plan.DesiredState.Equal(state.DesiredState) && !plan.DesiredState.IsNull() && plan.DesiredState.ValueString() != "" {
		sandbox, response, err := applySandboxDesiredState(ctx, a.client, state.ID.ValueString(), plan.DesiredState.ValueString())
		if err != nil {
			addAPIError(diags, "Unable to set Daytona sandbox state", "set sandbox state", response, err)
			return plan, false
		}
		plan = flattenSandbox(ctx, sandbox, plan)
	}

	current, response, err := a.client.api.SandboxAPI.GetSandbox(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		addAPIError(diags, "Unable to read Daytona sandbox", "read sandbox", response, err)
		return plan, false
	}

	return flattenSandbox(ctx, current, plan), true
}

func sandboxIntervalUpdateValue(plan, state types.Int64, clearValue int32) (float32, bool) {
	if plan.Equal(state) || plan.IsUnknown() {
		return 0, false
	}
	if value := optionalInt32(plan); value != nil {
		return float32(*value), true
	}
	return float32(clearValue), true
}

func applySandboxDesiredState(ctx context.Context, client *daytonaClient, id string, desiredState string) (*apiclient.Sandbox, *http.Response, error) {
	switch desiredState {
	case "started":
		return client.api.SandboxAPI.StartSandbox(ctx, id).Execute()
	case "stopped":
		return client.api.SandboxAPI.StopSandbox(ctx, id).Execute()
	case "archived":
		return client.api.SandboxAPI.ArchiveSandbox(ctx, id).Execute()
	case "":
		return client.api.SandboxAPI.GetSandbox(ctx, id).Execute()
	default:
		return nil, nil, fmt.Errorf("unsupported desired_state %q", desiredState)
	}
}
