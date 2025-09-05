# -------- settings (override via env or CLI: make CMD_PATH=cmd/other) --------
APP_NAME      ?= cdc-gateway
MODULE        ?= github.com/the-yorkshire-allen/cdc-gateway
CMD_PATH      ?= cmd/cdc-gateway
OUT_DIR       ?= out
BIN           := $(OUT_DIR)/$(APP_NAME)

GO            ?= go
GOFLAGS       ?=
GOOS          ?= $(shell $(GO) env GOOS)
GOARCH        ?= $(shell $(GO) env GOARCH)

# Version metadata (used in ldflags)
VERSION       ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo 0.0.0)
COMMIT        ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
DATE          ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags (adjust variable names if your main package differs)
LDFLAGS       ?= -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.date=$(DATE)'

# Docker/Compose
REGISTRY      ?=
IMAGE_NAME    ?= $(APP_NAME)
TAG           ?= $(VERSION)
IMAGE_REF     := $(if $(REGISTRY),$(REGISTRY)/,)$(IMAGE_NAME):$(TAG)
COMPOSE_FILE  ?= deploy/docker-compose.yml

SHELL         := bash
.SHELLFLAGS   := -eu -o pipefail -c

.PHONY: all build clean tidy fmt test vet lint run docker-build docker-push up down logs restart

# Default: build binary with ldflags
all: build

build:
	@mkdir -p "$(OUT_DIR)"
	@echo ">> building $(BIN) (version=$(VERSION) commit=$(COMMIT))"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o "$(BIN)" ./$(CMD_PATH)

run: build
	@echo ">> running $(BIN)"
	@"$(BIN)"

clean:
	@echo ">> clean"
	@rm -rf "$(OUT_DIR)"

tidy:
	@echo ">> go mod tidy"
	@$(GO) mod tidy

fmt:
	@echo ">> format (gofumpt if present, else gofmt)"
	@if command -v gofumpt >/dev/null 2>&1; then gofumpt -l -w .; else gofmt -s -w .; fi

vet:
	@echo ">> vet"
	@$(GO) vet ./...

test:
	@echo ">> test (-race)"
	@$(GO) test -race ./...

lint:
	@echo ">> lint (golangci-lint if present)"
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed (skipping)"; fi

# ----- Docker image using deploy/Dockerfile -----
docker-build:
	@echo ">> docker build $(IMAGE_REF)"
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg DATE="$(DATE)" \
		-t "$(IMAGE_REF)" \
		-f deploy/Dockerfile .

docker-push: docker-build
	@echo ">> docker push $(IMAGE_REF)"
	docker push "$(IMAGE_REF)"

# ----- Docker Compose using deploy/docker-compose.yml -----
up:
	@echo ">> compose up"
	docker compose -f "$(COMPOSE_FILE)" up -d

down:
	@echo ">> compose down"
	docker compose -f "$(COMPOSE_FILE)" down

logs:
	@echo ">> compose logs (follow)"
	docker compose -f "$(COMPOSE_FILE)" logs -f

restart:
	@echo ">> compose restart"
	docker compose -f "$(COMPOSE_FILE)" restart
