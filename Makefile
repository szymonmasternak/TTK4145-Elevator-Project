BINARY_NAME = elevator
BUILD_DIR = build
MAIN_DIR = cmd/$(BINARY_NAME)/main.go

.PHONY: build run test all
default: build run

all: build test run

build:
	@echo "[BUILD] Building project"
	@mkdir -p $(BUILD_DIR)
	go mod tidy
	go generate ./...
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_DIR)

run:
	@echo "[RUN] Running $(BUILD_DIR)/$(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "[TEST] Running Tests"
	@go test ./...
