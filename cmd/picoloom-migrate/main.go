package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	oldModulePath = "github.com/alnah/go-md2pdf"
	newModulePath = "github.com/alnah/picoloom/v2"
)

var (
	textExtensions = map[string]bool{
		".go":   true,
		".mod":  true,
		".md":   true,
		".txt":  true,
		".sh":   true,
		".yaml": true,
		".yml":  true,
		".toml": true,
		".env":  true,
	}
	textBasenames = map[string]bool{
		"Dockerfile": true,
		"Makefile":   true,
	}
	skipDirs = map[string]bool{
		".git":         true,
		".hg":          true,
		".svn":         true,
		".idea":        true,
		".vscode":      true,
		"dist":         true,
		"node_modules": true,
		"vendor":       true,
	}
)

type fileChange struct {
	path         string
	original     []byte
	updated      []byte
	replacements int
}

func main() {
	var write bool
	var verbose bool

	flag.BoolVar(&write, "write", false, "rewrite files in place")
	flag.BoolVar(&verbose, "verbose", false, "print unchanged skipped paths while walking")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [--write] [--verbose] [path ...]\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Rewrites common go-md2pdf references to picoloom/v2.")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Examples:")
		fmt.Fprintln(flag.CommandLine.Output(), "  picoloom-migrate .")
		fmt.Fprintln(flag.CommandLine.Output(), "  picoloom-migrate --write ./...")
	}
	flag.Parse()

	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	changes, err := collectChanges(roots, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		fmt.Fprintln(os.Stdout, "No rewrites needed.")
		return
	}

	if !write {
		fmt.Fprintf(os.Stdout, "Would rewrite %d file(s):\n", len(changes))
		for _, change := range changes {
			fmt.Fprintf(os.Stdout, "  %s (%d replacement(s))\n", change.path, change.replacements)
		}
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Re-run with --write to apply changes.")
		return
	}

	for _, change := range changes {
		if err := os.WriteFile(change.path, change.updated, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", change.path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "rewrote %s (%d replacement(s))\n", change.path, change.replacements)
	}

	if err := gofmtTouchedFiles(changes); err != nil {
		fmt.Fprintf(os.Stderr, "warning: gofmt failed: %v\n", err)
	}
}

func collectChanges(roots []string, verbose bool) ([]fileChange, error) {
	var changes []fileChange
	seen := make(map[string]bool)

	for _, root := range roots {
		expanded := root
		if root == "./..." {
			expanded = "."
		}

		info, err := os.Stat(expanded)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", expanded, err)
		}

		if !info.IsDir() {
			change, ok, err := analyzePath(expanded)
			if err != nil {
				return nil, err
			}
			if ok && !seen[change.path] {
				seen[change.path] = true
				changes = append(changes, change)
			}
			continue
		}

		err = filepath.WalkDir(expanded, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.IsDir() {
				if skipDirs[d.Name()] && path != expanded {
					return filepath.SkipDir
				}
				return nil
			}

			change, ok, err := analyzePath(path)
			if err != nil {
				return err
			}
			if ok && !seen[change.path] {
				seen[change.path] = true
				changes = append(changes, change)
			} else if verbose && !ok {
				fmt.Fprintf(os.Stdout, "skip %s\n", path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return changes, nil
}

func analyzePath(path string) (fileChange, bool, error) {
	if !shouldProcessPath(path) {
		return fileChange{}, false, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fileChange{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	if !utf8.Valid(content) {
		return fileChange{}, false, nil
	}

	updated, replacements := rewriteContent(path, content)
	if replacements == 0 || bytes.Equal(content, updated) {
		return fileChange{}, false, nil
	}

	return fileChange{
		path:         path,
		original:     content,
		updated:      updated,
		replacements: replacements,
	}, true, nil
}

func shouldProcessPath(path string) bool {
	base := filepath.Base(path)
	if textBasenames[base] {
		return true
	}
	return textExtensions[filepath.Ext(path)]
}

func rewriteContent(path string, content []byte) ([]byte, int) {
	updated := string(content)
	replacements := 0

	for _, pair := range replacementPairs(path) {
		count := strings.Count(updated, pair.old)
		if count == 0 {
			continue
		}
		updated = strings.ReplaceAll(updated, pair.old, pair.new)
		replacements += count
	}

	return []byte(updated), replacements
}

type replacementPair struct {
	old string
	new string
}

func replacementPairs(path string) []replacementPair {
	pairs := []replacementPair{
		{old: oldModulePath, new: newModulePath},
		{old: "ghcr.io/alnah/go-md2pdf", new: "ghcr.io/alnah/picoloom"},
		{old: "cmd/md2pdf", new: "cmd/picoloom"},
		{old: "MD2PDF_", new: "PICOLOOM_"},
		{old: "md2pdf.yaml", new: "picoloom.yaml"},
	}

	if filepath.Ext(path) == ".go" {
		pairs = append(pairs,
			replacementPair{old: "package md2pdf_test", new: "package picoloom_test"},
			replacementPair{old: "package md2pdf", new: "package picoloom"},
			replacementPair{old: "md2pdf.", new: "picoloom."},
			replacementPair{old: `md2pdf "github.com/alnah/picoloom/v2"`, new: `picoloom "github.com/alnah/picoloom/v2"`},
		)
	}

	return pairs
}

func gofmtTouchedFiles(changes []fileChange) error {
	var goFiles []string
	for _, change := range changes {
		if filepath.Ext(change.path) == ".go" {
			goFiles = append(goFiles, change.path)
		}
	}
	if len(goFiles) == 0 {
		return nil
	}

	cmd := exec.Command("gofmt", append([]string{"-w"}, goFiles...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
