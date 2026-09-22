# Makefile — build & release for hwcloud (openagent-go).
#
# Default identity: NAME=hwcloud, VERSION from git describe.
#
#   make build          # host-only quick build (single binary)
#   make release        # refresh tiktoken data → cross-compile 6 platforms → sha256sum
#   make tiktoken-data  # download latest BPE encoder files
#   make sha256         # generate SHA256SUMS in dist/
#   make clean

# ---- Build identity ----
NAME        := hwcloud
VERSION     ?= v0.0.1-alpha.27
MODULE      := github.com/yusheng-g/openagent-go
LDFLAGS     := -s -w -X $(MODULE)/version.Name=$(NAME) -X $(MODULE)/version.Version=$(VERSION)

# ---- Toolchain ----
CGO         := 0

# ---- Layout ----
BIN_DIR     := dist
CLI_MAIN    := ./cmd/cli/
TARGETS     := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64

# ---- Tiktoken BPE data ----
TIKTOKEN_DIR   := tokenizer/data
TIKTOKEN_BASE  := https://openaipublic.blob.core.windows.net/encodings
TIKTOKEN_FILES := cl100k_base o200k_base

# ---- Targets ----
.PHONY: build release tiktoken-data sha256 clean vet test

build:
	@echo ""
	@echo "==> build: $(NAME) $(VERSION)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(NAME) \
		$(CLI_MAIN)
	@echo "    -> $(BIN_DIR)/$(NAME)"

tiktoken-data:
	@echo ""
	@echo "==> tiktoken-data: downloading BPE encoder files"
	@mkdir -p $(TIKTOKEN_DIR)
	@set -e; \
	for f in $(TIKTOKEN_FILES); do \
		echo "  $$f.tiktoken"; \
		curl -fsSL -o $(TIKTOKEN_DIR)/$$f.tiktoken $(TIKTOKEN_BASE)/$$f.tiktoken; \
	done

release: tiktoken-data
	@echo ""
	@echo "==> release: $(NAME) $(VERSION) — 6 platforms"
	@mkdir -p $(BIN_DIR)
	@set -e; \
	for tgt in $(TARGETS); do \
		gos=$${tgt%%-*}; \
		garch=$${tgt##*-}; \
		ext=""; \
		if [ $$gos = windows ]; then ext=".exe"; fi; \
		out=$(BIN_DIR)/$(NAME)-$$gos-$$garch$$ext; \
		echo "  CC $$out"; \
		GOOS=$$gos GOARCH=$$garch CGO_ENABLED=$(CGO) \
			go build -ldflags "$(LDFLAGS)" -o $$out $(CLI_MAIN); \
	done
	@$(MAKE) --no-print-directory sha256
	@echo ""
	@echo "Built:"
	@for f in $(BIN_DIR)/$(NAME)-*; do echo "  $$f"; done
	@echo "  $(BIN_DIR)/SHA256SUMS"

sha256:
	@cd $(BIN_DIR) && sha256sum $(NAME)-* > SHA256SUMS
	@echo "    -> $(BIN_DIR)/SHA256SUMS"

clean:
	@rm -rf $(BIN_DIR)
	@rm -f $(NAME)
	@echo "cleaned $(BIN_DIR)/ and ./$(NAME)"

vet:
	@echo ""
	@echo "==> go vet ./..."
	@go vet ./...

test:
	@echo ""
	@echo "==> go test ./..."
	@go test ./...
