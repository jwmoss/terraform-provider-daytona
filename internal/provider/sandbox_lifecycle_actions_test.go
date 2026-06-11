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

func TestSandboxLifecycleActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory             func() action.Action
		expectedTypeName    string
		expectSandboxID     bool
		expectName          bool
		nameRequired        bool
		expectIncludeMemory bool
		expectSkipStart     bool
	}{
		"recover": {
			factory:          NewSandboxRecoverAction,
			expectedTypeName: "daytona_recover_sandbox",
			expectSkipStart:  true,
		},
		"backup": {
			factory:          NewSandboxCreateBackupAction,
			expectedTypeName: "daytona_create_sandbox_backup",
		},
		"snapshot": {
			factory:             NewSandboxCreateSnapshotAction,
			expectedTypeName:    "daytona_create_sandbox_snapshot",
			expectName:          true,
			nameRequired:        true,
			expectIncludeMemory: true,
		},
		"fork": {
			factory:          NewSandboxForkAction,
			expectedTypeName: "daytona_fork_sandbox",
			expectName:       true,
		},
		"last activity": {
			factory:          NewSandboxUpdateLastActivityAction,
			expectedTypeName: "daytona_update_sandbox_last_activity",
			expectSandboxID:  true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := testCase.factory()

			var metadataResp action.MetadataResponse
			actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != testCase.expectedTypeName {
				t.Fatalf("expected type name %q, got %q", testCase.expectedTypeName, metadataResp.TypeName)
			}

			var schemaResp action.SchemaResponse
			actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			if testCase.expectSandboxID {
				sandboxIDAttr, ok := schemaResp.Schema.Attributes["sandbox_id"].(actionschema.StringAttribute)
				if !ok {
					t.Fatalf("expected sandbox_id to be a string attribute, got %T", schemaResp.Schema.Attributes["sandbox_id"])
				}
				if !sandboxIDAttr.Required {
					t.Fatal("expected sandbox_id to be required")
				}
			} else {
				sandboxAttr, ok := schemaResp.Schema.Attributes["sandbox_id_or_name"].(actionschema.StringAttribute)
				if !ok {
					t.Fatalf("expected sandbox_id_or_name to be a string attribute, got %T", schemaResp.Schema.Attributes["sandbox_id_or_name"])
				}
				if !sandboxAttr.Required {
					t.Fatal("expected sandbox_id_or_name to be required")
				}
			}

			if testCase.expectName {
				nameAttr, ok := schemaResp.Schema.Attributes["name"].(actionschema.StringAttribute)
				if !ok {
					t.Fatalf("expected name to be a string attribute, got %T", schemaResp.Schema.Attributes["name"])
				}
				if nameAttr.Required != testCase.nameRequired {
					t.Fatalf("expected name required=%t, got %t", testCase.nameRequired, nameAttr.Required)
				}
				if nameAttr.Optional == testCase.nameRequired {
					t.Fatalf("expected name optional=%t, got %t", !testCase.nameRequired, nameAttr.Optional)
				}
			}

			if testCase.expectIncludeMemory {
				includeMemoryAttr, ok := schemaResp.Schema.Attributes["include_memory"].(actionschema.BoolAttribute)
				if !ok {
					t.Fatalf("expected include_memory to be a bool attribute, got %T", schemaResp.Schema.Attributes["include_memory"])
				}
				if !includeMemoryAttr.Optional {
					t.Fatal("expected include_memory to be optional")
				}
			}

			if testCase.expectSkipStart {
				skipStartAttr, ok := schemaResp.Schema.Attributes["skip_start"].(actionschema.BoolAttribute)
				if !ok {
					t.Fatalf("expected skip_start to be a bool attribute, got %T", schemaResp.Schema.Attributes["skip_start"])
				}
				if !skipStartAttr.Optional {
					t.Fatal("expected skip_start to be optional")
				}
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

func TestSandboxRecoverActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotSkipStart, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotSkipStart = r.URL.Query().Get("skipStart")
		gotOrganizationID = r.Header.Get(organizationHeader)
		writeSandboxActionResponse(t, w)
	}))
	defer server.Close()

	actionInstance := NewSandboxRecoverAction()
	configureActionClient(t, actionInstance, server.URL)

	config := recoverSandboxActionConfig(t, actionInstance, "sandbox-1", true, "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox/sandbox-1/recover" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/recover", gotPath)
	}
	if gotSkipStart != "true" {
		t.Fatalf("expected skipStart query value %q, got %q", "true", gotSkipStart)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxCreateBackupActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		writeSandboxActionResponse(t, w)
	}))
	defer server.Close()

	actionInstance := NewSandboxCreateBackupAction()
	configureActionClient(t, actionInstance, server.URL)

	config := simpleSandboxActionConfig(t, actionInstance, "sandbox-1", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox/sandbox-1/backup" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/backup", gotPath)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxCreateSnapshotActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("unable to decode request body: %s", err)
		}
		writeSandboxActionResponse(t, w)
	}))
	defer server.Close()

	actionInstance := NewSandboxCreateSnapshotAction()
	configureActionClient(t, actionInstance, server.URL)

	config := createSandboxSnapshotActionConfig(t, actionInstance, "sandbox-1", "snapshot-1", true, "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox/sandbox-1/snapshot" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/snapshot", gotPath)
	}
	if gotBody["name"] != "snapshot-1" {
		t.Fatalf("expected snapshot name %q, got %#v", "snapshot-1", gotBody["name"])
	}
	if gotBody["includeMemory"] != true {
		t.Fatalf("expected includeMemory true, got %#v", gotBody["includeMemory"])
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxForkActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("unable to decode request body: %s", err)
		}
		writeSandboxActionResponse(t, w)
	}))
	defer server.Close()

	actionInstance := NewSandboxForkAction()
	configureActionClient(t, actionInstance, server.URL)

	config := forkSandboxActionConfig(t, actionInstance, "sandbox-1", "sandbox-fork", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox/sandbox-1/fork" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/fork", gotPath)
	}
	if gotBody["name"] != "sandbox-fork" {
		t.Fatalf("expected fork name %q, got %#v", "sandbox-fork", gotBody["name"])
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxUpdateLastActivityActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	actionInstance := NewSandboxUpdateLastActivityAction()
	configureActionClient(t, actionInstance, server.URL)

	config := updateLastActivityActionConfig(t, actionInstance, "sandbox-1", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox/sandbox-1/last-activity" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/last-activity", gotPath)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxLifecycleActionConfigureRejectsUnexpectedType(t *testing.T) {
	t.Parallel()

	actionInstance := NewSandboxRecoverAction()
	configurable, ok := actionInstance.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("expected sandbox lifecycle action to implement ActionWithConfigure")
	}

	var configureResp action.ConfigureResponse
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: "unexpected"}, &configureResp)

	if !configureResp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostics for unexpected provider data")
	}
}

func simpleSandboxActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func recoverSandboxActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName string, skipStart bool, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"skip_start":         tftypes.Bool,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"skip_start":         terraformValue(t, types.BoolValue(skipStart)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func createSandboxSnapshotActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName, name string, includeMemory bool, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"name":               tftypes.String,
		"include_memory":     tftypes.Bool,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"name":               terraformValue(t, types.StringValue(name)),
		"include_memory":     terraformValue(t, types.BoolValue(includeMemory)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func forkSandboxActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName, name, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"name":               tftypes.String,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"name":               terraformValue(t, types.StringValue(name)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func updateLastActivityActionConfig(t *testing.T, actionInstance action.Action, sandboxID, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id":      tftypes.String,
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id":      terraformValue(t, types.StringValue(sandboxID)),
		"organization_id": terraformValue(t, types.StringValue(organizationID)),
	})
}

func writeSandboxActionResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"id":"sandbox-1","organizationId":"org-1","name":"sandbox-1","user":"user-1","env":{},"labels":{},"public":false,"networkBlockAll":false,"target":"region-1","cpu":1,"gpu":0,"memory":2,"disk":10,"toolboxProxyUrl":"https://toolbox.example.com"}`))
}
