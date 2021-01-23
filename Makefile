all: lint test

.PHONY: lint
lint:
		golangci-lint run ./...

.PHONY: test
test:
		go test -coverprofile=cover.out ./...

.PHONY: cover
cover:
		go tool cover -html cover.out
