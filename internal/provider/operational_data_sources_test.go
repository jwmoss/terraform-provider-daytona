package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOperationalDataSources_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "daytona" {}

data "daytona_config" "test" {}
data "daytona_current_user" "test" {}
data "daytona_current_user_organization_invitations" "test" {}
data "daytona_account_providers" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.daytona_config.test", "id", "config"),
					resource.TestCheckResourceAttrSet("data.daytona_config.test", "version"),
					resource.TestCheckResourceAttrSet("data.daytona_current_user.test", "id"),
					resource.TestCheckResourceAttrSet("data.daytona_current_user.test", "email"),
					resource.TestCheckResourceAttr("data.daytona_current_user_organization_invitations.test", "id", "current_user_organization_invitations"),
					resource.TestCheckResourceAttr("data.daytona_account_providers.test", "id", "account_providers"),
				),
			},
		},
	})
}
