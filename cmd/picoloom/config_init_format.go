package main

import "strings"

// formatConfigInitYAML inserts spacing between top-level sections so generated
// files are easier to review and edit manually.
func formatConfigInitYAML(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	content := string(data)
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		return data
	}

	var builder strings.Builder
	topLevelKeyCount := 0
	for i, line := range lines {
		if isTopLevelYAMLKeyLine(line) {
			if topLevelKeyCount > 0 {
				builder.WriteByte('\n')
			}
			topLevelKeyCount++
		}

		builder.WriteString(line)
		if i < len(lines)-1 {
			builder.WriteByte('\n')
		}
	}
	if hasTrailingNewline {
		builder.WriteByte('\n')
	}

	return []byte(builder.String())
}

// isTopLevelYAMLKeyLine scopes spacing rules to top-level keys only, preserving
// nested YAML structure untouched.
func isTopLevelYAMLKeyLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	return strings.Contains(line, ":")
}
