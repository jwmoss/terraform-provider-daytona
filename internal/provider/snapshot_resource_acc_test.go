package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSnapshotResource_basic creates and destroys a real Daytona snapshot.
// Creating a snapshot triggers a server-side image build, which is slow and may
// incur cost, so this test is opt-in via DAYTONA_ACC_SNAPSHOT_BUILD to keep it
// out of the default API-key CI run.
func TestAccSnapshotResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_SNAPSHOT_BUILD") == "" {
		t.Skip("set DAYTONA_ACC_SNAPSHOT_BUILD=1 to run the snapshot build acceptance test (builds a real image)")
	}

	name := "tf-acc-snapshot-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_snapshot.test", "id"),
					resource.TestCheckResourceAttr("daytona_snapshot.test", "name", name),
					resource.TestCheckResourceAttrSet("daytona_snapshot.test", "state"),
				),
			},
		},
	})
}

func testAccSnapshotResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_snapshot" "test" {
  name       = %[1]q
  image_name = "ubuntu:24.04"
}
`, name)
}
