package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccOrganizationOtelConfigResource_basic creates a throwaway organization,
// configures its OpenTelemetry endpoint, updates the endpoint, imports the config,
// and destroys everything. Requires the org API (JWT).
func TestAccOrganizationOtelConfigResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)
	if os.Getenv("DAYTONA_ACC_ORG_ADMIN") == "" {
		t.Skip("set DAYTONA_ACC_ORG_ADMIN=1 to run the OTel config acceptance test (the managed cloud returns HTTP 401 for an org owner; needs platform-admin or self-hosted Daytona)")
	}

	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationOtelConfigResourceConfig(suffix, "https://otel.example.com:4318"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_organization_otel_config.test", "organization_id"),
					resource.TestCheckResourceAttr("daytona_organization_otel_config.test", "endpoint", "https://otel.example.com:4318"),
				),
			},
			{
				Config: testAccOrganizationOtelConfigResourceConfig(suffix, "https://otel2.example.com:4318"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_otel_config.test", "endpoint", "https://otel2.example.com:4318"),
				),
			},
			{
				ResourceName:            "daytona_organization_otel_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       testAccSingleAttrImportID("daytona_organization_otel_config.test", "organization_id"),
				ImportStateVerifyIgnore: []string{"headers"},
			},
		},
	})
}

func testAccSingleAttrImportID(resourceName, attr string) func(*terraform.State) (string, error) {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes[attr], nil
	}
}

func testAccOrganizationOtelConfigResourceConfig(suffix, endpoint string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = "tf-acc-otel-org-%[1]s"
  default_region_id = %[2]q
}

resource "daytona_organization_otel_config" "test" {
  organization_id = daytona_organization.test.id
  endpoint        = %[3]q
  headers = {
    "x-tf-acc" = "1"
  }
}
`, suffix, testAccDefaultRegionID(), endpoint)
}
