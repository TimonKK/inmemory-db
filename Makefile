CLI_APP_NAME=inmemory-db

.PHONY: build
build: build-server build-cli

.PHONY: build-cli
build-cli:
	go build -o tmp/${CLI_APP_NAME}-cli cmd/cli/main.go

.PHONY: build-server
build-server:
	go build -o tmp/${CLI_APP_NAME}-cli cmd/server/main.go

.PHONY: run-cli
run-cli:
	go run cmd/cli/main.go

.PHONY: run-server
run-server:
	go run cmd/server/main.go

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out ./...
