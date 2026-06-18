package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCurrentAPIKeyDataSource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "daytona" {}

data "daytona_current_api_key" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.daytona_current_api_key.test", "name"),
					resource.TestCheckResourceAttrSet("data.daytona_current_api_key.test", "permissions.#"),
					resource.TestCheckResourceAttrSet("data.daytona_current_api_key.test", "user_id"),
				),
			},
		},
	})
}
