package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// sweepResourcePrefix is the name prefix every acceptance test uses for the
// resources it creates. Sweepers only delete resources carrying this prefix so a
// sweep never touches real infrastructure.
const sweepResourcePrefix = "tf-acc-"

// TestMain wires Terraform's acceptance-test sweeper framework so a failed
// acceptance run can be cleaned up with `go test ./... -sweep=all`. It also
// shrinks the client retry backoff so tests that simulate transient 5xx
// responses retry quickly instead of waiting out the production backoff.
func TestMain(m *testing.M) {
	retryWaitMin = time.Millisecond
	retryWaitMax = 5 * time.Millisecond
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("daytona_volume", &resource.Sweeper{
		Name: "daytona_volume",
		F:    sweepVolumes,
	})
	resource.AddTestSweepers("daytona_sandbox", &resource.Sweeper{
		Name: "daytona_sandbox",
		F:    sweepSandboxes,
	})
	resource.AddTestSweepers("daytona_snapshot", &resource.Sweeper{
		Name: "daytona_snapshot",
		F:    sweepSnapshots,
	})
	resource.AddTestSweepers("daytona_docker_registry", &resource.Sweeper{
		Name: "daytona_docker_registry",
		F:    sweepDockerRegistries,
	})
}

// newSweepClient builds a Daytona client from the same environment variables the
// provider reads, returning an error when no API key is configured so sweepers
// fail loudly instead of silently doing nothing.
func newSweepClient() (*daytonaClient, error) {
	apiKey := os.Getenv("DAYTONA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DAYTONA_API_KEY must be set to run sweepers")
	}

	apiURL := os.Getenv("DAYTONA_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	return newDaytonaClient(apiURL, apiKey, os.Getenv("DAYTONA_ORGANIZATION_ID"), "sweeper"), nil
}

func sweepVolumes(string) error {
	client, err := newSweepClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	volumes, _, err := client.api.VolumesAPI.ListVolumes(ctx).Execute()
	if err != nil {
		return fmt.Errorf("listing volumes for sweep: %w", err)
	}

	for _, volume := range volumes {
		if !strings.HasPrefix(volume.Name, sweepResourcePrefix) {
			continue
		}
		if _, err := client.api.VolumesAPI.DeleteVolume(ctx, volume.Id).Execute(); err != nil {
			log.Printf("[ERROR] failed to sweep volume %s (%s): %s", volume.Name, volume.Id, err)
		}
	}

	return nil
}

func sweepSandboxes(string) error {
	client, err := newSweepClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	cursor := ""
	for {
		request := client.api.SandboxAPI.ListSandboxes(ctx).Limit(100)
		if cursor != "" {
			request = request.Cursor(cursor)
		}
		sandboxes, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("listing sandboxes for sweep: %w", err)
		}

		for _, sandbox := range sandboxes.Items {
			if !strings.HasPrefix(sandbox.Name, sweepResourcePrefix) {
				continue
			}
			if _, _, err := client.api.SandboxAPI.DeleteSandbox(ctx, sandbox.Id).Execute(); err != nil {
				log.Printf("[ERROR] failed to sweep sandbox %s (%s): %s", sandbox.Name, sandbox.Id, err)
			}
		}

		next := sandboxes.NextCursor.Get()
		if next == nil || *next == "" || *next == cursor || len(sandboxes.Items) == 0 {
			break
		}
		cursor = *next
	}

	return nil
}

func sweepSnapshots(string) error {
	client, err := newSweepClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for page := float32(1); ; page++ {
		snapshots, _, err := client.api.SnapshotsAPI.GetAllSnapshots(ctx).Page(page).Limit(100).Execute()
		if err != nil {
			return fmt.Errorf("listing snapshots for sweep: %w", err)
		}

		for _, snapshot := range snapshots.Items {
			if !strings.HasPrefix(snapshot.Name, sweepResourcePrefix) {
				continue
			}
			if _, err := client.api.SnapshotsAPI.RemoveSnapshot(ctx, snapshot.Id).Execute(); err != nil {
				log.Printf("[ERROR] failed to sweep snapshot %s (%s): %s", snapshot.Name, snapshot.Id, err)
			}
		}

		if page >= snapshots.TotalPages || len(snapshots.Items) == 0 {
			break
		}
	}

	return nil
}

func sweepDockerRegistries(string) error {
	client, err := newSweepClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	registries, _, err := client.api.DockerRegistryAPI.ListRegistries(ctx).Execute()
	if err != nil {
		return fmt.Errorf("listing Docker registries for sweep: %w", err)
	}

	for _, registry := range registries {
		if !strings.HasPrefix(registry.Name, sweepResourcePrefix) {
			continue
		}
		if _, err := client.api.DockerRegistryAPI.DeleteRegistry(ctx, registry.Id).Execute(); err != nil {
			log.Printf("[ERROR] failed to sweep Docker registry %s (%s): %s", registry.Name, registry.Id, err)
		}
	}

	return nil
}
