# Architecture

## Pattern

**Pipeline** orchestrated by a **Converter Facade**, with **ConverterPool** for parallelism.

```
                        Converter.Convert()
                               │
       ┌───────────┬───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼           ▼
   mdtransform   md2html   htmlinject     pdf       assets
   (internal)   (internal)  (internal)   (root)    (internal)
```

- **Converter Facade** - Single entry point, owns browser lifecycle
- **Pipeline** - Chained transformations in `internal/pipeline/`
- **ConverterPool** - Lazy browser init, parallel batch processing
- **Dependency Injection** - Components via interfaces

---

## Package Structure

See [LAYOUT.md](LAYOUT.md) for the complete project layout.

**Design decision**: PDF generation (`pdf.go`) stays in root rather than `internal/pipeline/` to avoid circular dependencies. It depends on root types (`PageSettings`, `Watermark`) and the clean separation keeps `internal/pipeline/` focused on document structure (MD->HTML) while root handles rendering concerns (HTML->PDF).

---

## Data Flow

```
Markdown ──▶ mdtransform ──▶ md2html ──▶ htmlinject ──▶ pdf ──▶ PDF
                │               │             │           │
           Normalize        Goldmark      Page breaks  Chrome
           Highlights       GFM/TOC IDs   Watermark    Headless
           Blank lines      Footnotes     Cover page   Footer
                                          TOC inject
                                          CSS inject
                                          Signature
```

| Stage           | Transformation | Location                        | Tool            |
| --------------- | -------------- | ------------------------------- | --------------- |
| **mdtransform** | MD -> MD       | `internal/pipeline/`            | Regex           |
| **md2html**     | MD -> HTML     | `internal/pipeline/`            | Goldmark (GFM)  |
| **htmlinject**  | HTML -> HTML   | `internal/pipeline/`            | String/template |
| **pdf**         | HTML -> PDF    | root (`pdf.go`)                 | Rod (Chrome)    |

---

## Injection Order

```
1. Page breaks CSS      ──▶  <head> (lowest priority)
2. Watermark CSS        ──▶  <head>
3. User CSS             ──▶  <head> (highest priority)
4. Cover page           ──▶  after <body>
5. TOC                  ──▶  after cover (or <body>)
6. Signature            ──▶  before </body>
7. Footer               ──▶  Chrome native footer
```

---

## Interfaces

Pipeline stages communicate through interfaces defined in `internal/pipeline/`:

| Interface              | Method                          | Purpose                     |
| ---------------------- | ------------------------------- | --------------------------- |
| `MarkdownPreprocessor` | `PreprocessMarkdown(ctx, md)`   | MD normalization, highlights |
| `HTMLConverter`        | `ToHTML(ctx, md)`               | MD -> HTML via Goldmark     |
| `CSSInjector`          | `InjectCSS(ctx, html, css)`     | CSS into `<head>`           |
| `CoverInjector`        | `InjectCover(ctx, html, data)`  | Cover after `<body>`        |
| `TOCInjector`          | `InjectTOC(ctx, html, data)`    | TOC after cover             |
| `SignatureInjector`    | `InjectSignature(ctx, html, data)` | Signature before `</body>` |

Root package interface:

| Interface      | Method                        | Purpose                     |
| -------------- | ----------------------------- | --------------------------- |
| `AssetLoader`  | `LoadStyle(name)`             | Load CSS by name            |
|                | `LoadTemplateSet(name)`       | Load cover/signature templates |

---

## Concurrency

```
┌─────────────────────────────────────────────────────────┐
│                    ConverterPool                        │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐           │
│  │ Converter │  │ Converter │  │ Converter │  ...      │
│  │ (Chrome)  │  │ (Chrome)  │  │ (Chrome)  │           │
│  └───────────┘  └───────────┘  └───────────┘           │
└─────────────────────────────────────────────────────────┘
         ▲              ▲              ▲
         │              │              │
    Acquire()      Acquire()      Acquire()
         │              │              │
    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
    │ Worker  │    │ Worker  │    │ Worker  │
    └─────────┘    └─────────┘    └─────────┘
```

- Each `Converter` owns one Chrome browser instance (~200MB RAM)
- `ConverterPool` manages N converters
- `ResolvePoolSize(0)` auto-selects 1-8 converters based on CPU cores
- Explicit library pool sizes are not capped; callers own memory/process risk
- The CLI caps `--workers` at `MaxPoolSize` for safer default operation
- Converters created **lazily** on first `Acquire()` - no startup delay
- `Acquire()` blocks when all converters are in use
- `Release()` returns converter to pool for reuse
- `context.Context` propagates through all pipeline stages for cancellation

---

## Browser Lifecycle

