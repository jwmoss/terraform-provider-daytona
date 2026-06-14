// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDockerRegistryResource_basic creates and destroys a real Daytona Docker
// registry record using an API key. The registry stores credential metadata, so
// the test uses placeholder credentials and does not contact any external
// registry.
func TestAccDockerRegistryResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)

	name := "tf-acc-registry-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerRegistryResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_docker_registry.test", "id"),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "name", name),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "url", "registry.example.com"),
					resource.TestCheckResourceAttr("daytona_docker_registry.test", "username", "terraform"),
				),
			},
		},
	})
}

func testAccDockerRegistryResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_docker_registry" "test" {
  name     = %[1]q
  url      = "registry.example.com"
  username = "terraform"
  password = "tf-acc-placeholder-password"
}
`, name)
}
