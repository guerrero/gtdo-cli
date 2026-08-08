BINARY  := gtdo
PKG     := github.com/guerrero/gtdo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG)/internal/cli.Version=$(VERSION) \
	-X $(PKG)/internal/cli.Commit=$(COMMIT) \
	-X $(PKG)/internal/cli.Date=$(DATE)

export CGO_ENABLED := 0

.PHONY: build test lint man install release release-dry completions clean

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/gtdo

test:
	go test ./...

lint:
	golangci-lint run

man:
	go run ./tools/genman

# Shell completion scripts for the release archives (goreleaser's
# before.hooks run this; the archive files list picks up completions/*).
completions:
	mkdir -p completions
	go run ./cmd/gtdo completion bash > completions/gtdo.bash
	go run ./cmd/gtdo completion fish > completions/gtdo.fish

install:
	go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/gtdo

release-dry:
	goreleaser release --snapshot --clean --skip=publish

release:
	@tag=$$(git tag --points-at HEAD); \
	if [ -z "$$tag" ]; then \
		echo "error: HEAD has no tag; tag a release first (see CONTRIBUTING.md)" >&2; \
		exit 1; \
	fi; \
	echo "releasing $$tag"; \
	goreleaser release --clean

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/ completions/
