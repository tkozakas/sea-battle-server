BINARY_NAME=sea-battle-server
BUILD_DIR=bin

.PHONY: run test cover lint build docker env

env:
	@test -f .env || cp env.template .env

run:
	go run ./cmd/server/...

test:
	go test ./... -v -race -count=1

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/...

docker:
	docker build -t $(BINARY_NAME):latest .
