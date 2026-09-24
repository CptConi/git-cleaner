# Build helpers for macOS and Linux (on Windows, use the "go" commands
# documented in README.md).

BINARY    := git-cleaner
# Version shown by --version: derived from the latest git tag (v1.2.3 gives
# 1.2.3). Outside a git checkout it stays empty and the binary reports "dev".
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS   := -s -w -X main.version=$(VERSION)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

# CGO_ENABLED=0 produces fully static binaries that run on any machine of
# the target OS, without any system library requirement.
export CGO_ENABLED := 0

.PHONY: build install test vet release clean

## build: compile git-cleaner for the current machine
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: install git-cleaner into $(go env GOPATH)/bin
install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

## test: run the test suite (requires git)
test:
	go test ./...

## vet: static checks for every target OS
vet:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} go vet ./... || exit 1; \
	done

## release: cross-compile every platform into dist/
release:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; ext=""; \
		if [ "$$os" = windows ]; then ext=.exe; fi; \
		out=dist/$(BINARY)-$$os-$$arch$$ext; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
	done

## clean: remove build outputs
clean:
	rm -rf dist $(BINARY) $(BINARY).exe
