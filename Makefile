.PHONY: build-web build dev-web dev-api test clean

RIFT_BINARY ?= rift

build-web:
	cd web && npm run build
	rm -rf internal/embed/ui/*
	cp -R web/dist/. internal/embed/ui/
	touch internal/embed/ui/.gitkeep

build: build-web
	go build -o $(RIFT_BINARY) ./cmd/rift

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
