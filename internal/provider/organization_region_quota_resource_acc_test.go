package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccOrganizationRegionQuotaResource_basic creates a throwaway organization,
// sets a per-region sandbox-class quota, updates the quota values, and imports it.
// The org quota endpoint does not support deletion (delete is a state-only no-op),
// so the org teardown removes the underlying record. Requires the org API (JWT).
func TestAccOrganizationRegionQuotaResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)
	if os.Getenv("DAYTONA_ACC_ORG_ADMIN") == "" {
		t.Skip("set DAYTONA_ACC_ORG_ADMIN=1 to run the region quota acceptance test (the managed cloud returns HTTP 401 for an org owner; needs platform-admin or self-hosted Daytona)")
	}

	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	region := testAccDefaultRegionID()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationRegionQuotaResourceConfig(suffix, region, 4, 8, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "region_id", region),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "sandbox_class", "container"),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_cpu_quota", "4"),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_memory_quota", "8"),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_disk_quota", "20"),
				),
			},
			{
				Config: testAccOrganizationRegionQuotaResourceConfig(suffix, region, 8, 16, 40),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_cpu_quota", "8"),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_memory_quota", "16"),
					resource.TestCheckResourceAttr("daytona_organization_region_quota.test", "total_disk_quota", "40"),
				),
			},
			{
				ResourceName:      "daytona_organization_region_quota.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRegionQuotaImportID("daytona_organization_region_quota.test"),
			},
		},
	})
}

func testAccRegionQuotaImportID(resourceName string) func(*terraform.State) (string, error) {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		a := rs.Primary.Attributes
		return fmt.Sprintf("%s:%s:%s", a["organization_id"], a["region_id"], a["sandbox_class"]), nil
	}
}

func testAccOrganizationRegionQuotaResourceConfig(suffix, region string, cpu, mem, disk int) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_organization" "test" {
  name              = "tf-acc-quota-org-%[1]s"
  default_region_id = %[2]q
}

resource "daytona_organization_region_quota" "test" {
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
