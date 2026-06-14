package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"daytona": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheckAPIKey(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	if os.Getenv("DAYTONA_API_KEY") == "" {
		t.Fatal("DAYTONA_API_KEY must be set for acceptance tests")
	}

	t.Setenv("DAYTONA_ACCESS_TOKEN", "")
}

func testAccPreCheckAccessToken(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	if os.Getenv("DAYTONA_ACCESS_TOKEN") == "" {
		t.Skip("DAYTONA_ACCESS_TOKEN must be set for JWT-only Daytona API acceptance tests")
	}
	if os.Getenv("DAYTONA_ORGANIZATION_ID") == "" {
		t.Skip("DAYTONA_ORGANIZATION_ID must be set for JWT-only organization-scoped Daytona API acceptance tests")
	}

	t.Setenv("DAYTONA_API_KEY", "")
}

func testAccPreCheckHealthCheckAPIKey(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	healthCheckKey := os.Getenv("DAYTONA_HEALTH_CHECK_API_KEY")
	if healthCheckKey == "" {
		t.Skip("DAYTONA_HEALTH_CHECK_API_KEY must be set for Daytona readiness acceptance tests")
	}

	t.Setenv("DAYTONA_API_KEY", healthCheckKey)
	t.Setenv("DAYTONA_ACCESS_TOKEN", "")
}

func TestDaytonaProviderSchemaIncludesAccessToken(t *testing.T) {
	t.Parallel()

	providerInstance := New("test")()

	var schemaResp provider.SchemaResponse
	providerInstance.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	accessTokenAttr, ok := schemaResp.Schema.Attributes["access_token"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected access_token to be a string attribute, got %T", schemaResp.Schema.Attributes["access_token"])
	}
	if !accessTokenAttr.Optional {
		t.Fatal("expected access_token to be optional")
	}
	if !accessTokenAttr.Sensitive {
		t.Fatal("expected access_token to be sensitive")
	}
}
