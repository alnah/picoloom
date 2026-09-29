BINARY := picoloom
LEGACY_BINARY := md2pdf

.PHONY: help build build-legacy test test-integration test-cover test-cover-all bench bench-cpu bench-mem run clean fmt vet lint sec check check-all tools examples

.DEFAULT_GOAL := help

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

tools: ## Install development tools (gosec; golangci-lint separately)
	go get -tool github.com/securego/gosec/v2/cmd/gosec
	@echo "Install golangci-lint separately: https://golangci-lint.run/welcome/install/"

deps: ## Download dependencies from go.mod
	go mod download

build: ## Build the binary
	go build -o $(BINARY) ./cmd/picoloom

build-legacy: ## Build the legacy md2pdf alias binary
	go build -o $(LEGACY_BINARY) ./cmd/picoloom

test: ## Run unit tests
	go test -v ./...

test-integration: ## Run integration tests (rod auto-downloads chromium)
	go test -v -tags=integration ./...

test-cover: ## Run unit tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-cover-all: ## Run all tests with coverage report
	go test -v -tags=integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench: ## Run benchmarks
	go test -tags=bench -bench=. -benchmem ./...

bench-cpu: ## Run benchmarks with CPU profiling
	go test -tags=bench -bench=. -benchmem -cpuprofile=cpu.prof ./...
	@echo "Run 'go tool pprof cpu.prof' to analyze"

bench-mem: ## Run benchmarks with memory profiling
	go test -tags=bench -bench=. -benchmem -memprofile=mem.prof ./...
	@echo "Run 'go tool pprof mem.prof' to analyze"

bench-compare: ## Compare benchmarks (usage: make bench-compare OLD=old.txt NEW=new.txt)
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Usage: make bench-compare OLD=old.txt NEW=new.txt"; \
		exit 1; \
	fi
	go run golang.org/x/perf/cmd/benchstat@latest $(OLD) $(NEW)

run: build ## Build and run the shell
	./$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY) $(LEGACY_BINARY) coverage.out coverage.html cpu.prof mem.prof

fmt: ## Format source code
	go fmt ./...

vet: ## Run go vet for static analysis
	go vet ./...

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found in PATH"; echo "Install: https://golangci-lint.run/welcome/install/"; exit 1)
	@build_go=$$(golangci-lint version | sed -nE 's/.*built with go([0-9]+\.[0-9]+).*/\1/p'); \
	runtime_go=$$(go env GOVERSION | sed -nE 's/^go([0-9]+\.[0-9]+).*/\1/p'); \
	if [ -n "$$build_go" ] && [ -n "$$runtime_go" ]; then \
		awk -v b="$$build_go" -v r="$$runtime_go" 'BEGIN {split(b,bp,"."); split(r,rp,"."); if (bp[1] < rp[1] || (bp[1] == rp[1] && bp[2] < rp[2])) exit 1; exit 0}' || \
		(echo "golangci-lint was built with go$$build_go but current toolchain is go$$runtime_go."; \
		 echo "Reinstall golangci-lint from an official binary release or rebuild it with current Go."; \
		 exit 1); \
	fi
	golangci-lint run

sec: ## Run gosec security scanner
	go tool gosec ./...

check: fmt vet lint sec test ## Run all checks (unit tests only)

check-all: fmt vet lint sec test-integration ## Run all checks including integration tests

examples: build ## Regenerate example PDFs in examples/
	./$(BINARY) convert examples/simple-report.md -o examples/simple-default.pdf
	./$(BINARY) convert examples/simple-report.md --style technical -o examples/simple-technical.pdf
	./$(BINARY) convert examples/simple-report.md --style academic -o examples/simple-academic.pdf
	./$(BINARY) convert examples/simple-report.md --style corporate -o examples/simple-corporate.pdf
	./$(BINARY) convert examples/simple-report.md --style creative -o examples/simple-creative.pdf
	./$(BINARY) convert examples/simple-report.md --style invoice -o examples/simple-invoice.pdf
	./$(BINARY) convert examples/simple-report.md --style legal -o examples/simple-legal.pdf
	./$(BINARY) convert examples/simple-report.md --style manuscript -o examples/simple-manuscript.pdf
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-academic.pdf --style academic
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-corporate.pdf --style corporate
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-creative.pdf --style creative
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-default.pdf --style default
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-invoice.pdf --style invoice
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-legal.pdf --style legal
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-manuscript.pdf --style manuscript
	./$(BINARY) convert examples/full-featured.md -c examples/full-featured.yaml -o examples/full-technical.pdf --style technical
	@echo "Done. Review with 'git diff examples/' and commit if needed."
