package picoloom_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const releaseGuardScriptPath = "scripts/ensure-v2-release-tag.sh"

func TestReleaseTagGuardAllowsOnlyV2SemverTags(t *testing.T) {
	script, err := filepath.Abs(releaseGuardScriptPath)
	if err != nil {
		t.Fatalf("filepath.Abs(release tag guard) error = %v", err)
	}

	tests := []struct {
		name                 string
		args                 []string
		goreleaserCurrentTag string
		wantSuccess          bool
	}{
		{name: "snapshot without explicit tag", wantSuccess: true},
		{name: "v2.1.0", args: []string{"v2.1.0"}, wantSuccess: true},
		{name: "v2.12.3", args: []string{"v2.12.3"}, wantSuccess: true},
		{name: "empty explicit tag", args: []string{""}},
		{name: "v0.1.0", args: []string{"v0.1.0"}},
		{name: "v1.0.0", args: []string{"v1.0.0"}},
		{name: "v3.0.0", args: []string{"v3.0.0"}},
		{name: "prerelease", args: []string{"v2.1.0-rc.1"}},
		{name: "missing v prefix", args: []string{"2.1.0"}},
		{name: "not semver", args: []string{"foo"}},
		{name: "env GORELEASER_CURRENT_TAG v2.1.0", goreleaserCurrentTag: "v2.1.0", wantSuccess: true},
		{name: "env GORELEASER_CURRENT_TAG v1.0.0", goreleaserCurrentTag: "v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			//nolint:gosec // Test executes a repository-local release guard with controlled table inputs.
			cmd := exec.CommandContext(ctx, "sh", append([]string{script}, tt.args...)...)
			cmd.Dir = t.TempDir()
			cmd.Env = append(os.Environ(), "GITHUB_REF_NAME=", "GORELEASER_CURRENT_TAG="+tt.goreleaserCurrentTag)
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("release tag guard timed out: %v; combined output:\n%s", ctx.Err(), output)
			}
			if tt.wantSuccess && err != nil {
				t.Fatalf("release tag guard error = %v, want success; combined output:\n%s", err, output)
			}
			if !tt.wantSuccess && err == nil {
				t.Fatalf("release tag guard succeeded for args %q, want rejection; combined output:\n%s", tt.args, output)
			}
			if !tt.wantSuccess && !failedWithExitCode(err) {
				t.Fatalf("release tag guard error = %v, want non-zero exit; combined output:\n%s", err, output)
			}
		})
	}
}

func TestReleaseWorkflowRestrictsAndGuardsTags(t *testing.T) {
	workflowPath := filepath.Join(".github", "workflows", "release.yml")
	//nolint:gosec // Test reads a fixed repository workflow path.
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", workflowPath, err)
	}
	text := string(content)

	if message := releaseWorkflowPolicyError(text); message != "" {
		t.Fatalf("%s %s; content:\n%s", workflowPath, message, text)
	}
}

func TestRenovateConfigCoversGoAndGitHubActions(t *testing.T) {
	//nolint:gosec // Test reads a fixed repository config path.
	content, err := os.ReadFile("renovate.json")
	if err != nil {
		t.Fatalf("os.ReadFile(renovate.json) error = %v", err)
	}

	var cfg struct {
		EnabledManagers []string `json:"enabledManagers"`
	}
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("renovate.json is not valid JSON: %v", err)
	}

	managers := make(map[string]bool, len(cfg.EnabledManagers))
	for _, manager := range cfg.EnabledManagers {
		managers[manager] = true
	}
	for _, manager := range []string{"gomod", "github-actions"} {
		if !managers[manager] {
			t.Fatalf("renovate.json enabledManagers = %v, want %q", cfg.EnabledManagers, manager)
		}
	}
}

func TestReleaseWorkflowRejectsNonV2TagPatterns(t *testing.T) {
	for _, pattern := range []string{"v0.*.*", "v1.*.*", "v3.*.*", "v[0-1]*.*.*", "v[3-9]*.*.*"} {
		t.Run(pattern, func(t *testing.T) {
			workflow := strings.Join([]string{
				"on:",
				"  push:",
				"    tags:",
				"      - v2.*.*",
				"      - " + pattern,
				"jobs:",
				"  release:",
				"    steps:",
				"      - run: " + releaseGuardScriptPath,
				"",
			}, "\n")
			if message := releaseWorkflowPolicyError(workflow); message == "" {
				t.Fatalf("release workflow policy accepted tag pattern %q, want rejection", pattern)
			}
		})
	}
}

func releaseWorkflowPolicyError(text string) string {
	patterns := releaseWorkflowTagPatterns(text)
	if len(patterns) == 0 {
		return "must define push tag filters"
	}

	hasV2ReleaseTag := false
	for _, pattern := range patterns {
		if pattern == "v2.*.*" {
			hasV2ReleaseTag = true
			continue
		}
		if strings.HasPrefix(pattern, "v") {
			return "tag pattern " + strconv.Quote(pattern) + " must be exactly v2.*.*"
		}
	}
	if !hasV2ReleaseTag {
		return "push tag filter must include v2.*.*"
	}
	if !strings.Contains(text, releaseGuardScriptPath) {
		return "must run " + releaseGuardScriptPath + " before release publishing"
	}
	return ""
}

func releaseWorkflowTagPatterns(text string) []string {
	var patterns []string
	inTags := false
	tagsIndent := -1

	for _, line := range strings.Split(text, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if inTags {
			if indent <= tagsIndent && !strings.HasPrefix(trimmedLine, "- ") {
				inTags = false
			} else if strings.HasPrefix(trimmedLine, "- ") {
				patterns = append(patterns, unquoteYAMLListValue(strings.TrimPrefix(trimmedLine, "- ")))
				continue
			}
		}
		if trimmedLine == "tags:" {
			inTags = true
			tagsIndent = indent
		}
	}
	return patterns
}

func unquoteYAMLListValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), "'\"")
}

func failedWithExitCode(err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError)
}
