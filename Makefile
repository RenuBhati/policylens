.PHONY: setup ui engine test vet build run eval share

export GOCACHE := $(CURDIR)/.cache/go-build
export GOPATH := $(CURDIR)/.cache/go-path
export GOMODCACHE := $(CURDIR)/.cache/go-mod
export npm_config_cache := $(CURDIR)/.cache/npm

PREVIEW_ORIGIN ?= http://127.0.0.1:8080

setup: engine ui

engine:
	./scripts/install-kyverno.sh

ui:
	cd frontend && npm ci --no-fund && npm run build

test:
	go test -race ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/policylens-server ./cmd/server
	go build -trimpath -o bin/policylens ./cmd/policylens

run:
	go run ./cmd/server

eval:
	go run ./cmd/eval

# Keep the application running in another terminal before sharing.
share:
	cf tunnels quick-start "$(PREVIEW_ORIGIN)"
