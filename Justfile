export CGO_ENABLED := "0"

# List available recipes
default:
    @just --list

# Enter the Nix dev environment (requires devenv)
dev:
    devenv shell

# Build CLI binaries into ./bin/
build:
    mkdir -p bin
    go build -o bin/nacmail ./cmd/nacmail
    go build -o bin/nacclean ./cmd/nacclean
    go build -o bin/nacls ./cmd/nacls
    go build -o bin/nacwrite ./cmd/nacwrite

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run golangci-lint
lint:
    golangci-lint run ./...

# Remove built artifacts
clean:
    rm -rf bin/ dist/

# Build release artifacts locally (requires goreleaser)
release-snapshot:
    goreleaser release --snapshot --clean

# Format all Go source
fmt:
    gofmt -w .

# Tidy go.mod
tidy:
    go mod tidy
