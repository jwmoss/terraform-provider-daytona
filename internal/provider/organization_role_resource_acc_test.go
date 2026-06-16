package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationRoleResource_basic creates a throwaway organization, defines a
// custom role in it, updates the role's name/description/permissions, imports it,
// and destroys everything. Requires the org API (JWT); see issue #3.
func TestAccOrganizationRoleResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)

	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationRoleResourceConfig(suffix, "tf-acc-role", "initial description", `["write:sandboxes"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_organization_role.test", "id"),
					resource.TestCheckResourceAttr("daytona_organization_role.test", "name", "tf-acc-role"),
					resource.TestCheckResourceAttr("daytona_organization_role.test", "description", "initial description"),
					resource.TestCheckResourceAttr("daytona_organization_role.test", "permissions.#", "1"),
				),
			},
			{
				Config: testAccOrganizationRoleResourceConfig(suffix, "tf-acc-role-renamed", "updated description", `["write:sandboxes", "delete:sandboxes"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_role.test", "name", "tf-acc-role-renamed"),
					resource.TestCheckResourceAttr("daytona_organization_role.test", "description", "updated description"),
					resource.TestCheckResourceAttr("daytona_organization_role.test", "permissions.#", "2"),
				),
			},
			{
				ResourceName:      "daytona_organization_role.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCompositeImportID("daytona_organization_role.test", "organization_id", "id"),
			},
		},
	})
}

func testAccOrganizationRoleResourceConfig(suffix, name, description, permissions string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = "tf-acc-role-org-%[1]s"
  default_region_id = %[2]q
}

resource "daytona_organization_role" "test" {
  organization_id = daytona_organization.test.id
  name            = %[3]q
  description     = %[4]q
  permissions     = %[5]s
}
`, suffix, testAccDefaultRegionID(), name, description, permissions)
}
