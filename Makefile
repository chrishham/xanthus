.PHONY: dev build test lint css css-watch clean

# Development mode
dev: css
	go run cmd/xanthus/main.go

# Build for production
build: css
	go build -o bin/xanthus cmd/xanthus/main.go

# Build CSS
css:
	npm run build-css-prod

# Watch CSS changes during development
css-watch:
	npm run build-css

# Run tests
test:
	go test ./...

# Lint code
lint:
	go fmt ./...
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f web/static/css/output.css