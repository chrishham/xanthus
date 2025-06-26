.PHONY: dev build test test-unit test-integration test-coverage test-all lint css css-watch clean

# Development mode
dev: css
	go run cmd/xanthus/main.go

# Build for production
build: assets
	go build -o bin/xanthus cmd/xanthus/main.go

# Build all assets (CSS + JS)
assets:
	npm run build-assets

# Build CSS only
css:
	npm run build-css-prod

# Watch CSS changes during development
css-watch:
	npm run build-css

# Run unit tests only
test-unit:
	go test -v ./tests/unit/...

# Run integration tests (when they exist)
test-integration:
	go test -v ./tests/integration/...

# Run all structured tests
test:
	go test -v ./tests/...

# Run tests with coverage
test-coverage:
	go test -v ./tests/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run all tests including any legacy tests
test-all:
	go test -v ./...

# Lint code
lint:
	go fmt ./...
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f web/static/css/output.css
	rm -rf web/static/js/vendor/
	rm -f coverage.out coverage.html
