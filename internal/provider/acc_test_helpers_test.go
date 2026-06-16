package provider

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCompositeImportID builds an ImportStateIdFunc that joins two state
// attributes with a slash, matching the composite import IDs used by the
// organization-scoped resources (parseCompositeImportID splits on "/").
func testAccCompositeImportID(resourceName, firstAttr, secondAttr string) func(*terraform.State) (string, error) {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes[firstAttr], rs.Primary.Attributes[secondAttr]), nil
	}
}

// testAccDefaultRegionID returns the region id used as default_region_id when an
// acceptance test creates a throwaway organization. Daytona requires a valid
// default region, so this defaults to the shared "us" region and can be overridden
// with DAYTONA_ACC_DEFAULT_REGION_ID for other deployments.
func testAccDefaultRegionID() string {
	if v := os.Getenv("DAYTONA_ACC_DEFAULT_REGION_ID"); v != "" {
		return v
	}
	return "us"
}
