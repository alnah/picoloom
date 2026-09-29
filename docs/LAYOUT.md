# Project Layout

```
picoloom/                       # package picoloom (library)
│
├── doc.go                      # Package documentation (godoc)
├── converter.go                # NewConverter(), Convert(), Close() - facade
├── pool.go                     # ConverterPool, ResolvePoolSize()
├── types.go                    # Input, PageSettings, Footer, Signature, Watermark, Cover, TOC, PageBreaks, Options, Validate() methods
├── assets.go                   # AssetLoader, TemplateSet, NewAssetLoader(), NewTemplateSet()
├── errors.go                   # Sentinel errors
├── pdf.go                      # HTML -> PDF (Rod/Chrome)
├── cssbuilders.go              # Watermark/PageBreaks/Admonition CSS overlays
├── example_test.go             # Runnable examples for godoc (Example*, ExampleConverterPool, etc.)
│
├── cmd/picoloom/               # CLI (picoloom convert|config|doctor|version|help|completion)
│   ├── main.go                 # Entry point, command dispatch
│   ├── exit_codes.go           # Semantic exit codes (0-4) and exitCodeFor()
│   ├── convert.go              # Convert command orchestration
│   ├── convert_batch.go        # Batch processing, worker pool
│   ├── convert_params.go       # Parameter builders (cover, signature, footer, etc.)
│   ├── convert_discovery.go    # File discovery, output path resolution
│   ├── config_init.go          # Config init wizard, prompts, and safe file publishing
│   ├── config_init_test.go     # Unit + acceptance-style command behavior tests
│   ├── config_init_integration_test.go # Integration tests for file lifecycle safety
│   ├── doctor.go               # Doctor command (system diagnostics)
│   ├── flags.go                # Flag definitions by category
│   ├── help.go                 # Usage text
│   ├── env.go                  # Environment (Now, Stdout, Stderr, AssetLoader)
│   ├── env_config.go           # Environment variable configuration
│   ├── completion.go           # Shell completion command, flag/command definitions
│   ├── completion_{bash,zsh,fish,pwsh}.go  # Shell-specific generators
│   └── signal_{unix,windows}.go
│
├── internal/
│   ├── assets/                 # Asset loading (styles, templates)
│   │   ├── assets.go           # Loader interface and factory
│   │   ├── embedded.go         # Embedded assets (go:embed)
│   │   ├── filesystem.go       # Filesystem-based loader
│   │   ├── resolver.go         # Asset resolution logic
│   │   ├── templateset.go      # Template set management
│   │   ├── validation.go       # Asset validation
│   │   ├── styles/             # Embedded CSS styles
│   │   │   ├── default.css
│   │   │   ├── technical.css
│   │   │   ├── creative.css
│   │   │   ├── academic.css
│   │   │   ├── corporate.css
│   │   │   ├── legal.css
│   │   │   ├── invoice.css
│   │   │   └── manuscript.css
│   │   ├── templates/default/  # Default HTML templates
│   │   │   ├── cover.html
│   │   │   └── signature.html
│   │   └── overlays/           # Embedded structural CSS overlays
│   │       ├── admonition.css
│   │       ├── pagebreaks.css.tmpl
│   │       └── watermark.css.tmpl
│   ├── config/                 # YAML config, validation
│   ├── dateutil/               # Date format parsing, ResolveDate()
│   ├── fileutil/               # File utilities (FileExists, IsFilePath, IsURL)
│   ├── hints/                  # Actionable error message hints
│   ├── pipeline/               # Conversion pipeline components
│   │   ├── admonition.go       # Admonition node, renderer, extension
│   │   ├── admonition_fence.go # ::: fence block parser
│   │   ├── admonition_quote.go # > [!TYPE] blockquote transformer
│   │   ├── mdtransform.go      # MD -> MD (preprocessing)
│   │   ├── md2html.go          # MD -> HTML (Goldmark)
│   │   ├── cssinject.go        # CSS injection and sanitization
│   │   ├── coverinject.go      # Cover page template injection
│   │   ├── signatureinject.go  # Signature template injection
│   │   ├── tocinject.go        # TOC extraction, numbering, injection
│   │   ├── footer.go           # Footer data shared with PDF rendering
│   │   └── pathrewrite.go      # Rewrite relative paths for SourceDir
│   ├── process/                # OS-specific process management
│   │   ├── kill_unix.go        # KillProcessGroup (Unix)
│   │   └── kill_windows.go     # KillProcessGroup (Windows)
│   └── yamlutil/               # YAML wrapper with limits
│
├── examples/                   # Example markdown files and generated PDFs
│
└── docs/                       # Documentation
```

## Root Configuration Files

```
picoloom/
├── go.mod                      # Module definition, dependencies
├── go.sum                      # Dependency checksums
├── Makefile                    # Build, test, lint commands
├── Dockerfile                  # Container build
├── README.md                   # User documentation
├── CONTRIBUTING.md             # Contributor guide
├── SECURITY.md                 # Security policy
├── CODE_OF_CONDUCT.md          # Community guidelines
└── LICENSE                 # BSD-3-Clause license
```

## Conventions

- **Library at root** - `import "github.com/alnah/picoloom/v2"`
- **Public API only at root** - Converter, Input, types, errors
- **Pipeline in internal/** - mdtransform, md2html, htmlinject
- **Platform suffix** - `_unix.go`, `_windows.go` for OS-specific code
- **internal/** - Private implementation (pipeline, assets, config, utilities)
- **cmd/** - Binaries

## Test Conventions

| Pattern                     | Purpose                              | Example                        |
| --------------------------- | ------------------------------------ | ------------------------------ |
| `*_test.go`                 | Unit tests (same package)            | `converter_test.go`            |
| `*_integration_test.go`     | Integration tests (require browser)  | `converter_integration_test.go`|
| `*_bench_test.go`           | Benchmarks                           | `pool_bench_test.go`           |
| `example_test.go`           | Runnable examples for godoc          | `example_test.go`              |

- Unit tests: `make test` - fast, no external dependencies
- Integration tests: `make test-integration` - require Chrome, use `-tags=integration`
- Benchmarks: `make bench` - use `-tags=bench`
- Examples: `go test -run Example` - appear on pkg.go.dev

## Embedded Styles

| Style          | Target Use Case                                |
| -------------- | ---------------------------------------------- |
| `default`      | Minimal, neutral baseline                      |
| `technical`    | System fonts, GitHub syntax highlighting       |
| `creative`     | Colorful headings, visual flair                |
| `academic`     | Serif fonts, academic formatting               |
| `corporate`    | Arial/Helvetica, blue accents, business style  |
| `legal`        | Times New Roman, double spacing                |
| `invoice`      | Optimized tables, minimal cover                |
| `manuscript`   | Courier New mono, scene breaks                 |
