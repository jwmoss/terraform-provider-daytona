package provider

import (
	"fmt"
	"os"
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

func TestAccSandboxResource_volumeMounts(t *testing.T) {
	testAccPreCheckAPIKey(t)

	sandboxName := "tf-acc-sandbox-vol-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	volumeName := "tf-acc-volume-mount-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSandboxResourceVolumeMountConfig(sandboxName, volumeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_volume.test", "id"),
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "id"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "volumes.#", "1"),
					resource.TestCheckResourceAttrPair("daytona_sandbox.test", "volumes.0.volume_id", "daytona_volume.test", "id"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "volumes.0.mount_path", "/workspace/daytona-volume"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "volumes.0.subpath", "terraform-provider-daytona"),
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "last_activity_at"),
				),
			},
		},
	})
}

func TestAccSandboxResource_gpuTypes(t *testing.T) {
	testAccPreCheckAPIKey(t)
	if os.Getenv("DAYTONA_ACC_GPU") == "" {
		t.Skip("set DAYTONA_ACC_GPU=1 to run the sandbox GPU acceptance test")
	}

	name := "tf-acc-sandbox-gpu-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSandboxResourceGPUConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("daytona_sandbox.test", "id"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "gpu_types.#", "1"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "gpu_types.0", "H100"),
					resource.TestCheckResourceAttr("daytona_sandbox.test", "gpu_type", "H100"),
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

func testAccSandboxResourceVolumeMountConfig(sandboxName string, volumeName string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_volume" "test" {
  name = %[2]q
}

resource "daytona_sandbox" "test" {
  name     = %[1]q
  snapshot = "daytonaio/sandbox:0.6.0"

  volumes = [
    {
      volume_id  = daytona_volume.test.id
      mount_path = "/workspace/daytona-volume"
      subpath    = "terraform-provider-daytona"
    }
  ]
}
`, sandboxName, volumeName)
}

func testAccSandboxResourceGPUConfig(name string) string {
	return fmt.Sprintf(`
provider "daytona" {}

resource "daytona_sandbox" "test" {
  name                 = %[1]q
  gpu                  = 1
  gpu_types            = ["H100"]
  auto_delete_interval = 0

  build_info = {
    dockerfile_content = "FROM ubuntu:24.04\nRUN echo terraform-provider-daytona\n"
  }
}
`, name)
}
