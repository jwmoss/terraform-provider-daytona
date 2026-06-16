package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationInvitationResource_basic creates a throwaway organization,
// invites a member by email, updates the invitation role, imports it, and destroys
// everything. Creating an invitation sends a real email, so the test is opt-in via
// DAYTONA_ACC_INVITE. Requires the org API (JWT).
func TestAccOrganizationInvitationResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)
	if os.Getenv("DAYTONA_ACC_INVITE") == "" {
		t.Skip("set DAYTONA_ACC_INVITE=1 to run the organization invitation acceptance test (sends a real invitation email)")
	}

	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	email := fmt.Sprintf("tf-acc-invitee-%s@example.com", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationInvitationResourceConfig(suffix, email, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_organization_invitation.test", "id"),
					resource.TestCheckResourceAttr("daytona_organization_invitation.test", "email", email),
					resource.TestCheckResourceAttr("daytona_organization_invitation.test", "role", "member"),
				),
			},
			{
				Config: testAccOrganizationInvitationResourceConfig(suffix, email, "owner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_invitation.test", "role", "owner"),
				),
			},
			{
				ResourceName:      "daytona_organization_invitation.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCompositeImportID("daytona_organization_invitation.test", "organization_id", "id"),
			},
		},
	})
}

func testAccOrganizationInvitationResourceConfig(suffix, email, role string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = "tf-acc-invite-org-%[1]s"
  default_region_id = %[2]q
}

resource "daytona_organization_invitation" "test" {
  organization_id   = daytona_organization.test.id
  email             = %[3]q
  role              = %[4]q
  assigned_role_ids = []
}
`, suffix, testAccDefaultRegionID(), email, role)
}
