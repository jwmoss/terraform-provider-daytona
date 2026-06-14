package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSandboxResource_basic creates and destroys a real Daytona sandbox using
// an API key. The snapshot reference matches the value exercised by the
// published example module, which is available to API-key-authenticated
// organizations.
func TestAccSandboxResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)

	name := "tf-acc-sandbox-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSandboxResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "id"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "name", name),
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "organization_id"),
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "state"),
				),
			},
		},
	})
}

func testAccSandboxResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_sandbox" "test" {
  name     = %[1]q
  snapshot = "daytonaio/sandbox:0.6.0"
}
`, name)
}
