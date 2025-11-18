# Test and lint commands

# Set PATH to include Go bin directories
export PATH := env_var_or_default("HOME", "") + "/go/bin:" + env_var_or_default("GOPATH", "") + "/bin:" + env_var_or_default("PATH", "")

# Install dependencies
deps:
    #!/usr/bin/env bash
    set -e
    if ! command -v staticcheck &> /dev/null; then
        echo "Installing staticcheck..."
        go install honnef.co/go/tools/cmd/staticcheck@latest
    fi
    if ! command -v goimports &> /dev/null; then
        echo "Installing goimports..."
        go install golang.org/x/tools/cmd/goimports@latest
    fi

check_deps:
    #!/usr/bin/env bash
    set -e
    if ! command -v staticcheck &> /dev/null; then
        echo "staticcheck is not installed"
        exit 1
    fi
    if ! command -v goimports &> /dev/null; then
        echo "goimports is not installed"
        exit 1
    fi

# Run all tests
test:
    go test ./...

# Run staticcheck linter
lint: deps
    staticcheck ./...

# Format code using gofancyimports
fmt:
    #!/usr/bin/env bash
    set -e
    # Find all directories with .go files, excluding testdata
    find . \
      -type f \
      -name "*.go" \
      -not -path "./testdata/*" \
      -not -path "./.git/*" \
      -not -path "./dist/*" \
      -not -path "./internal/stdlib/go_x_stdlib/*" \
    | xargs -n1 dirname \
    | sort -u \
    | xargs go run ./cmd/gofancyimports fix -w --local github.com/NonLogicalDev/gofancyimports

# Verify go.mod and go.sum are tidy
verify:
    go mod tidy
    git diff --exit-code go.mod go.sum || (echo "::error::go.mod or go.sum changed after running 'go mod tidy'. Please run 'go mod tidy' and commit the changes." && exit 1)

# Run all CI checks (verify, lint, test)
ci: deps verify lint test
    @echo "All checks passed!"
