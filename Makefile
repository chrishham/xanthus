.PHONY: dev build test test-unit test-integration test-e2e test-e2e-live test-e2e-coverage test-e2e-vps test-e2e-ssl test-e2e-apps test-e2e-ui test-e2e-perf test-e2e-security test-e2e-dr test-coverage test-all test-everything lint css css-watch clean help-testing

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
