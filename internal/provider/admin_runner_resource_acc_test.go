package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAdminRunnerResource_basic registers a runner through the Daytona admin
// API, toggles the schedulable flag, and destroys it.
//
// The admin runner endpoints are not served by the managed Daytona cloud
// (app.daytona.io returns HTTP 404) and require admin access on a self-hosted
// Daytona deployment. The test is opt-in via DAYTONA_ACC_SELF_HOSTED_ADMIN so it is
// skipped on the cloud CI run and on non-admin credentials.
func TestAccAdminRunnerResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_SELF_HOSTED_ADMIN") == "" {
		t.Skip("set DAYTONA_ACC_SELF_HOSTED_ADMIN=1 to run the admin runner acceptance test (self-hosted Daytona with admin access only)")
	}

	regionID := os.Getenv("DAYTONA_ACC_RUNNER_REGION_ID")
	if regionID == "" {
		t.Skip("set DAYTONA_ACC_RUNNER_REGION_ID to the region the admin runner should be registered in")
	}

	name := "tf-acc-admin-runner-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAdminRunnerResourceConfig(name, regionID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_admin_runner.test", "id"),
					resource.TestCheckResourceAttr("daytona_admin_runner.test", "name", name),
					resource.TestCheckResourceAttr("daytona_admin_runner.test", "region_id", regionID),
					resource.TestCheckResourceAttr("daytona_admin_runner.test", "unschedulable", "false"),
				),
			},
			{
				Config: testAccAdminRunnerResourceConfig(name, regionID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_admin_runner.test", "unschedulable", "true"),
				),
			},
		},
	})
}

func testAccAdminRunnerResourceConfig(name, regionID string, unschedulable bool) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_admin_runner" "test" {
  region_id     = %[2]q
  name          = %[1]q
  api_key       = "tf-acc-admin-runner-placeholder-key"
  api_version   = "2"
  unschedulable = %[3]t
}
`, name, regionID, unschedulable)
}
