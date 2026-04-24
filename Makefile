BINARY_DIR := bin
CMDS       := server tui cli

.PHONY: build test integration-test lint clean

## build: compile all binaries into ./bin/
build:
	@mkdir -p $(BINARY_DIR)
	@for cmd in $(CMDS); do \
		echo "Building $$cmd..."; \
		go build -o $(BINARY_DIR)/$$cmd ./cmd/$$cmd; \
	done

## test: run all unit tests with race detector
test:
	go test ./... -race -count=1

## integration-test: run semi-manual integration tests via Docker Compose
integration-test:
	cd tests/integration && docker compose up --abort-on-container-exit

## lint: run static analysis
lint:
	go vet ./...

## clean: remove build artefacts
clean:
	rm -rf $(BINARY_DIR)/ coverage.out

