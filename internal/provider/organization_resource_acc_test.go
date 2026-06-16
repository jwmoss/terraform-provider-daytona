package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationResource_basic creates a real Daytona organization, reads it,
// imports it, and destroys it. Organization management is rejected for API-key auth
// and requires the org API (JWT).
//
// Only the create/read/delete lifecycle and default_region_id are exercised here.
// The quota, sandbox-egress, experimental-config, and OTel attributes are
// platform-admin only (the managed cloud returns HTTP 401 for an org owner) and are
// covered separately behind DAYTONA_ACC_ORG_ADMIN.
func TestAccOrganizationResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)

	name := "tf-acc-org-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_organization.test", "id"),
					resource.TestCheckResourceAttr("daytona_organization.test", "name", name),
					resource.TestCheckResourceAttr("daytona_organization.test", "default_region_id", testAccDefaultRegionID()),
				),
			},
			{
				ResourceName:      "daytona_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccOrganizationResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = %[1]q
  default_region_id = %[2]q
}
`, name, testAccDefaultRegionID())
}
