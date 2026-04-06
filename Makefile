.PHONY: serve build wire generate test test-integration lint clean migrate-up migrate-down

# Development
serve:
	air

build:
	go build -o ./tmp/main ./app/main/

# Code generation
wire:
	wire ./app/di/

generate:
	go generate ./...

# Testing
test:
	go test ./... -short -v

test-integration:
	go test ./... -run Integration -v

# Linting
lint:
	golangci-lint run

# Database
migrate-up:
	go run ./app/main migrate up

migrate-down:
	go run ./app/main migrate down

# Cleanup
clean:
	rm -rf ./tmp
	go clean
