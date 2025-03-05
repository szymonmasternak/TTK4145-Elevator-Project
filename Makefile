ELEVATOR_BINARY_NAME = elevator
ELEVATOR_MAIN_DIR = cmd/$(ELEVATOR_BINARY_NAME)/main.go

ELEVATORTESTER_BINARY_NAME = elevatortester
ELEVATORTESTER_MAIN_DIR = cmd/$(ELEVATORTESTER_BINARY_NAME)/main.go

BUILD_DIR = build

.PHONY: build run test all
default: build run

all: build test run

build:
	@echo "[BUILD] Building projects"
	@mkdir -p $(BUILD_DIR)
	go mod tidy
	go generate ./...
	@echo "[BUILD] Building $(ELEVATOR_BINARY_NAME)"
	go build -o $(BUILD_DIR)/$(ELEVATOR_BINARY_NAME) $(ELEVATOR_MAIN_DIR)
	@echo "[BUILD] Building $(ELEVATORTESTER_BINARY_NAME)"
	go build -o $(BUILD_DIR)/$(ELEVATORTESTER_BINARY_NAME) $(ELEVATORTESTER_MAIN_DIR)

run:
	@echo "[RUN] Running $(ELEVATOR_MAIN_DIR)/$(ELEVATOR_BINARY_NAME)"
	@$(BUILD_DIR)/$(ELEVATOR_BINARY_NAME)

test:
	@echo "[TEST] Running Tests"
	@go generate ./...
	@go test ./... -timeout 60s
