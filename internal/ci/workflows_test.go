package ci_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readWorkflow(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", ".github", "workflows", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return string(content)
}

func requireContains(t *testing.T, content string, want string) {
	t.Helper()

	if !strings.Contains(content, want) {
		t.Fatalf("expected workflow to contain %q", want)
	}
}

func requireNotContains(t *testing.T, content string, want string) {
	t.Helper()

	if strings.Contains(content, want) {
		t.Fatalf("expected workflow not to contain %q", want)
	}
}

func TestCIWorkflowUsesCostGuards(t *testing.T) {
	content := readWorkflow(t, "ci.yml")

	requireContains(t, content, "concurrency:")
	requireContains(t, content, "group: ${{ github.workflow }}-${{ github.ref }}")
	requireContains(t, content, "cancel-in-progress: true")
	requireContains(t, content, "paths-ignore:")
	requireContains(t, content, "- '**.md'")
	requireContains(t, content, "- 'docs/**'")
	requireContains(t, content, "if: github.event_name != 'pull_request' || github.event.pull_request.draft == false")
	requireContains(t, content, "nix-community/cache-nix-action")
	requireContains(t, content, "nix develop --accept-flake-config --impure -c just lint")
	requireContains(t, content, "just test && go test -v -race -coverprofile=coverage.out -covermode=atomic ./...")
	requireContains(t, content, "nix develop --accept-flake-config --impure -c gitleaks detect --source . --redact --exit-code 1")
	requireContains(t, content, "cachix/install-nix-action")
}

func TestSonarWorkflowUsesCostGuards(t *testing.T) {
	content := readWorkflow(t, "sonar.yml")

	requireContains(t, content, "concurrency:")
	requireContains(t, content, "group: ${{ github.workflow }}-${{ github.ref }}")
	requireContains(t, content, "cancel-in-progress: true")
	requireContains(t, content, "paths-ignore:")
	requireContains(t, content, "- '**.md'")
	requireContains(t, content, "- 'docs/**'")
	requireContains(t, content, "if: github.event_name != 'pull_request' || github.event.pull_request.draft == false")
	requireContains(t, content, "nix-community/cache-nix-action")
	requireContains(t, content, "nix develop --accept-flake-config --impure -c go test -coverprofile=coverage.out -covermode=atomic ./...")
}

func TestReleaseWorkflowUsesNixAndConcurrency(t *testing.T) {
	content := readWorkflow(t, "release.yml")

	requireContains(t, content, "concurrency:")
	requireContains(t, content, "group: ${{ github.workflow }}-${{ github.ref }}")
	requireContains(t, content, "cancel-in-progress: true")
	requireContains(t, content, "nix-community/cache-nix-action")
	requireContains(t, content, "nix develop --accept-flake-config --impure -c goreleaser release --clean")
}

func TestPRTitleWorkflowKeepsOnlyLightweightGuards(t *testing.T) {
	content := readWorkflow(t, "pr-title.yml")

	requireContains(t, content, "concurrency:")
	requireContains(t, content, "group: ${{ github.workflow }}-${{ github.ref }}")
	requireContains(t, content, "cancel-in-progress: true")
	requireNotContains(t, content, "paths-ignore:")
	requireNotContains(t, content, "nix-community/cache-nix-action")
	requireNotContains(t, content, "github.event.pull_request.draft == false")
}
