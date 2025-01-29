BINARY_NAME = elevator
BUILD_DIR = build
MAIN_DIR = cmd/$(BINARY_NAME)/main.go

.PHONY: all build run clean

all: build run

build:
	@echo "Building project"
	@mkdir -p $(BUILD_DIR)
	go mod tidy
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_DIR)

run:
	@echo "Running $(BUILD_DIR)/$(BINARY_NAME)"
	@$(BUILD_DIR)/$(BINARY_NAME)