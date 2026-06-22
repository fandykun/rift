.PHONY: build-web build dev-web dev-api test clean

RIFT_BINARY ?= rift
VERSION ?= 0.1.0-dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GO_LDFLAGS := -X main.version=$(VERSION) -X main.buildCommit=$(BUILD_COMMIT)

build-web:
	cd web && npm run build
	rm -rf internal/embed/ui/*
	cp -R web/dist/. internal/embed/ui/
	touch internal/embed/ui/.gitkeep

build: build-web
	go build -ldflags "$(GO_LDFLAGS)" -o $(RIFT_BINARY) ./cmd/rift

dev-web:
	cd web && npm run dev

dev-api:
	go run ./cmd/rift server

test:
	go test ./...
	cd web && npm run build

clean:
	rm -f $(RIFT_BINARY)
	rm -rf web/dist internal/embed/ui/*
	touch internal/embed/ui/.gitkeep