- Browsers created lazily on first `Acquire()` from pool
- `process.KillProcessGroup()` terminates Chrome + all child processes (GPU, renderer)
- Platform-specific: `syscall.Kill(-pid)` on Unix, `taskkill /T` on Windows
- Implementation in `internal/process/`

---

## CLI Commands

| Command      | Purpose                                | Location              |
| ------------ | -------------------------------------- | --------------------- |
| `convert`    | Markdown to PDF conversion             | `cmd/picoloom/convert.go` |
| `config`     | Config management (`init` wizard)      | `cmd/picoloom/config_init.go` |
| `doctor`     | System diagnostics (Chrome, container) | `cmd/picoloom/doctor.go`  |
| `completion` | Shell completion scripts               | `cmd/picoloom/completion.go` |
| `version`    | Show version information               | `cmd/picoloom/main.go`  |
| `help`       | Command help                           | `cmd/picoloom/help.go`  |

The `doctor` command performs system checks without starting a conversion:
- Chrome/Chromium detection (binary, version, sandbox status)
- Container detection (Docker, Podman, Kubernetes via multi-signal approach)
- CI environment detection (GitHub Actions, GitLab CI, Jenkins, CircleCI)
- Temp directory writability

`config init` architecture:
- **Input mode boundary** - interactive mode requires TTY; `--no-input` supports CI/scripts.
- **Prompt pipeline** - prompt + validation + inline YAML help (`?`) per field, then summary/preview confirmation.
- **Write safety** - destination lock file prevents concurrent writes to the same target.
- **Overwrite safety** - `--force` uses backup/rollback semantics and restores interrupted writes on next run.
- **Publish strategy** - temp file is validated via `config.LoadConfig` before atomic publish.

---

## Validation Architecture

The codebase follows a "validate at trust boundaries" pattern:

```
CLI Path:
  YAML/env/flags ──▶ merge precedence ──▶ Config.Validate() ──▶ buildXxxData() ──▶ validateInput()
                                                   ▲                  (no validation)        ▲
                                             BOUNDARY 1                               BOUNDARY 2

Library Path:
  User builds Input ─────────────────────────────────────▶ validateInput()
                                                               ▲
                                                          BOUNDARY 2
```

| Boundary | Location | Purpose |
| -------- | -------- | ------- |
| Config.Validate() | `internal/config/` + `cmd/picoloom/convert.go` | Validates final CLI config after YAML/env/flag precedence is resolved |
| validateInput() | `converter.go` | Validates library API input before processing |

**Design principles:**

- **CLI param builders trust config** - `buildXxxData()` functions in `cmd/picoloom/convert_params.go` transform config already validated after YAML/env/flag merging into library types without re-validation
- **Library validates at entry** - `validateInput()` is the trust boundary for direct library users
- **No redundant validation** - Each constraint is checked once at the appropriate boundary
- **Validation methods on types** - `PageSettings.Validate()`, `Cover.Validate()`, `Signature.Validate()`, etc.

Config-to-library builder parity tests detect shared constraint drift.

---

## Adding Features

| Feature Type        | Location                          | Example                      |
| ------------------- | --------------------------------- | ---------------------------- |
| New MD syntax       | `internal/pipeline/mdtransform.go`| `==highlight==` support      |
| New HTML injection  | `internal/pipeline/{feature}inject.go` | New metadata block           |
| New Input field     | `types.go` + `converter.go`       | Add to `Input` struct        |
| New CLI flag        | `cmd/picoloom/flags.go`             | Add flag definition          |
| New CLI command     | `cmd/picoloom/{name}.go`            | Add `doctor.go`              |
| New config option   | `internal/config/config.go`       | Add to `Config` struct       |
| New CSS style       | `internal/assets/styles/`         | Add `{name}.css`             |
| New template        | `internal/assets/templates/`      | Add `{name}/cover.html`      |

**Checklist for new features:**
1. Add types to `types.go` (if public) or internal package
2. Add `Validate()` method on the type (validates technical constraints only)
3. Call `Validate()` from `validateInput()` in `converter.go`
4. Add config validation in `internal/config/` (for CLI path)
5. Wire into `converter.go` pipeline
6. Add CLI flags in `cmd/picoloom/flags.go`
7. Add param builder in `cmd/picoloom/convert_params.go` (no validation - trusts config)
8. Add tests: unit + integration
9. Update README.md documentation

**Checklist for new CLI commands:**
1. Create `cmd/picoloom/{name}.go` with command logic
2. Register in `cmd/picoloom/main.go` switch statement
3. Add to `isCommand()` function
4. Add help text in `cmd/picoloom/help.go`
5. Add tests in `cmd/picoloom/{name}_test.go`
6. Update README.md documentation
