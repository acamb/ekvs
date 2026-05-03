BINARY_DIR := bin
CMDS       := server tui cli
VERSION := $(shell cat version)

.PHONY: build test integration-test integration-test-passphrase integration-test-down lint clean

## build: compile all binaries into ./bin/
build:
	@mkdir -p $(BINARY_DIR)
	@for cmd in $(CMDS); do \
		echo "Building $$cmd..."; \
		GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/$$cmd ./cmd/$$cmd; \
	done

build-tui-win:
	@mkdir -p $(BINARY_DIR)
	@echo "Building tui for Windows...";
	@GOOS=windows GOARCH=amd64 go build -o $(BINARY_DIR)/tui.exe ./cmd/tui;

build-static:
	@mkdir -p $(BINARY_DIR)
	@for cmd in server cli; do \
		echo "Building static $$cmd..."; \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/$$cmd-static ./cmd/$$cmd; \
	done

release: clean build build-tui-win build-static

docker-release: clean build-static
	docker build -t ekvs:latest -f Dockerfile .

docker-publish: docker-release
	docker tag ekvs:latest acamb23/ekvs:latest
	docker push acamb23/ekvs:latest
	docker tag ekvs:latest acamb23/ekvs:$(VERSION)
	docker push acamb23/ekvs:$(VERSION)

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

