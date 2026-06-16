package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationMemberAccessResource_basic updates the role/assigned roles of
// an existing organization member, then removes them on destroy. It cannot
// provision a second user, so it operates on a pre-existing member supplied via
// environment variables and is skipped otherwise. Requires the org API (JWT); see
// issue #3.
func TestAccOrganizationMemberAccessResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)

	orgID := os.Getenv("DAYTONA_ACC_MEMBER_ORG_ID")
	userID := os.Getenv("DAYTONA_ACC_MEMBER_USER_ID")
	if orgID == "" || userID == "" {
		t.Skip("set DAYTONA_ACC_MEMBER_ORG_ID and DAYTONA_ACC_MEMBER_USER_ID (an existing member) to run the member access acceptance test")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationMemberAccessResourceConfig(orgID, userID, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_member_access.test", "organization_id", orgID),
					resource.TestCheckResourceAttr("daytona_organization_member_access.test", "user_id", userID),
					resource.TestCheckResourceAttr("daytona_organization_member_access.test", "role", "member"),
				),
			},
			{
				Config: testAccOrganizationMemberAccessResourceConfig(orgID, userID, "owner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_member_access.test", "role", "owner"),
				),
			},
			{
				ResourceName:      "daytona_organization_member_access.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCompositeImportID("daytona_organization_member_access.test", "organization_id", "user_id"),
			},
		},
	})
}

func testAccOrganizationMemberAccessResourceConfig(orgID, userID, role string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization_member_access" "test" {
  organization_id   = %[1]q
  user_id           = %[2]q
  role              = %[3]q
  assigned_role_ids = []
}
`, orgID, userID, role)
}
