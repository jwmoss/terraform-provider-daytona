package provider

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const daytonaAPIClientModule = "github.com/daytonaio/daytona/libs/api-client-go"
const expectedDaytonaAPIClientVersion = "v0.189.0"

func TestOpenAPIEndpointCoverage(t *testing.T) {
	t.Parallel()

	moduleVersion := daytonaAPIClientVersion(t)
	if moduleVersion != expectedDaytonaAPIClientVersion {
		t.Fatalf("expected %s %s, got %s", daytonaAPIClientModule, expectedDaytonaAPIClientVersion, moduleVersion)
	}

	sdkMethods := sdkAPIMethods(t, moduleVersion)
	providerMethods := providerAPIMethods(t)

	manualCoverage := map[string]string{
		"AdminAPI.AdminUpdateRunnerScheduling":           "covered by AdminRunnerResource.updateAdminRunnerScheduling because the generated request has no typed request body setter",
		"OrganizationsAPI.UpdateOrganizationRegionQuota": "covered by OrganizationRegionQuotaResource.applyOrganizationRegionQuota because the organization quota endpoint is patched manually",
		"RunnersAPI.UpdateRunnerDraining":                "covered by RunnerResource.updateRunnerDraining because the generated request has no typed request body setter",
		"RunnersAPI.UpdateRunnerScheduling":              "covered by RunnerResource.updateRunnerScheduling because the generated request has no typed request body setter",
	}
	intentionalSkips := map[string]string{
		"SandboxAPI.ListSandboxesPaginatedDeprecated": "deprecated endpoint replaced by SandboxAPI.ListSandboxes",
	}

	var missing []string
	for method := range sdkMethods {
		if providerMethods[method] {
			continue
		}
		if _, ok := manualCoverage[method]; ok {
			continue
		}
		if _, ok := intentionalSkips[method]; ok {
			continue
		}
		if strings.HasPrefix(method, "ToolboxAPI.") && strings.HasSuffix(method, "Deprecated") {
			continue
		}
		missing = append(missing, method)
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("uncovered Daytona OpenAPI methods:\n%s", strings.Join(missing, "\n"))
	}
}

func daytonaAPIClientVersion(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", daytonaAPIClientModule)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("unable to resolve %s version: %s\n%s", daytonaAPIClientModule, err, stderr.String())
	}
	return strings.TrimSpace(string(out))
}

func sdkAPIMethods(t *testing.T, moduleVersion string) map[string]bool {
	t.Helper()

	moduleDir := filepath.Join(goModCache(t), daytonaAPIClientModule+"@"+moduleVersion)
	files, err := filepath.Glob(filepath.Join(moduleDir, "api_*.go"))
	if err != nil {
		t.Fatalf("unable to glob SDK API files: %s", err)
	}
	if len(files) == 0 {
		t.Fatalf("no SDK API files found under %s", moduleDir)
	}

	re := regexp.MustCompile(`(?m)^func \(a \*([A-Za-z0-9_]+APIService)\) ([A-Za-z0-9_]+)\(`)
	methods := map[string]bool{}
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("unable to read %s: %s", file, err)
		}
		for _, match := range re.FindAllSubmatch(body, -1) {
			method := string(match[2])
			if strings.HasSuffix(method, "Execute") {
				continue
			}
			service := strings.TrimSuffix(string(match[1]), "Service")
			methods[service+"."+method] = true
		}
	}
	return methods
}

func providerAPIMethods(t *testing.T) map[string]bool {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate current test file")
	}
	providerDir := filepath.Dir(currentFile)
	files, err := filepath.Glob(filepath.Join(providerDir, "*.go"))
	if err != nil {
		t.Fatalf("unable to glob provider files: %s", err)
	}

	re := regexp.MustCompile(`(?s)\.api\.([A-Za-z0-9_]+API)\s*\.\s*([A-Za-z0-9_]+)\s*\(`)
	methods := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("unable to read %s: %s", file, err)
		}
		for _, match := range re.FindAllSubmatch(body, -1) {
			methods[string(match[1])+"."+string(match[2])] = true
		}
	}
	return methods
}

func goModCache(t *testing.T) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv("GOMODCACHE")); value != "" {
		return value
	}

	cmd := exec.Command("go", "env", "GOMODCACHE")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("unable to locate GOMODCACHE: %s\n%s", err, stderr.String())
	}
	return strings.TrimSpace(string(out))
}
