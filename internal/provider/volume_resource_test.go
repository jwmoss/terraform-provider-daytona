// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVolumeResource_basic(t *testing.T) {
	testAccPreCheck(t)

	name := "tf-acc-volume-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_volume.test", "id"),
					resource.TestCheckResourceAttr("daytona_volume.test", "name", name),
					resource.TestCheckResourceAttrSet("daytona_volume.test", "organization_id"),
					resource.TestCheckResourceAttrSet("daytona_volume.test", "state"),
				),
			},
		},
	})
}

func testAccVolumeResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_volume" "test" {
  name = %[1]q
}
`, name)
}
