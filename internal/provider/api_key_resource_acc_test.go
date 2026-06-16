package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAPIKeyResource_basic creates a real Daytona API key, verifies the
// generated value and metadata, imports it, and destroys it.
//
// API key creation is rejected for API-key auth (HTTP 401); it requires an
// interactive access token (org API / JWT), so this test uses the access-token
// pre-check.
func TestAccAPIKeyResource_basic(t *testing.T) {
	testAccPreCheckAccessToken(t)

	name := "tf-acc-apikey-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_api_key.test", "name", name),
					resource.TestCheckResourceAttr("daytona_api_key.test", "id", name),
					resource.TestCheckResourceAttrSet("daytona_api_key.test", "value"),
					resource.TestCheckResourceAttrSet("daytona_api_key.test", "created_at"),
					resource.TestCheckResourceAttr("daytona_api_key.test", "permissions.#", "1"),
				),
			},
			{
				ResourceName:      "daytona_api_key.test",
				ImportState:       true,
				ImportStateVerify: true,
				// value is only returned at creation; user_id and last_used_at are
				// only available from the read endpoint (null right after create);
				// expires_at is not reflected back through import.
				ImportStateVerifyIgnore: []string{"value", "user_id", "last_used_at", "expires_at"},
			},
		},
	})
}

func testAccAPIKeyResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_api_key" "test" {
  name        = %[1]q
  permissions = ["read:volumes"]
}
`, name)
}
