# Contributing

Contributions are welcome! Picoloom aims to be one of the best Markdown-to-PDF tools in the Go ecosystem.

## What I'm Looking For

**Actively seeking:**

- Documentation improvements
- Bug fixes with tests

**Requires discussion first:**

- New styles (see Contributing Styles below)
- New features: open an issue before coding
- Architectural changes
- New dependencies

## Setup

```bash
git clone https://github.com/alnah/picoloom.git
cd picoloom
git lfs install  # Required for example PDFs
git lfs pull     # Download example PDFs
```

### Lint Tooling

`make lint` expects `golangci-lint` to be available in your `PATH`.

Recommended install method: official binary release from golangci-lint docs.

If lint fails with a message about `built with go...`:

1. Check your Go toolchain: `go env GOVERSION`
2. Check your linter build toolchain: `golangci-lint version`
3. Reinstall `golangci-lint` so it is built with a Go version greater than or equal to your current toolchain.

This avoids intermittent crashes when the linter binary is older than the active Go toolchain.

## How to Contribute

### Bug Reports

Before reporting a bug, please check:

- [Known Limitations](README.md#known-limitations): your issue may be documented
- Existing issues: someone may have reported it already

If the issue is not documented, please open an issue with:

- OS and Go version
- Minimal reproduction steps
- Expected vs actual behavior

### Feature Requests

Open an issue to discuss before implementing. This ensures alignment and avoids wasted effort.

### Pull Requests

1. Reference an existing issue
2. Follow existing code patterns
3. Add tests for new functionality
4. Run `make check-all` before submitting
5. Enable "Allow edits from maintainers" when creating your PR - this lets me make small fixes directly instead of requesting changes, speeding up the review process

Contributions done with coding agents are welcome if you know what you are doing. If you used a coding agent, please be especially careful to explain in your PR how you worked with it. Thank you. I also used coding agents to create Picoloom, but I did not just ask stuff to my coding agent to build it for me.

### Contributing Styles

New CSS styles in `internal/assets/styles/` are also welcome. Follow the established structure (see `technical.css` as reference):

1. CSS Variables: Central configuration
2. Reset and base styles
3. Typography: Headings and hierarchy
4. Text formatting
5. Components: Blockquotes
6. Components: Lists (including task lists)
7. Components: Tables
8. Code and syntax highlighting (Chroma classes)
9. Images and media
10. Footnotes
11. Signature block
12. Chrome PDF specific rules (`@media all`)
13. Cover page
14. Table of contents
15. Print settings (`@page`, `@media print`)

Requirements:

- Use CSS variables for colors and spacing
- Include `-webkit-print-color-adjust: exact` for color preservation
- Test with actual PDF generation before submitting

### CSS Variables

All themes must define these core variables in `:root`:

```css
:root {
  /* Colors - Semantic */
  --color-fg-default: #...; /* Main text */
  --color-fg-muted: #...; /* Secondary text */
  --color-fg-subtle: #...; /* Tertiary text */
  --color-fg-on-emphasis: #...; /* Text on colored backgrounds */
  --color-canvas-default: #...; /* Page background */
  --color-canvas-subtle: #...; /* Code blocks, subtle backgrounds */
  --color-border-default: #...; /* Primary borders */
  --color-border-muted: #...; /* Subtle borders */
  --color-accent-fg: #...; /* Links, accents */
  --color-accent-emphasis: #...; /* Strong accents */

  /* Colors - Status */
  --color-success-fg: #...;
  --color-attention-fg: #...;
  --color-danger-fg: #...;

  /* Colors - Syntax highlighting */
  --color-syntax-comment: #...;
  --color-syntax-keyword: #...;
  --color-syntax-string: #...;
  --color-syntax-number: #...;
  --color-syntax-function: #...;
  --color-syntax-variable: #...;

  /* Highlight */
  --color-mark-bg: #...; /* <mark> background */

  /* Spacing */
  --spacing-xs: 0.25em;
  --spacing-sm: 0.5em;
  --spacing-md: 1em;
  --spacing-lg: 1.5em;
  --spacing-xl: 2em;

  /* Typography */
  --font-size-base: 11pt;
  --font-size-small: 0.9em;
  --font-size-tiny: 0.8em;
  --line-height: 1.6;

  /* Components */
  --checkbox-offset-top: -2px;
  --checkbox-offset-left: 2px;
  --signature-margin-top: 30px;
  --signature-image-radius: 4px; /* or 50% for circular */
  --cover-logo-radius: 0;
}
```

Themes may add additional variables for unique features (e.g., `creative.css` defines `--heading-red`, `--heading-yellow`, etc. for colored heading badges).

## Issue Labels

When you open an issue, I will add appropriate labels to help with triage.
So far, I am the only maintainer who can apply labels. Contributors are welcome to suggest appropriate labels in issue comments. New maintainers are welcome.

### Area

- `area/cli`: CLI commands and flags
- `area/pdf`: PDF generation and rendering
- `area/markdown`: Markdown parsing
- `area/css`: Styles and CSS themes
- `area/config`: Configuration files

### Platform

- `os/linux`, `os/macos`, `os/windows`: OS-specific
- `env/docker`, `env/ci`: Environment-specific

### Priority

- `priority/critical`: Blocks core functionality
- `priority/high`: Important issue
- `wontfix`: Won't be addressed

Looking to contribute? Filter by `good first issue` or `help wanted`.

## Code of Conduct

Be respectful. Technical disagreements are fine. Personal attacks are not.
