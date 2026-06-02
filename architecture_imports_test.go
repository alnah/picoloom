package picoloom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type listedPackage struct {
	ImportPath string
	Imports    []string
}

func TestImportBoundaries(t *testing.T) {
	modulePath := goListModulePath(t)
	packages := goListPackages(t)

	tests := []struct {
		name     string
		violates func(pkg listedPackage, imported string) bool
		explain  string
	}{
		{
			name: "internal pipeline stays leaf inside module",
			violates: func(pkg listedPackage, imported string) bool {
				return pkg.ImportPath == modulePath+"/internal/pipeline" && strings.HasPrefix(imported, modulePath+"/")
			},
			explain: "internal/pipeline must not import other project packages",
		},
		{
			name: "root does not import cmd packages",
			violates: func(pkg listedPackage, imported string) bool {
				return pkg.ImportPath == modulePath && strings.HasPrefix(imported, modulePath+"/cmd/")
			},
			explain: "root package must not import CLI packages",
		},
		{
			name: "root does not import internal config",
			violates: func(pkg listedPackage, imported string) bool {
				return pkg.ImportPath == modulePath && imported == modulePath+"/internal/config"
			},
			explain: "root package must not import internal/config",
		},
		{
			name: "internal config does not import cmd packages",
			violates: func(pkg listedPackage, imported string) bool {
				return pkg.ImportPath == modulePath+"/internal/config" && strings.HasPrefix(imported, modulePath+"/cmd/")
			},
			explain: "internal/config must not import CLI packages",
		},
		{
			name: "migrate command stays standalone",
			violates: func(pkg listedPackage, imported string) bool {
				return pkg.ImportPath == modulePath+"/cmd/picoloom-migrate" && strings.HasPrefix(imported, modulePath+"/")
			},
			explain: "cmd/picoloom-migrate must stay standalone and avoid project package imports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pkg := range packages {
				for _, imported := range pkg.Imports {
					if tt.violates(pkg, imported) {
						t.Errorf("%s: %s imports %s", tt.explain, pkg.ImportPath, imported)
					}
				}
			}
		})
	}
}

func goListModulePath(t *testing.T) string {
	t.Helper()

	output := runGoList(t, "go", "list", "-m")
	modulePath := strings.TrimSpace(string(output))
	if modulePath == "" {
		t.Fatal("go list -m returned empty module path")
	}
	return modulePath
}

func goListPackages(t *testing.T) []listedPackage {
	t.Helper()

	output := runGoList(t, "go", "list", "-json", "./...")
	dec := json.NewDecoder(bytes.NewReader(output))
	var packages []listedPackage
	for {
		var pkg listedPackage
		err := dec.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list -json ./...: %v\noutput:\n%s", err, output)
		}
		packages = append(packages, pkg)
	}
	if len(packages) == 0 {
		t.Fatal("go list -json ./... returned no packages")
	}
	return packages
}

func runGoList(t *testing.T, name string, args ...string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//nolint:gosec // Test executes the Go tool with fixed arguments to inspect this module.
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %v\noutput:\n%s", commandString(name, args...), ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("%s error = %v\noutput:\n%s", commandString(name, args...), err, output)
	}
	return output
}

func commandString(name string, args ...string) string {
	return strings.TrimSpace(fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
}
