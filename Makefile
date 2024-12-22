.PHONY: test cover build

BUILD_DIR := build
TARGETS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
SERVER_SRC := ./cmd/server/server.go
CLIENT_SRC := ./cmd/client/client.go

test:
	go test -race -count 1 ./...

cover:
	go test -short -race -count 1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	rm -f coverage.out

define build_target
	GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(3)_$(1)_$(2)$(if $(findstring windows,$(1)),.exe) $(4)
endef

build:
	mkdir -p $(BUILD_DIR)
	rm -rf $(BUILD_DIR)/*

	@echo "Начинаем сборку сервера..."
	$(foreach target, $(TARGETS), \
		$(call build_target,$(word 1,$(subst /, ,$(target))),$(word 2,$(subst /, ,$(target))),stream-recorder_server,$(SERVER_SRC);))

	@echo "Начинаем сборку клиента..."
	$(foreach target, $(TARGETS), \
		$(call build_target,$(word 1,$(subst /, ,$(target))),$(word 2,$(subst /, ,$(target))),stream-recorder_client,$(CLIENT_SRC);))