.PHONY: tidy build test vet run docker

GO ?= go
BIN ?= bin/server

tidy:
	$(GO) mod tidy

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -race -count=1

build:
	$(GO) build -trimpath -o $(BIN) ./cmd/server

run: build
	./$(BIN)

docker:
	docker build -t ai-proxy-service:dev -f dockerfile .
