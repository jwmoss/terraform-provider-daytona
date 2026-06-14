package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestVolumeResourceWaitForDeletable(t *testing.T) {
	previousPollInterval := volumePollInterval
	volumePollInterval = time.Millisecond
	t.Cleanup(func() {
		volumePollInterval = previousPollInterval
	})

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}
		if r.URL.EscapedPath() != "/volumes/volume-1" {
			t.Fatalf("expected path %q, got %q", "/volumes/volume-1", r.URL.EscapedPath())
		}

		requests++
		w.Header().Set("Content-Type", "application/json")
		state := "pending_create"
		if requests > 1 {
			state = "ready"
		}
		_, _ = fmt.Fprintf(w, `{"id":"volume-1","name":"volume-1","organizationId":"org-1","state":%q,"createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z","lastUsedAt":null,"errorReason":null}`, state)
	}))
	defer server.Close()

	volumeResource := &VolumeResource{client: newDaytonaClient(server.URL, "test-key", "", "test")}

	volume, _, err := volumeResource.waitForVolumeDeletable(context.Background(), "volume-1")
	if err != nil {
		t.Fatalf("unexpected wait error: %s", err)
	}
	if volume.State != "ready" {
		t.Fatalf("expected ready volume, got %s", volume.State)
	}
	if requests != 2 {
		t.Fatalf("expected 2 polling requests, got %d", requests)
	}
}

func TestAccVolumeResource_basic(t *testing.T) {
	testAccPreCheckAPIKey(t)

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
