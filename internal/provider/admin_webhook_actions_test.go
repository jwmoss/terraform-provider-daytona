// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminWebhookActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory      func() action.Action
		hasPayload   bool
		requiredAttr string
	}{
		"daytona_admin_initialize_webhooks": {
			factory: NewAdminInitializeWebhooksAction,
		},
		"daytona_admin_send_webhook": {
			factory:      NewAdminSendWebhookAction,
			hasPayload:   true,
			requiredAttr: "event_type",
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
			if !organizationAttr.Optional {
				t.Fatal("expected organization_id to be optional")
			}

			if testCase.hasPayload {
				eventTypeAttr, ok := schemaResp.Schema.Attributes[testCase.requiredAttr].(actionschema.StringAttribute)
				if !ok {
					t.Fatalf("expected %s to be a string attribute, got %T", testCase.requiredAttr, schemaResp.Schema.Attributes[testCase.requiredAttr])
				}
				if !eventTypeAttr.Required {
					t.Fatalf("expected %s to be required", testCase.requiredAttr)
				}

				payloadAttr, ok := schemaResp.Schema.Attributes["payload_json"].(actionschema.StringAttribute)
				if !ok {
					t.Fatalf("expected payload_json to be a string attribute, got %T", schemaResp.Schema.Attributes["payload_json"])
				}
				if !payloadAttr.WriteOnly {
					t.Fatal("expected payload_json to be write-only")
				}
			}
		})
	}
}

func TestAdminInitializeWebhooksActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	actionInstance := NewAdminInitializeWebhooksAction()
	configureActionClientWithOrganization(t, actionInstance, server.URL, "org-1")

	config := adminInitializeWebhooksActionConfig(t, actionInstance, "")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/webhooks/organizations/org-1/initialize" {
		t.Fatalf("expected path %q, got %q", "/admin/webhooks/organizations/org-1/initialize", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
}

func TestAdminSendWebhookActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %s", err)
		}
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewAdminSendWebhookAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminSendWebhookActionConfig(t, actionInstance, "org-1", "sandbox.created", `{"sandboxId":"sandbox-1"}`, "event-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/webhooks/organizations/org-1/send" {
		t.Fatalf("expected path %q, got %q", "/admin/webhooks/organizations/org-1/send", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if gotPayload["eventType"] != "sandbox.created" {
		t.Fatalf("expected eventType %q, got %#v", "sandbox.created", gotPayload["eventType"])
	}
	if gotPayload["eventId"] != "event-1" {
		t.Fatalf("expected eventId %q, got %#v", "event-1", gotPayload["eventId"])
	}
	payload, ok := gotPayload["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %#v", gotPayload["payload"])
	}
	if payload["sandboxId"] != "sandbox-1" {
		t.Fatalf("expected sandboxId %q, got %#v", "sandbox-1", payload["sandboxId"])
	}
}

func adminInitializeWebhooksActionConfig(t *testing.T, actionInstance action.Action, organizationID string) *tfsdk.Config {
	t.Helper()

	var organizationValue tftypes.Value
	if organizationID == "" {
		organizationValue = terraformValue(t, types.StringNull())
	} else {
		organizationValue = terraformValue(t, types.StringValue(organizationID))
	}

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"organization_id": organizationValue,
	})
}

func adminSendWebhookActionConfig(t *testing.T, actionInstance action.Action, organizationID, eventType, payloadJSON, eventID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"organization_id": tftypes.String,
		"event_type":      tftypes.String,
		"payload_json":    tftypes.String,
		"event_id":        tftypes.String,
	}}, map[string]tftypes.Value{
		"organization_id": terraformValue(t, types.StringValue(organizationID)),
		"event_type":      terraformValue(t, types.StringValue(eventType)),
		"payload_json":    terraformValue(t, types.StringValue(payloadJSON)),
		"event_id":        terraformValue(t, types.StringValue(eventID)),
	})
}
