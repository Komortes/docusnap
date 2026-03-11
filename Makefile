BINARY ?= docusnap
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo dev-$(COMMIT))
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(DATE)

.PHONY: print-version
print-version:
	@echo $(VERSION)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/docusnap

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/docusnap

.PHONY: test
test:
	go test ./...
