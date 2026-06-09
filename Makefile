# Makefile
BINARY  := bin/galactica
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION ?= dev
LDFLAGS := -s -w \
  -X main.Version=$(VERSION) \
  -X main.Commit=$(COMMIT) \
  -X main.Date=$(DATE)

.PHONY: build ui-build ui-watch dev test lint docker

build: ui-build
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/galactica

ui-build:
	cd web && npm ci --silent && npm run build
	mkdir -p internal/api/swagger-ui
	cp web/node_modules/swagger-ui-dist/swagger-ui-bundle.js \
	   web/node_modules/swagger-ui-dist/swagger-ui.css \
	   web/node_modules/swagger-ui-dist/swagger-ui.css.map \
	   internal/api/swagger-ui/

ui-watch:
	cd web && npm run dev

dev:
	air

test:
	go test ./...

lint:
	golangci-lint run ./...

docker:
	docker build -t techblog/galactica:dev .
