package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/launcher"
)

// doctorResult holds all diagnostic information.
type doctorResult struct {
	Status   string     `json:"status"` // "ready", "warnings", "errors"
	Chrome   chromeInfo `json:"chrome"`
	Env      envInfo    `json:"environment"`
	System   systemInfo `json:"system"`
	Warnings []string   `json:"warnings,omitempty"`
	Errors   []string   `json:"errors,omitempty"`
}

// chromeInfo holds Chrome/Chromium detection results.
type chromeInfo struct {
	Found   bool   `json:"found"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Sandbox bool   `json:"sandbox"`
}

// envInfo holds environment detection results.
type envInfo struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Container     bool   `json:"container"`
	ContainerHint string `json:"container_hint,omitempty"`
	CI            bool   `json:"ci"`
	NoSandbox     string `json:"rod_no_sandbox"`
	BrowserBin    string `json:"rod_browser_bin"`
}

// systemInfo holds system check results.
type systemInfo struct {
	TempWritable bool `json:"temp_writable"`
}

// runDoctorCmd executes the doctor command and returns an exit code.
// Exit codes: 0 = OK (including warnings), 1 = errors found.
func runDoctorCmd(args []string, env *Environment) int {
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	result := runDoctor()

	if jsonOutput {
		enc := json.NewEncoder(env.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		printDoctorResult(env.Stdout, result, envCLIName(env))
	}

	if result.Status == "errors" {
		return ExitGeneral
	}
	return ExitSuccess
}

// runDoctor performs all diagnostic checks.
func runDoctor() *doctorResult {
	result := &doctorResult{
		Status: "ready",
		Env: envInfo{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			NoSandbox:  os.Getenv("ROD_NO_SANDBOX"),
			BrowserBin: os.Getenv("ROD_BROWSER_BIN"),
		},
	}

	checkChrome(result)
	checkEnvironment(result)
	checkSystem(result)

	// Determine final status
	if len(result.Errors) > 0 {
		result.Status = "errors"
	} else if len(result.Warnings) > 0 {
		result.Status = "warnings"
	}

	return result
}

// checkChrome detects Chrome/Chromium installation.
func checkChrome(result *doctorResult) {
	chromePath := result.Env.BrowserBin

	if chromePath == "" {
		// Use rod's launcher to locate Chrome
		var found bool
		chromePath, found = launcher.LookPath()
		if !found {
			result.Errors = append(result.Errors,
				"Chrome/Chromium not found. Install Chrome or set ROD_BROWSER_BIN")
			return
		}
	}

	// Verify it exists
	if _, err := os.Stat(chromePath); err != nil {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Chrome not found at %s", chromePath))
		return
	}

	result.Chrome.Found = true
	result.Chrome.Path = chromePath

	// Get version by running chrome --version.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// #nosec G204 -- chromePath comes from launcher.LookPath() or ROD_BROWSER_BIN env var
	cmd := exec.CommandContext(ctx, chromePath, "--version")
	out, err := cmd.Output()
	if err == nil {
		result.Chrome.Version = strings.TrimSpace(string(out))
	} else {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not get Chrome version: %v", err))
	}

	// Sandbox status: disabled if ROD_NO_SANDBOX=1
	result.Chrome.Sandbox = result.Env.NoSandbox != "1"
}

// checkEnvironment detects container and CI environments.
func checkEnvironment(result *doctorResult) {
	// Detect container (multi-signal approach)
	result.Env.Container, result.Env.ContainerHint = isContainer()

	// Detect CI environments
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			result.Env.CI = true
			break
		}
	}

	// Warn if container/CI without sandbox disabled
	if (result.Env.Container || result.Env.CI) && result.Env.NoSandbox != "1" {
		result.Warnings = append(result.Warnings,
			"Container/CI detected but ROD_NO_SANDBOX not set. Set ROD_NO_SANDBOX=1")
	}
}

// isContainer detects if running in a container environment.
// Returns (isContainer, hint) where hint indicates which signal was detected.
func isContainer() (bool, string) {
	// Explicit override (highest priority)
	if os.Getenv("PICOLOOM_CONTAINER") == "1" {
		return true, "PICOLOOM_CONTAINER=1"
	}
	if os.Getenv("MD2PDF_CONTAINER") == "1" {
		return true, "MD2PDF_CONTAINER=1"
	}
	// Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true, "/.dockerenv"
	}
	// Podman / systemd-nspawn / general container indicator
	if v := os.Getenv("container"); v != "" {
		return true, "container=" + v
	}
	// Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true, "KUBERNETES_SERVICE_HOST"
	}
	return false, ""
}

// checkSystem verifies system requirements.
func checkSystem(result *doctorResult) {
	// Check temp directory is writable
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "picoloom-doctor-test")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Temp directory not writable: %s", tmpDir))
	} else {
		_ = os.Remove(testFile)
		result.System.TempWritable = true
	}
}

// printDoctorResult outputs human-readable diagnostic results.
func printDoctorResult(w io.Writer, r *doctorResult, cliName string) {
	fmt.Fprintf(w, "%s doctor\n", cliName)
	fmt.Fprintln(w)

	// Chrome section
	fmt.Fprintln(w, "Chrome/Chromium")
	if r.Chrome.Found {
		fmt.Fprintf(w, "  [OK] Found at %s\n", r.Chrome.Path)
		if r.Chrome.Version != "" {
			fmt.Fprintf(w, "  [OK] Version: %s\n", r.Chrome.Version)
		}
		if r.Chrome.Sandbox {
			fmt.Fprintln(w, "  [OK] Sandbox: enabled")
		} else {
			fmt.Fprintln(w, "  [OK] Sandbox: disabled (ROD_NO_SANDBOX=1)")
		}
	} else {
		fmt.Fprintln(w, "  [ERROR] Not found")
	}
	fmt.Fprintln(w)

	// Environment section
	fmt.Fprintln(w, "Environment")
	fmt.Fprintf(w, "  [OK] Platform: %s/%s\n", r.Env.OS, r.Env.Arch)
	if r.Env.Container {
		fmt.Fprintf(w, "  [OK] Container: detected (%s)\n", r.Env.ContainerHint)
	}
	if r.Env.CI {
		fmt.Fprintln(w, "  [OK] CI: detected")
	}
	fmt.Fprintln(w)

	// System section
	fmt.Fprintln(w, "System")
	if r.System.TempWritable {
		fmt.Fprintln(w, "  [OK] Temp directory: writable")
	} else {
		fmt.Fprintln(w, "  [ERROR] Temp directory: not writable")
	}
	fmt.Fprintln(w)

	// Warnings
	if len(r.Warnings) > 0 {
		fmt.Fprintln(w, "Warnings:")
		for _, warn := range r.Warnings {
			fmt.Fprintf(w, "  [WARN] %s\n", warn)
		}
		fmt.Fprintln(w)
	}

	// Errors
	if len(r.Errors) > 0 {
		fmt.Fprintln(w, "Errors:")
		for _, err := range r.Errors {
			fmt.Fprintf(w, "  [ERROR] %s\n", err)
		}
		fmt.Fprintln(w)
	}

	// Final status
	switch r.Status {
	case "ready":
		fmt.Fprintln(w, "Status: Ready to convert")
	case "warnings":
		fmt.Fprintln(w, "Status: Ready with warnings")
	case "errors":
		fmt.Fprintln(w, "Status: Not ready (see errors above)")
	}
}
