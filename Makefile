CLI_APP_NAME=inmemory-db

.PHONY: build
build: build-cli

.PHONY: build-cli
build-cli:
	go build -o ${CLI_APP_NAME}-cli cmd/cli/main.go

.PHONY: run-cli
run-cli:
	go run cmd/cli/main.go

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -v -coverprofile=coverage.out ./...
