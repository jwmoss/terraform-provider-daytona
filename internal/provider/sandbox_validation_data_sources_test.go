package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSandboxValidationDataSourcesSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory       func() datasource.DataSource
		tokenAttr     string
		expectedBool  string
		expectedInput string
	}{
		"daytona_sandbox_public_status": {
			factory:       NewSandboxPublicStatusDataSource,
			expectedBool:  "public",
			expectedInput: "sandbox_id",
		},
		"daytona_sandbox_auth_token_validation": {
			factory:       NewSandboxAuthTokenValidationDataSource,
			tokenAttr:     "auth_token",
			expectedBool:  "valid",
			expectedInput: "sandbox_id",
		},
		"daytona_sandbox_access": {
			factory:       NewSandboxAccessDataSource,
			expectedBool:  "has_access",
			expectedInput: "sandbox_id",
		},
		"daytona_sandbox_id_from_signed_preview_token": {
			factory:   NewSandboxIDFromSignedPreviewTokenDataSource,
			tokenAttr: "signed_preview_token",
		},
		"daytona_sandbox_ssh_access_validation": {
			factory:      NewSandboxSSHAccessValidationDataSource,
			tokenAttr:    "token",
			expectedBool: "valid",
		},
	}

	for expectedTypeName, testCase := range testCases {
		t.Run(expectedTypeName, func(t *testing.T) {
			t.Parallel()

			dataSource := testCase.factory()

			var metadataResp datasource.MetadataResponse
			dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != expectedTypeName {
				t.Fatalf("expected type name %q, got %q", expectedTypeName, metadataResp.TypeName)
			}

			var schemaResp datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			if testCase.expectedInput != "" {
				inputAttr, ok := schemaResp.Schema.Attributes[testCase.expectedInput].(schema.StringAttribute)
				if !ok {
					t.Fatalf("expected %s to be a string attribute, got %T", testCase.expectedInput, schemaResp.Schema.Attributes[testCase.expectedInput])
				}
				if !inputAttr.Required {
					t.Fatalf("expected %s to be required", testCase.expectedInput)
				}
			}

			if testCase.tokenAttr != "" {
				tokenAttr, ok := schemaResp.Schema.Attributes[testCase.tokenAttr].(schema.StringAttribute)
				if !ok {
					t.Fatalf("expected %s to be a string attribute, got %T", testCase.tokenAttr, schemaResp.Schema.Attributes[testCase.tokenAttr])
				}
				if !tokenAttr.Required {
					t.Fatalf("expected %s to be required", testCase.tokenAttr)
				}
				if !tokenAttr.Sensitive {
					t.Fatalf("expected %s to be sensitive", testCase.tokenAttr)
				}
			}

			if testCase.expectedBool != "" {
				boolAttr, ok := schemaResp.Schema.Attributes[testCase.expectedBool].(schema.BoolAttribute)
				if !ok {
					t.Fatalf("expected %s to be a bool attribute, got %T", testCase.expectedBool, schemaResp.Schema.Attributes[testCase.expectedBool])
				}
				if !boolAttr.Computed {
					t.Fatalf("expected %s to be computed", testCase.expectedBool)
				}
			}
		})
	}
}

func TestSandboxPublicStatusDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`true`))
	}))
	defer server.Close()

	dataSource := NewSandboxPublicStatusDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := sandboxIDOnlyDataSourceConfig(t, dataSource, "sandbox-1")

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/preview/sandbox-1/public" {
		t.Fatalf("expected path %q, got %q", "/preview/sandbox-1/public", gotPath)
	}

	var data sandboxPublicStatusDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.Public.ValueBool() {
		t.Fatal("expected sandbox to be public")
	}
}

func TestSandboxAuthTokenValidationDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`true`))
	}))
	defer server.Close()

	dataSource := NewSandboxAuthTokenValidationDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"sandbox_id": tftypes.String,
		"auth_token": tftypes.String,
	}, map[string]tftypes.Value{
		"sandbox_id": tftypes.NewValue(tftypes.String, "sandbox-1"),
		"auth_token": tftypes.NewValue(tftypes.String, "auth-token"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/preview/sandbox-1/validate/auth-token" {
		t.Fatalf("expected path %q, got %q", "/preview/sandbox-1/validate/auth-token", gotPath)
	}

	var data sandboxAuthTokenValidationDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.Valid.ValueBool() {
		t.Fatal("expected sandbox auth token to be valid")
	}
}

func TestSandboxAccessDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`true`))
	}))
	defer server.Close()

	dataSource := NewSandboxAccessDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := sandboxIDOnlyDataSourceConfig(t, dataSource, "sandbox-1")

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/preview/sandbox-1/access" {
		t.Fatalf("expected path %q, got %q", "/preview/sandbox-1/access", gotPath)
	}

	var data sandboxAccessDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.HasAccess.ValueBool() {
		t.Fatal("expected user to have sandbox access")
	}
}

func TestSandboxIDFromSignedPreviewTokenDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"sandbox-1"`))
	}))
	defer server.Close()

	dataSource := NewSandboxIDFromSignedPreviewTokenDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"signed_preview_token": tftypes.String,
		"port":                 tftypes.Number,
	}, map[string]tftypes.Value{
		"signed_preview_token": tftypes.NewValue(tftypes.String, "signed-token"),
		"port":                 terraformValue(t, types.Int64Value(3000)),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/preview/signed-token/3000/sandbox-id" {
		t.Fatalf("expected path %q, got %q", "/preview/signed-token/3000/sandbox-id", gotPath)
	}

	var data sandboxIDFromSignedPreviewTokenDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.SandboxID.ValueString() != "sandbox-1" {
		t.Fatalf("expected sandbox ID %q, got %q", "sandbox-1", data.SandboxID.ValueString())
	}
}

func TestSandboxSSHAccessValidationDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotToken, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotToken = r.URL.Query().Get("token")
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"sandboxId":"sandbox-1"}`))
	}))
	defer server.Close()

	dataSource := NewSandboxSSHAccessValidationDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"token":                   tftypes.String,
		"request_organization_id": tftypes.String,
	}, map[string]tftypes.Value{
		"token":                   tftypes.NewValue(tftypes.String, "ssh-token"),
		"request_organization_id": tftypes.NewValue(tftypes.String, "org-1"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/sandbox/ssh-access/validate" {
		t.Fatalf("expected path %q, got %q", "/sandbox/ssh-access/validate", gotPath)
	}
	if gotToken != "ssh-token" {
		t.Fatalf("expected token query value %q, got %q", "ssh-token", gotToken)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}

	var data sandboxSSHAccessValidationDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.Valid.ValueBool() {
		t.Fatal("expected SSH access token to be valid")
	}
	if data.SandboxID.ValueString() != "sandbox-1" {
		t.Fatalf("expected sandbox ID %q, got %q", "sandbox-1", data.SandboxID.ValueString())
	}
}

func TestSandboxIDFromSignedPreviewTokenRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	dataSource := NewSandboxIDFromSignedPreviewTokenDataSource()
	configureDataSource(t, dataSource, "https://daytona.invalid")

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"signed_preview_token": tftypes.String,
		"port":                 tftypes.Number,
	}, map[string]tftypes.Value{
		"signed_preview_token": tftypes.NewValue(tftypes.String, "signed-token"),
		"port":                 terraformValue(t, types.Int64Value(0)),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid port")
	}
}

func sandboxIDOnlyDataSourceConfig(t *testing.T, dataSource datasource.DataSource, sandboxID string) *tfsdk.Config {
	t.Helper()

	return runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"sandbox_id": tftypes.String,
	}, map[string]tftypes.Value{
		"sandbox_id": tftypes.NewValue(tftypes.String, sandboxID),
	})
}
