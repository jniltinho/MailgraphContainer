## Variables for UPX
UPX_VERSION := 5.1.1
UPX_ARCHIVE  := upx-$(UPX_VERSION)-amd64_linux.tar.xz
UPX_DIR      := upx-$(UPX_VERSION)-amd64_linux
UPX_BIN      := /usr/local/bin/upx
UPX_URL      := https://github.com/upx/upx/releases/download/v$(UPX_VERSION)/$(UPX_ARCHIVE)

## Variables for Go application
APP        := mailgraph
BIN        := bin/$(APP)
PKG        := mailgraph/internal/buildinfo
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE         ?= davidullrich/mailgraph:latest
TEST_IMAGE    ?= mailgraph:test
TEST_PORT     ?= 8585
TESTDATA_HOST ?= mx02
TESTDATA_LOG  ?= /var/log/mail.log
COMPOSE_TEST  := docker-compose -f docker-compose.test.yml

LDFLAGS := -trimpath -ldflags "-s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).BuildDate=$(BUILD_TIME) \
	-X $(PKG).GitCommit=$(GIT_COMMIT)"

.PHONY: all build build-prod run clean deps tidy test install-upx build-docker \
	fetch-testdata test-docker-build test-docker test-docker-down test-docker-validate certs help

all: clean build

build: clean
	@echo "Building $(APP)..."
	CGO_ENABLED=0 go build -o $(BIN) $(LDFLAGS) .

build-prod: clean
	@echo "Building $(APP) (UPX compressed)..."
	CGO_ENABLED=0 go build -o $(BIN) $(LDFLAGS) .
	upx --best --lzma $(BIN)

run: build
	@echo "Starting $(APP)..."
	./$(BIN) server \
		--logfile=/var/log/mail/mail.log \
		--daemon-rrd=./rrd \
		--listen=:8080

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning up..."
	rm -f $(BIN)
	rm -rf $(UPX_DIR) $(UPX_ARCHIVE)

tidy:
	@echo "Tidying go modules..."
	go mod tidy

deps:
	@echo "Installing dependencies..."
	go mod download

install-upx:
	@echo "Installing UPX binary..."
	curl -ksSL "$(UPX_URL)" -o "$(UPX_ARCHIVE)"
	tar -xf "$(UPX_ARCHIVE)"
	chmod +x "$(UPX_DIR)/upx"
	sudo mv "$(UPX_DIR)/upx" "$(UPX_BIN)"
	rm -rf "$(UPX_DIR)" "$(UPX_ARCHIVE)"

build-docker:
	@echo "Building Docker image $(IMAGE)..."
	docker build --no-cache --progress=plain \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(IMAGE) .

fetch-testdata: testdata/mail.log

testdata/mail.log:
	@echo "Fetching $(TESTDATA_LOG) from $(TESTDATA_HOST)..."
	@mkdir -p testdata/rrd
	scp -C $(TESTDATA_HOST):$(TESTDATA_LOG) $@
	@ls -lh $@

test-docker-build:
	@echo "Building test image $(TEST_IMAGE)..."
	docker build --progress=plain \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(TEST_IMAGE) .

test-docker: testdata/mail.log test-docker-build
	@echo "Starting test container on port $(TEST_PORT)..."
	$(COMPOSE_TEST) up -d
	@echo "Waiting for mailgraph to import log and start HTTP server..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do \
		if curl -sf http://127.0.0.1:$(TEST_PORT)/ >/dev/null 2>&1; then \
			echo "Mailgraph is up at http://127.0.0.1:$(TEST_PORT)/"; \
			exit 0; \
		fi; \
		sleep 5; \
	done; \
	echo "Timeout waiting for mailgraph on port $(TEST_PORT)"; \
	$(COMPOSE_TEST) logs --tail=50; \
	exit 1

test-docker-down:
	@echo "Stopping test container..."
	$(COMPOSE_TEST) down

test-docker-validate:
	@echo "Validating test container..."
	@curl -sf -o /dev/null -w "HTTP %{http_code}\n" http://127.0.0.1:$(TEST_PORT)/
	@curl -sf "http://127.0.0.1:$(TEST_PORT)/chart?period=0&type=n" | head -c 200
	@echo ""
	@ls -la testdata/rrd/
	@$(COMPOSE_TEST) ps

certs:
	@echo "Generating SSL certificates..."
	mkdir -p ssl
	openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
		-keyout ssl/server.key -out ssl/server.crt \
		-subj "/C=BR/ST=SP/L=Sao Paulo/O=Development/CN=localhost"

help:
	@echo "Makefile commands:"
	@echo "  build         - Build Go binary (trimpath + ldflags)"
	@echo "  build-prod    - Build + UPX compression (for releases)"
	@echo "  build-docker  - Multi-stage Docker image ($(IMAGE))"
	@echo "  run           - Build and start mailgraph locally"
	@echo "  test          - Run go test ./..."
	@echo "  clean         - Remove bin/$(APP)"
	@echo "  deps / tidy   - Go module download / cleanup"
	@echo "  install-upx   - Install UPX compressor"
	@echo "  certs            - Generate self-signed TLS certs in ssl/"
	@echo "  fetch-testdata   - Download mail.log from $(TESTDATA_HOST)"
	@echo "  test-docker-build - Build $(TEST_IMAGE) image only"
	@echo "  test-docker      - Build, run test container on :$(TEST_PORT), validate"
	@echo "  test-docker-down - Stop test container"
	@echo "  test-docker-validate - Check HTTP and RRD output"
	@echo ""
	@echo "Examples:"
	@echo "  make build-prod"
	@echo "  make build-docker IMAGE=jniltinho/mailgraph:latest"
	@echo "  make test-docker TESTDATA_HOST=mx02"
	@echo "  make certs"