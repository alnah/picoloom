# AGENTS.md

## Principles

- KISS: keep solutions simple.
- DRY on behavior, not at the cost of unclear data structures.
- Prefer small, explicit Go code over clever abstractions.
- Preserve public API behavior unless the task explicitly requires a change.

## Source of truth

- Codebase root: this repository.
- Project documentation: `README.md`, `docs/ARCHITECTURE.md`, `docs/LAYOUT.md`, `CONTRIBUTING.md`.
- Validation commands: `Makefile`, `.github/workflows/ci.yml`, `.golangci.yml`, `go.mod`.
- Stack references: `~/workspace/stack/` contains local source-of-truth clones named `owner.repo`.
- For Go questions, prefer relevant local stack sources before web search:
  - `~/workspace/stack/golang.go` for language, stdlib, tests, toolchain behavior.
  - `~/workspace/stack/golang.website` for go.dev docs and release notes.
  - `~/workspace/stack/yuin.goldmark` for Markdown parser behavior.
  - `~/workspace/stack/go-rod.rod` for browser lifecycle, context, timeout, Rod APIs.
  - `~/workspace/stack/goccy.go-yaml` for YAML behavior.
  - `~/workspace/stack/golangci.golangci-lint` for lint behavior.
- Read only task-relevant stack files. Do not bulk-audit stack.

## Project architecture

Picoloom is a Go library and CLI for Markdown-to-PDF conversion.

Main areas:

- Root package `github.com/alnah/picoloom/v2`: public library API, converter facade, PDF rendering, pool, public types.
- `cmd/picoloom`: CLI commands, flags, config wiring, batch conversion.
- `internal/pipeline`: Markdown preprocessing, Goldmark conversion, HTML injection, path rewriting.
- `internal/assets`: embedded/custom styles and templates.
- `internal/config`: YAML config and validation.
- `internal/process`: OS process cleanup for browser lifecycle.

Keep boundaries stable:

- Public API stays in the root package.
- Implementation details stay under `internal/`.
- CLI orchestration stays in `cmd/picoloom`.
- Avoid importing CLI code from library/internal packages.

## Go practices

- Use standard library first when reasonable.
- Keep `context.Context` as the first parameter when a function accepts context.
- Propagate context to I/O, browser, and long-running operations when supported.
- Make resource ownership explicit. Close browsers, pages, files, pools, timers, and temp resources.
- Wrap errors with useful context and `%w` when callers may need `errors.Is` or `errors.As`.
- Use `panic` only for programmer errors or unrecoverable initialization paths already established in the project.
- Prefer consumer-side interfaces at boundaries used by tests or alternate implementations.
- Do not introduce new dependencies without a clear need and validation.

## Files and names

- Prefer cohesive files. If a file grows beyond about 500 LOC, consider splitting by responsibility.
- Avoid generic names such as `utils`, `helpers`, or `common` for new files/packages.
- Names should describe responsibility and behavior.
- Keep comments short and contract-focused: inputs, outputs, side effects, edge cases.

## Testing

- Prefer behavior-focused tests over implementation-detail tests.
- Use table-driven tests for validation, parsing, config mapping, and boundary cases.
- Include happy paths, failure paths, edge cases, cleanup paths, and concurrency/race paths when relevant.
- For pipeline robustness, consider fuzz tests with small meaningful seeds.
- For browser/PDF behavior, use integration tests with the existing `integration` build tag.
- Keep tests deterministic and avoid real external services unless explicitly required.

Useful commands:

```bash
go test ./...
go test -race ./...
ROD_NO_SANDBOX=1 go test -race -tags=integration ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Validation before handoff

Run the safest relevant checks for the change. Prefer non-mutating checks first.

Common checks:

```bash
gofmt -w .
go mod verify
go vet ./...
golangci-lint run
go tool gosec ./...
govulncheck ./...
go test ./...
go test -race ./...
go build ./...
```

Integration checks when browser/PDF/CLI conversion is touched:

```bash
ROD_NO_SANDBOX=1 go test -race -tags=integration ./...
go build -o /tmp/picoloom ./cmd/picoloom
ROD_NO_SANDBOX=1 /tmp/picoloom convert examples/simple-report.md -o /tmp/picoloom-simple-test.pdf
```

If a command is skipped, state why.

## Dependency and security changes

- For dependency updates, run `go mod tidy`, `go mod verify`, tests, and `govulncheck ./...`.
- Prefer upgrading vulnerable direct dependencies over replacing stable stack components.
- Do not silence `govulncheck`, `gosec`, or linter findings without documented rationale.

## Refactoring

- Keep refactors behavior-preserving unless task says otherwise.
- Split large files by responsibility, not by arbitrary layers.
- Prefer small PR-sized changes.
- When renaming exported or widely-used symbols, prefer semantic tools such as `gopls rename` if available.
- Do not rewrite architecture unless evidence shows a concrete risk.

## Picoloom-specific cautions

- Goldmark raw HTML remains unsafe by design. Do not enable `html.WithUnsafe()` without explicit security review.
- Rod/Chrome lifecycle must avoid leaked browser processes. Preserve explicit close/kill behavior.
- `SourceDir` path rewriting must preserve path traversal protection.
- Config validation and public type validation are separate trust boundaries. Keep them aligned with tests.
- CLI flags, config, and environment precedence must match README behavior.
