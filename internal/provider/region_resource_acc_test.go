package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRegionResource_basic creates a real Daytona region, updates a mutable
// URL, verifies import, and destroys it.
//
// The region create/update/delete endpoints are not served by the managed Daytona
// cloud (app.daytona.io returns HTTP 404). This test is therefore self-hosted only
// and opt-in via DAYTONA_ACC_SELF_HOSTED so it is skipped on the cloud CI run.
func TestAccRegionResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_SELF_HOSTED") == "" {
		t.Skip("set DAYTONA_ACC_SELF_HOSTED=1 to run the region acceptance test (self-hosted Daytona only; cloud returns 404)")
	}

	name := "tf-acc-region-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegionResourceConfig(name, "https://proxy.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_region.test", "id"),
					resource.TestCheckResourceAttr("daytona_region.test", "name", name),
					resource.TestCheckResourceAttr("daytona_region.test", "proxy_url", "https://proxy.example.com"),
				),
			},
			{
				Config: testAccRegionResourceConfig(name, "https://proxy2.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_region.test", "proxy_url", "https://proxy2.example.com"),
				),
			},
			{
				ResourceName:            "daytona_region.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"proxy_api_key", "ssh_gateway_api_key", "snapshot_manager_username", "snapshot_manager_password"},
			},
		},
	})
}

func testAccRegionResourceConfig(name, proxyURL string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_region" "test" {
  name      = %[1]q
  proxy_url = %[2]q
}
`, name, proxyURL)
}
