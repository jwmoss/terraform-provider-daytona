// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
)

func TestDaytonaProviderActions(t *testing.T) {
	t.Parallel()

	actions := (&DaytonaProvider{}).Actions(context.Background())
	if got, want := len(actions), 25; got != want {
		t.Fatalf("expected %d actions, got %d", want, got)
	}

	actionNames := make(map[string]bool, len(actions))
	for _, factory := range actions {
		actionInstance := factory()

		var metadataResp action.MetadataResponse
		actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
		actionNames[metadataResp.TypeName] = true
	}

	for _, name := range []string{
		"daytona_activate_snapshot",
		"daytona_accept_organization_invitation",
		"daytona_admin_set_default_docker_registry",
		"daytona_admin_set_snapshot_general_status",
		"daytona_archive_sandbox",
		"daytona_create_sandbox_backup",
		"daytona_create_sandbox_snapshot",
		"daytona_deactivate_snapshot",
		"daytona_decline_organization_invitation",
		"daytona_enroll_sms_mfa",
		"daytona_expire_sandbox_signed_port_preview_url",
		"daytona_fork_sandbox",
		"daytona_initialize_webhooks",
		"daytona_leave_organization",
		"daytona_link_account",
		"daytona_recover_sandbox",
		"daytona_refresh_webhook_endpoints",
		"daytona_revoke_api_key_for_user",
		"daytona_revoke_sandbox_ssh_access",
		"daytona_start_sandbox",
		"daytona_stop_sandbox",
		"daytona_suspend_organization",
		"daytona_unlink_account",
		"daytona_unsuspend_organization",
		"daytona_update_sandbox_last_activity",
	} {
		if !actionNames[name] {
			t.Fatalf("expected action %q to be registered, got %#v", name, actionNames)
		}
	}
}

func TestSnapshotActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]func() action.Action{
		"daytona_activate_snapshot":   NewSnapshotActivateAction,
		"daytona_deactivate_snapshot": NewSnapshotDeactivateAction,
	}

	for name, factory := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := factory()

			var metadataResp action.MetadataResponse
			actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != name {
				t.Fatalf("expected type name %q, got %q", name, metadataResp.TypeName)
			}

			var schemaResp action.SchemaResponse
			actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			idAttr, ok := schemaResp.Schema.Attributes["id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected id to be a string attribute, got %T", schemaResp.Schema.Attributes["id"])
			}
			if !idAttr.Required {
				t.Fatal("expected id to be required")
			}

			organizationAttr, ok := schemaResp.Schema.Attributes["organization_id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected organization_id to be a string attribute, got %T", schemaResp.Schema.Attributes["organization_id"])
			}
			if !organizationAttr.Optional {
				t.Fatal("expected organization_id to be optional")
			}
		})
	}
}

func TestSnapshotActionConfigureRejectsUnexpectedType(t *testing.T) {
	t.Parallel()

	actionInstance := NewSnapshotActivateAction()
	configurable, ok := actionInstance.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("expected snapshot action to implement ActionWithConfigure")
	}

	var configureResp action.ConfigureResponse
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: "unexpected"}, &configureResp)

	if !configureResp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostics for unexpected provider data")
	}
}
