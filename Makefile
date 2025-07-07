.PHONY: dev dev-full build-svelte build test test-unit test-integration test-e2e test-e2e-live test-e2e-coverage test-e2e-vps test-e2e-ssl test-e2e-apps test-e2e-ui test-e2e-perf test-e2e-security test-e2e-dr test-coverage test-all test-everything lint css css-watch clean help-testing docker-build docker-push docker-tag docker-multi help-docker release re-release

# Development mode
dev: css
	go run *.go

# Build Svelte app and copy to deployment directory
build-svelte:
	cd svelte-app && npm run build
	cp -r svelte-app/build/* web/svelte-app/

# Full development build including Svelte
dev-full: build-svelte dev

# Build for production
build: assets
	go build -o bin/xanthus .

# Build for Windows 64-bit
build-windows: assets
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/xanthus.exe .

# Build for Linux ARM64
build-linux-arm64: assets
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/xanthus-linux-arm64 .

# Build for macOS Intel
build-macos-intel: assets
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/xanthus-macos-intel .

# Build for macOS Apple Silicon
build-macos-arm64: assets
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/xanthus-macos-arm64 .

# Build all platforms
build-all: assets
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/xanthus-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/xanthus-linux-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/xanthus-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/xanthus-macos-intel .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/xanthus-macos-arm64 .

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

# Run integration tests (excluding E2E)
test-integration:
	go test -v ./tests/integration/... -skip "TestE2E_"

# Run end-to-end tests
test-e2e:
	@echo "Running End-to-End tests..."
	@echo "Environment variables required:"
	@echo "  E2E_TEST_MODE: 'mock' (default) or 'live'"
	@echo "  TEST_HETZNER_API_KEY: Hetzner API key (for live tests)"
	@echo "  TEST_CLOUDFLARE_TOKEN: Cloudflare token (for live tests)"
	@echo "  TEST_CLOUDFLARE_ACCOUNT_ID: Cloudflare account ID (for live tests)"
	@echo "  TEST_DOMAIN: Test domain (default: test.xanthus.local)"
	@echo ""
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/... -timeout=30m

# Run E2E tests in live mode with real external services
test-e2e-live:
	@echo "Running E2E tests in LIVE mode with real external services..."
	@echo "WARNING: This will create real resources and may incur costs!"
	@read -p "Are you sure you want to continue? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	E2E_TEST_MODE=live go test -v ./tests/integration/e2e/... -timeout=60m

# Run specific E2E test suites
test-e2e-vps:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/vps_lifecycle_test.go -timeout=20m

test-e2e-ssl:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/ssl_management_test.go -timeout=15m

test-e2e-apps:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/application_deployment_test.go -timeout=15m

test-e2e-ui:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/ui_integration_test.go -timeout=20m

test-e2e-perf:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/performance_test.go -timeout=25m

test-e2e-security:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/security_test.go -timeout=20m

test-e2e-dr:
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/disaster_recovery_test.go -timeout=30m

# Run all structured tests (unit + integration, excluding E2E)
test:
	go test -v ./tests/unit/... ./tests/integration/... -skip "TestE2E_"

# Run tests with coverage (excluding E2E tests)
test-coverage:
	go test -v ./tests/unit/... ./tests/integration/... -skip "TestE2E_" -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run E2E tests with coverage
test-e2e-coverage:
	@echo "Running E2E tests with coverage..."
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./tests/integration/e2e/... -coverprofile=e2e-coverage.out -timeout=30m
	go tool cover -html=e2e-coverage.out -o e2e-coverage.html
	@echo "E2E coverage report generated: e2e-coverage.html"

# Run all tests including any legacy tests (excluding E2E)
test-all:
	go test -v ./... -skip "TestE2E_"

# Run everything including E2E tests
test-everything:
	@echo "Running ALL tests including E2E tests..."
	@echo "This may take a long time and create test resources..."
	E2E_TEST_MODE=$${E2E_TEST_MODE:-mock} go test -v ./... -timeout=60m

# Lint code
lint:
	go fmt ./...
	go vet ./...

# Clean build artifacts and test files
clean:
	rm -rf bin/
	rm -f web/static/css/output.css
	rm -rf web/static/js/vendor/
	rm -f coverage.out coverage.html
	rm -f e2e-coverage.out e2e-coverage.html

# Docker build targets
DOCKER_REGISTRY ?= ghcr.io/chrishham
DOCKER_IMAGE ?= xanthus
DOCKER_TAG ?= latest
DOCKER_PLATFORMS ?= linux/amd64,linux/arm64

# Build Docker image locally
docker-build:
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image to registry
docker-push:
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

# Tag Docker image with version
docker-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make docker-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	docker tag $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)

# Build multi-architecture Docker image
docker-multi:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make docker-multi VERSION=v1.0.0"; \
		exit 1; \
	fi
	docker buildx build --platform $(DOCKER_PLATFORMS) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest \
		--push .

# Display Docker commands
help-docker:
	@echo "Docker commands:"
	@echo ""
	@echo "  make docker-build     - Build Docker image locally"
	@echo "  make docker-push      - Push image to registry"
	@echo "  make docker-tag       - Tag image with version (requires VERSION=)"
	@echo "  make docker-multi     - Build and push multi-arch image (requires VERSION=)"
	@echo ""
	@echo "Environment variables:"
	@echo "  DOCKER_REGISTRY=ghcr.io/chrishham (default)"
	@echo "  DOCKER_IMAGE=xanthus (default)"
	@echo "  DOCKER_TAG=latest (default)"
	@echo "  DOCKER_PLATFORMS=linux/amd64,linux/arm64 (default)"
	@echo ""
	@echo "Examples:"
	@echo "  make docker-build"
	@echo "  make docker-tag VERSION=v1.0.0"
	@echo "  make docker-multi VERSION=v1.0.0"

# Create a new release
release:
	@echo "Creating a new release..."
	@echo ""
	@echo "This will:"
	@echo "1. Run tests to ensure quality"
	@echo "2. Create a new Git tag"
	@echo "3. Push the tag to trigger GitHub Actions release workflow"
	@echo "4. GitHub Actions will automatically:"
	@echo "   - Build multi-architecture Docker images"
	@echo "   - Create cross-platform binaries"
	@echo "   - Create GitHub Release with assets"
	@echo ""
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required."; \
		echo "Usage: make release VERSION=v1.0.0"; \
		echo ""; \
		echo "Version format: v<major>.<minor>.<patch> (e.g., v1.0.0)"; \
		echo "Pre-release format: v<major>.<minor>.<patch>-<pre> (e.g., v1.0.0-rc.1)"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@echo ""
	@echo "Running tests first..."
	@make test
	@echo ""
	@echo "Tests passed! Creating Git tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Pushing tag to GitHub (this will trigger the release workflow)..."
	@git push origin $(VERSION)
	@echo ""
	@echo "✅ Release $(VERSION) initiated!"
	@echo ""
	@echo "🚀 GitHub Actions will now:"
	@echo "   - Build Docker images for linux/amd64 and linux/arm64"
	@echo "   - Create binaries for Windows, macOS, and Linux"
	@echo "   - Publish to GitHub Container Registry (ghcr.io)"
	@echo "   - Create GitHub Release with all assets"
	@echo ""
	@echo "📦 Release artifacts will be available at:"
	@echo "   - Docker: ghcr.io/chrishham/xanthus:$(VERSION)"
	@echo "   - Binaries: https://github.com/chrishham/xanthus/releases/tag/$(VERSION)"

# Re-release an existing version (force update tag)
re-release:
	@echo "Re-releasing an existing version..."
	@echo ""
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required."; \
		echo "Usage: make re-release VERSION=v1.0.0"; \
		echo ""; \
		echo "This will force-update the existing tag to the current commit."; \
		echo "Use this to fix issues in a release without bumping version."; \
		exit 1; \
	fi
	@echo "Re-releasing $(VERSION) with current changes..."
	@echo ""
	@echo "Running tests first..."
	@make test
	@echo ""
	@echo "Tests passed! Force-updating Git tag $(VERSION)..."
	@git tag -f -a $(VERSION) -m "Re-release $(VERSION)"
	@echo "Force-pushing tag to GitHub..."
	@git push origin -f $(VERSION)
	@echo ""
	@echo "✅ Re-release $(VERSION) initiated!"
	@echo ""
	@echo "🚀 GitHub Actions will now rebuild and replace:"
	@echo "   - Docker images: ghcr.io/chrishham/xanthus:$(VERSION)"
	@echo "   - Release binaries and assets"
	@echo "   - GitHub Release page"

# Display available test commands
help-testing:
	@echo "Available test commands:"
	@echo ""
	@echo "  make test              - Run unit and integration tests (fast)"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo ""
	@echo "  make test-e2e          - Run end-to-end tests (mock mode)"
	@echo "  make test-e2e-live     - Run E2E tests with real services"
	@echo "  make test-e2e-coverage - Run E2E tests with coverage"
	@echo ""
	@echo "  E2E Test Suites:"
	@echo "    make test-e2e-vps      - VPS lifecycle tests"
	@echo "    make test-e2e-ssl      - SSL certificate management tests"
	@echo "    make test-e2e-apps     - Application deployment tests"
	@echo "    make test-e2e-ui       - User interface integration tests"
	@echo "    make test-e2e-perf     - Performance and load tests"
	@echo "    make test-e2e-security - Security tests"
	@echo "    make test-e2e-dr       - Disaster recovery tests"
	@echo ""
	@echo "  make test-all          - All tests except E2E"
	@echo "  make test-everything   - ALL tests including E2E"
	@echo ""
	@echo "Environment variables for E2E tests:"
	@echo "  E2E_TEST_MODE=mock|live"
	@echo "  TEST_HETZNER_API_KEY=your_key"
	@echo "  TEST_CLOUDFLARE_TOKEN=your_token"
	@echo "  TEST_CLOUDFLARE_ACCOUNT_ID=your_account_id"
	@echo "  TEST_DOMAIN=your_test_domain"
