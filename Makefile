BINARY_DIR := bin
CMDS       := server tui cli

.PHONY: build test integration-test integration-test-passphrase integration-test-down lint clean

## build: compile all binaries into ./bin/
build:
	@mkdir -p $(BINARY_DIR)
	@for cmd in $(CMDS); do \
		echo "Building $$cmd..."; \
		go build -o $(BINARY_DIR)/$$cmd ./cmd/$$cmd; \
	done

build-tui-win:
	@mkdir -p $(BINARY_DIR)
	echo "Building tui for Windows..."; \
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_DIR)/tui.exe ./cmd/tui;
## test: run all unit tests with race detector
test:
	go test ./... -race -count=1

## integration-test: start nopass scenario in background (server + cli + tui, no passphrase)
integration-test:
	cd tests/integration && docker compose -f docker-compose.nopass.yml up --build -d

## integration-test-passphrase: start passphrase scenario in background (server + cli + tui, passphrase=changeme)
integration-test-passphrase:
	cd tests/integration && docker compose -f docker-compose.passphrase.yml up --build -d

## integration-test-down: stop and remove all integration containers
integration-test-down:
	cd tests/integration && docker compose -f docker-compose.nopass.yml down 2>nul & docker compose -f docker-compose.passphrase.yml down 2>nul & exit 0

## lint: run static analysis
lint:
	go vet ./...

## clean: remove build artefacts
clean:
	rm -rf $(BINARY_DIR)/ coverage.out

