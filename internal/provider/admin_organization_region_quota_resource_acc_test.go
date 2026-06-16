package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAdminOrganizationRegionQuotaResource_basic creates a throwaway
// organization and manages its per-region quota through the Daytona admin API,
// updating quota values and importing the record.
//
// The admin quota endpoints require platform-admin credentials and are not usable
// by an org owner on the managed cloud (HTTP 401), so the test is opt-in via
// DAYTONA_ACC_ORG_ADMIN. Requires the org API (JWT) with admin access.
func TestAccAdminOrganizationRegionQuotaResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)
	if os.Getenv("DAYTONA_ACC_ORG_ADMIN") == "" {
		t.Skip("set DAYTONA_ACC_ORG_ADMIN=1 to run the admin region quota acceptance test (needs platform-admin or self-hosted Daytona)")
	}

	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	region := testAccDefaultRegionID()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAdminOrganizationRegionQuotaResourceConfig(suffix, region, 4, 8, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_admin_organization_region_quota.test", "region_id", region),
					resource.TestCheckResourceAttr("daytona_admin_organization_region_quota.test", "sandbox_class", "container"),
					resource.TestCheckResourceAttr("daytona_admin_organization_region_quota.test", "total_cpu_quota", "4"),
				),
			},
			{
				Config: testAccAdminOrganizationRegionQuotaResourceConfig(suffix, region, 8, 16, 40),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_admin_organization_region_quota.test", "total_cpu_quota", "8"),
				),
			},
			{
				ResourceName:      "daytona_admin_organization_region_quota.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRegionQuotaImportID("daytona_admin_organization_region_quota.test"),
			},
		},
	})
}

func testAccAdminOrganizationRegionQuotaResourceConfig(suffix, region string, cpu, mem, disk int) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = "tf-acc-admin-quota-org-%[1]s"
  default_region_id = %[2]q
}

resource "daytona_admin_organization_region_quota" "test" {
  organization_id    = daytona_organization.test.id
  region_id          = %[2]q
  sandbox_class      = "container"
  total_cpu_quota    = %[3]d
  total_memory_quota = %[4]d
  total_disk_quota   = %[5]d
  total_gpu_quota    = 0
}
`, suffix, region, cpu, mem, disk)
}
