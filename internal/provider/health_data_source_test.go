package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHealthDataSource_basic(t *testing.T) {
	testAccPreCheckHealthCheckAPIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "daytona" {}

data "daytona_health" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.daytona_health.test", "id", "health"),
					resource.TestCheckResourceAttr("data.daytona_health.test", "live", "true"),
					resource.TestCheckResourceAttr("data.daytona_health.test", "ready", "true"),
					resource.TestCheckResourceAttr("data.daytona_health.test", "live_http_status", "200"),
					resource.TestCheckResourceAttr("data.daytona_health.test", "ready_http_status", "200"),
					resource.TestCheckResourceAttrSet("data.daytona_health.test", "status"),
					resource.TestCheckResourceAttrSet("data.daytona_health.test", "response_json"),
				),
			},
		},
	})
}
