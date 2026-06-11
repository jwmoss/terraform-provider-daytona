// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionDataSources_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "daytona" {}

data "daytona_volumes" "test" {}
data "daytona_regions" "test" {}
data "daytona_organizations" "test" {}
data "daytona_sandboxes" "test" {}
data "daytona_snapshots" "test" {}
data "daytona_docker_registries" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.daytona_volumes.test", "id", "volumes"),
					resource.TestCheckResourceAttr("data.daytona_regions.test", "id", "regions"),
					resource.TestCheckResourceAttr("data.daytona_organizations.test", "id", "organizations"),
					resource.TestCheckResourceAttr("data.daytona_sandboxes.test", "id", "sandboxes"),
					resource.TestCheckResourceAttr("data.daytona_snapshots.test", "id", "snapshots"),
					resource.TestCheckResourceAttr("data.daytona_docker_registries.test", "id", "docker_registries"),
				),
			},
		},
	})
}
