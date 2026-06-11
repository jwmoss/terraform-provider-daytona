// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestOrganizationInvitationActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]func() action.Action{
		"daytona_accept_organization_invitation":  NewOrganizationInvitationAcceptAction,
		"daytona_decline_organization_invitation": NewOrganizationInvitationDeclineAction,
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

			invitationAttr, ok := schemaResp.Schema.Attributes["invitation_id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected invitation_id to be a string attribute, got %T", schemaResp.Schema.Attributes["invitation_id"])
			}
			if !invitationAttr.Required {
				t.Fatal("expected invitation_id to be required")
			}
		})
	}
}

func TestOrganizationLifecycleActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory          func() action.Action
		expectSuspension bool
	}{
		"daytona_leave_organization": {
			factory: NewOrganizationLeaveAction,
		},
		"daytona_suspend_organization": {
			factory:          NewOrganizationSuspendAction,
			expectSuspension: true,
		},
		"daytona_unsuspend_organization": {
			factory: NewOrganizationUnsuspendAction,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := testCase.factory()

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

			organizationAttr, ok := schemaResp.Schema.Attributes["organization_id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected organization_id to be a string attribute, got %T", schemaResp.Schema.Attributes["organization_id"])
			}
			if !organizationAttr.Required {
				t.Fatal("expected organization_id to be required")
			}

			_, hasReason := schemaResp.Schema.Attributes["reason"].(actionschema.StringAttribute)
			_, hasUntil := schemaResp.Schema.Attributes["until"].(actionschema.StringAttribute)
			_, hasGracePeriod := schemaResp.Schema.Attributes["suspension_cleanup_grace_period_hours"].(actionschema.Float64Attribute)
			if hasReason != testCase.expectSuspension || hasUntil != testCase.expectSuspension || hasGracePeriod != testCase.expectSuspension {
				t.Fatalf("expected suspension attributes present=%t, got reason=%t until=%t grace=%t", testCase.expectSuspension, hasReason, hasUntil, hasGracePeriod)
			}
		})
	}
}

func TestOrganizationInvitationAcceptActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"invitation-1","organizationId":"org-1","organizationName":"Automation","email":"user@example.com","role":"member","assignedRoles":[],"status":"accepted","invitedBy":"owner@example.com","createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z","expiresAt":"2026-06-18T00:00:00Z"}`))
	}))
	defer server.Close()

	actionInstance := NewOrganizationInvitationAcceptAction()
	configureActionClient(t, actionInstance, server.URL)

	config := organizationInvitationActionConfig(t, actionInstance, "invitation-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/organizations/invitations/invitation-1/accept" {
		t.Fatalf("expected path %q, got %q", "/organizations/invitations/invitation-1/accept", gotPath)
	}
}

func TestOrganizationInvitationDeclineActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewOrganizationInvitationDeclineAction()
	configureActionClient(t, actionInstance, server.URL)

	config := organizationInvitationActionConfig(t, actionInstance, "invitation-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/organizations/invitations/invitation-1/decline" {
		t.Fatalf("expected path %q, got %q", "/organizations/invitations/invitation-1/decline", gotPath)
	}
}

func TestOrganizationLeaveActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	actionInstance := NewOrganizationLeaveAction()
	configureActionClient(t, actionInstance, server.URL)

	config := organizationActionConfig(t, actionInstance, "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/organizations/org-1/leave" {
		t.Fatalf("expected path %q, got %q", "/organizations/org-1/leave", gotPath)
	}
}

func TestOrganizationSuspendActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("unable to decode request body: %s", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	actionInstance := NewOrganizationSuspendAction()
	configureActionClient(t, actionInstance, server.URL)

	config := organizationSuspendActionConfig(t, actionInstance, "org-1", "billing", "2026-12-31T23:59:59Z", 24)

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/organizations/org-1/suspend" {
		t.Fatalf("expected path %q, got %q", "/organizations/org-1/suspend", gotPath)
	}
	if gotBody["reason"] != "billing" {
		t.Fatalf("expected reason %q, got %#v", "billing", gotBody["reason"])
	}
	if gotBody["until"] != "2026-12-31T23:59:59Z" {
		t.Fatalf("expected until timestamp, got %#v", gotBody["until"])
	}
	if gotBody["suspensionCleanupGracePeriodHours"] != float64(24) {
		t.Fatalf("expected grace period 24, got %#v", gotBody["suspensionCleanupGracePeriodHours"])
	}
}

func TestOrganizationUnsuspendActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	actionInstance := NewOrganizationUnsuspendAction()
	configureActionClient(t, actionInstance, server.URL)

	config := organizationActionConfig(t, actionInstance, "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/organizations/org-1/unsuspend" {
		t.Fatalf("expected path %q, got %q", "/organizations/org-1/unsuspend", gotPath)
	}
}

func TestOrganizationSuspendActionRejectsInvalidUntil(t *testing.T) {
	t.Parallel()

	actionInstance := NewOrganizationSuspendAction()
	configureActionClient(t, actionInstance, "https://daytona.invalid")

	config := organizationSuspendActionConfig(t, actionInstance, "org-1", "billing", "not-a-time", 24)

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if !invokeResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid until timestamp")
	}
}

func organizationInvitationActionConfig(t *testing.T, actionInstance action.Action, invitationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"invitation_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"invitation_id": terraformValue(t, types.StringValue(invitationID)),
	})
}

func organizationActionConfig(t *testing.T, actionInstance action.Action, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"organization_id": terraformValue(t, types.StringValue(organizationID)),
	})
}

func organizationSuspendActionConfig(t *testing.T, actionInstance action.Action, organizationID, reason, until string, gracePeriodHours float64) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"organization_id":                       tftypes.String,
		"reason":                                tftypes.String,
		"until":                                 tftypes.String,
		"suspension_cleanup_grace_period_hours": tftypes.Number,
	}}, map[string]tftypes.Value{
		"organization_id":                       terraformValue(t, types.StringValue(organizationID)),
		"reason":                                terraformValue(t, types.StringValue(reason)),
		"until":                                 terraformValue(t, types.StringValue(until)),
		"suspension_cleanup_grace_period_hours": terraformValue(t, types.Float64Value(gracePeriodHours)),
	})
}
