package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDockerRegistryResource_basic creates and destroys a real Daytona Docker
// registry record using an API key. The registry stores credential metadata, so
// the test uses placeholder credentials and does not contact any external
// registry. Creating a registry requires the WRITE_REGISTRIES organization
// permission, which not every API key carries, so the test is opt-in via
// DAYTONA_ACC_REGISTRY to avoid failing the default API-key CI run on a 403.
func TestAccDockerRegistryResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_REGISTRY") == "" {
		t.Skip("set DAYTONA_ACC_REGISTRY=1 to run the Docker registry acceptance test (needs the WRITE_REGISTRIES permission)")
	}

	name := "tf-acc-registry-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with project unset, exercising the optional/empty mapping.
				Config: testAccDockerRegistryResourceConfig(name, "terraform", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_docker_registry.test", "id"),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "name", name),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "url", "registry.example.com"),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "username", "terraform"),
					resource.TestCheckNoResourceAttr("daytona_docker_registry.test", "project"),
				),
			},
			{
				// Update username and set project to confirm in-place update works.
				Config: testAccDockerRegistryResourceConfig(name, "terraform-updated", "team-namespace"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "username", "terraform-updated"),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "project", "team-namespace"),
				),
			},
			{
				ResourceName:            "daytona_docker_registry.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccDockerRegistryResourceConfig(name, username, project string) string {
	projectLine := ""
	if project != "" {
		projectLine = fmt.Sprintf("  project  = %q\n", project)
	}
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_docker_registry" "test" {
  name     = %[1]q
  url      = "registry.example.com"
  username = %[2]q
  password = "tf-acc-placeholder-password"
%[3]s}
`, name, username, projectLine)
}
