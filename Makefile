.PHONY: build test clean help

BINARY_NAME=serv
BIN_DIR=bin
CMD_DIR=cmd/serv

ifeq ($(OS),Windows_NT)
	BINARY_PATH=$(BIN_DIR)\$(BINARY_NAME).exe
	MKDIR=if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	RM=if exist $(BIN_DIR) rmdir /s /q $(BIN_DIR)
else
	BINARY_PATH=$(BIN_DIR)/$(BINARY_NAME)
	MKDIR=mkdir -p $(BIN_DIR)
	RM=rm -rf $(BIN_DIR)
endif

help:
	@echo "Available targets:"
	@echo "  make build   - Build the CLI binary"
	@echo "  make test    - Run tests"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make help    - Show this help message"

build:
	@$(MKDIR)
	@go build -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Built $(BINARY_PATH)"

test:
	@go test -v ./...

clean:
	@$(RM)
	@go clean
	@echo "Cleaned build artifacts"
