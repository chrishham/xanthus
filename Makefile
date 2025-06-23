.PHONY: dev build test lint css css-watch clean

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
	rm -rf web/static/js/vendor/
