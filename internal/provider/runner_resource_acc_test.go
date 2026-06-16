package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRunnerResource_basic creates a region, registers a custom runner in it,
// toggles the schedulable flag, verifies import, and destroys both.
//
// The runner create/update/delete endpoints are not served by the managed Daytona
// cloud (app.daytona.io returns HTTP 404). This test is therefore self-hosted only
// and opt-in via DAYTONA_ACC_SELF_HOSTED so it is skipped on the cloud CI run.
func TestAccRunnerResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_SELF_HOSTED") == "" {
		t.Skip("set DAYTONA_ACC_SELF_HOSTED=1 to run the runner acceptance test (self-hosted Daytona only; cloud returns 404)")
	}

	name := "tf-acc-runner-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRunnerResourceConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_runner.test", "id"),
					resource.TestCheckResourceAttr("daytona_runner.test", "name", name),
					resource.TestCheckResourceAttrSet("daytona_runner.test", "region_id"),
					resource.TestCheckResourceAttr("daytona_runner.test", "unschedulable", "false"),
				),
			},
			{
				Config: testAccRunnerResourceConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_runner.test", "unschedulable", "true"),
				),
			},
			{
				ResourceName:            "daytona_runner.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key", "draining"},
			},
		},
	})
}

func testAccRunnerResourceConfig(name string, unschedulable bool) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_region" "test" {
  name = "%[1]s-region"
}

resource "daytona_runner" "test" {
  region_id     = daytona_region.test.id
  name          = %[1]q
  unschedulable = %[2]t
}
`, name, unschedulable)
}
