BINARY := md2pdf

.PHONY: help build test test-integration test-cover test-cover-all bench bench-cpu bench-mem run clean fmt vet lint sec check check-all tools examples

.DEFAULT_GOAL := help

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

tools: ## Install development tools (staticcheck, gosec; golangci-lint separately)
	go get -tool honnef.co/go/tools/cmd/staticcheck
	go get -tool github.com/securego/gosec/v2/cmd/gosec
	@echo "Install golangci-lint separately: https://golangci-lint.run/welcome/install/"

deps: ## Download dependencies from go.mod
	go mod download

build: ## Build the binary
	go build -o $(BINARY) ./cmd/md2pdf

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
	rm -f $(BINARY) coverage.out coverage.html cpu.prof mem.prof

fmt: ## Format source code
	go fmt ./...

vet: ## Run go vet for static analysis
	go vet ./...

lint: ## Run golangci-lint
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
	./$(BINARY) convert -c examples/full-featured.yaml examples/full-featured.md -o examples/full-featured.pdf
	@echo "Done. Review with 'git diff examples/' and commit if needed."
