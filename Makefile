# ZTMF Development Environment Makefile

# Single source of truth for the local API port
API_PORT ?= 8080

.PHONY: dev-setup dev-up dev-down dev-logs generate-jwt clean help test-empire-data test test-unit test-integration test-coverage test-coverage-view test-coverage-text test-e2e test-full full-stack-up full-stack-down frontend-env

# Default target
help:
	@echo "ZTMF Development Environment"
	@echo ""
	@echo "Development:"
	@echo "  make dev-setup       Create development docker-compose file and start services"
	@echo "  make dev-up          Start backend services only"
	@echo "  make dev-down        Stop backend services only"
	@echo "  make dev-logs        Show backend service logs"
	@echo "  make frontend-env    Generate frontend .env.development.local (auto-login token)"
	@echo "  make full-stack-up   Start both backend and frontend (configures everything)"
	@echo "  make full-stack-down Stop both backend and frontend"
	@echo "  make clean           Clean up generated files"
	@echo ""
	@echo "Testing:"
	@echo "  make test                Run all tests"
	@echo "  make test-unit           Run unit tests only (fast)"
	@echo "  make test-integration    Run integration tests"
	@echo "  make test-coverage       Run tests and generate HTML coverage report"
	@echo "  make test-coverage-view  Open HTML coverage report in browser"
	@echo "  make test-coverage-text  Show coverage summary in terminal"
	@echo "  make test-e2e            Run Emberfall E2E tests"
	@echo "  make test-full           Run all tests including E2E (comprehensive)"
	@echo ""
	@echo "Authentication:"
	@echo "  make generate-jwt     Generate JWT token for testing (requires EMAIL variable)"
	@echo "  make test-empire-data Generate JWT tokens for Empire test users"
	@echo ""
	@echo "Examples:"
	@echo "  make dev-setup                           # Full dev environment setup"
	@echo "  make test                                 # Run all tests"
	@echo "  make generate-jwt EMAIL=test@example.com # Generate JWT for specific user"
	@echo "  make test-empire-data                    # Get tokens for Star Wars test users"

# Create compose-dev.yml and start services
dev-setup: backend/compose-dev.yml backend/dev.compose.env
	@echo "🚀 Starting development environment..."
	cd backend && docker compose -f compose-dev.yml up -d
	@echo "✅ Development environment ready!"
	@echo "📡 API available at: http://localhost:$(API_PORT)"
	@echo "🗄️  Database available at: localhost:54321"
	@echo ""
	@echo "🧪 Ready to test! Run:"
	@echo "  make test-empire-data    # Get JWT tokens for all test users"
	@echo ""
	@echo "📋 Example API tests:"
	@echo "  # Basic current user info:"
	@echo "  curl -H \"Authorization: TOKEN\" \"http://localhost:$(API_PORT)/api/v1/users/current\""
	@echo ""
	@echo "  # Scores with pillar breakdown (new feature):"
	@echo "  curl -H \"Authorization: TOKEN\" \"http://localhost:$(API_PORT)/api/v1/scores/aggregate?include_pillars=true\""
	@echo ""
	@echo "  # List all FISMA systems:"
	@echo "  curl -H \"Authorization: TOKEN\" \"http://localhost:$(API_PORT)/api/v1/fismasystems\""
	@echo ""
	@echo "💡 Replace TOKEN with output from 'make test-empire-data'"

# Generate the compose-dev.yml file
backend/compose-dev.yml:
	@echo "📝 Creating compose-dev.yml..."
	@echo "# Generated development docker-compose file" > backend/compose-dev.yml
	@echo "# DO NOT EDIT - Managed by Makefile" >> backend/compose-dev.yml
	@echo "" >> backend/compose-dev.yml
	@echo "services:" >> backend/compose-dev.yml
	@echo "  postgre:" >> backend/compose-dev.yml
	@echo "    image: postgres:16.8" >> backend/compose-dev.yml
	@echo "    env_file:" >> backend/compose-dev.yml
	@echo "      - dev.compose.env" >> backend/compose-dev.yml
	@echo "    network_mode: host" >> backend/compose-dev.yml
	@echo "    volumes:" >> backend/compose-dev.yml
	@echo "      - postgres_data:/var/lib/postgresql/data" >> backend/compose-dev.yml
	@echo "    command: -p 54321" >> backend/compose-dev.yml
	@echo "    healthcheck:" >> backend/compose-dev.yml
	@echo "      test: [\"CMD-SHELL\", \"pg_isready -U admin -d ztmf -p 54321\"]" >> backend/compose-dev.yml
	@echo "      interval: 5s" >> backend/compose-dev.yml
	@echo "      timeout: 5s" >> backend/compose-dev.yml
	@echo "      retries: 5" >> backend/compose-dev.yml
	@echo "" >> backend/compose-dev.yml
	@echo "  api:" >> backend/compose-dev.yml
	@echo "    build:" >> backend/compose-dev.yml
	@echo "      context: ." >> backend/compose-dev.yml
	@echo "      dockerfile: Dockerfile" >> backend/compose-dev.yml
	@echo "    command: [\"/usr/local/bin/ztmfapi\"]" >> backend/compose-dev.yml
	@echo "    env_file:" >> backend/compose-dev.yml
	@echo "      - dev.compose.env" >> backend/compose-dev.yml
	@echo "    network_mode: host" >> backend/compose-dev.yml
	@echo "    volumes:" >> backend/compose-dev.yml
	@echo "      - ./_test_data_empire.sql:/app/_test_data_empire.sql:ro" >> backend/compose-dev.yml
	@echo "    depends_on:" >> backend/compose-dev.yml
	@echo "      postgre:" >> backend/compose-dev.yml
	@echo "        condition: service_healthy" >> backend/compose-dev.yml
	@echo "" >> backend/compose-dev.yml
	@echo "volumes:" >> backend/compose-dev.yml
	@echo "  postgres_data:" >> backend/compose-dev.yml
	@echo "✅ compose-dev.yml created"

# Create dev.compose.env for development (PHONY so it always regenerates)
.PHONY: backend/dev.compose.env
backend/dev.compose.env:
	@echo "📝 Creating dev.compose.env for development..."
	@echo "# Development environment file - Generated by Makefile" > backend/dev.compose.env
	@echo "# Fixed password for local development" >> backend/dev.compose.env
	@echo "" >> backend/dev.compose.env
	@echo "# for postgre container" >> backend/dev.compose.env
	@echo "POSTGRES_DB=ztmf" >> backend/dev.compose.env
	@echo "POSTGRES_USER=admin" >> backend/dev.compose.env
	@echo "POSTGRES_PASSWORD=localdevpassword" >> backend/dev.compose.env
	@echo "" >> backend/dev.compose.env
	@echo "# for api container" >> backend/dev.compose.env
	@echo "DB_ENDPOINT=localhost" >> backend/dev.compose.env
	@echo "DB_PORT=54321" >> backend/dev.compose.env
	@echo "DB_NAME=ztmf" >> backend/dev.compose.env
	@echo "DB_USER=admin" >> backend/dev.compose.env
	@echo "DB_PASS=localdevpassword" >> backend/dev.compose.env
	@echo "DB_POPULATE=/app/_test_data_empire.sql" >> backend/dev.compose.env
	@echo "" >> backend/dev.compose.env
	@echo "# for api auth handling" >> backend/dev.compose.env
	@echo "AUTH_HS256_SECRET=zeroTrust" >> backend/dev.compose.env
	@echo "AUTH_HEADER_FIELD=Authorization" >> backend/dev.compose.env
	@echo "" >> backend/dev.compose.env
	@echo "# api server settings" >> backend/dev.compose.env
	@echo "PORT=$(API_PORT)" >> backend/dev.compose.env
	@echo "ENVIRONMENT=local" >> backend/dev.compose.env
	@echo "✅ dev.compose.env created"

# Start development services
dev-up: backend/compose-dev.yml backend/dev.compose.env
	@if docker ps | grep -q backend-api-1; then \
		echo "✅ Development services already running"; \
	else \
		echo "🚀 Starting development services..."; \
		cd backend && docker compose -f compose-dev.yml up -d; \
	fi

# Stop development services
dev-down:
	@echo "🛑 Stopping development services..."
	cd backend && docker compose -f compose-dev.yml down

# Show service logs
dev-logs:
	cd backend && docker compose -f compose-dev.yml logs -f

# Generate JWT token for testing
generate-jwt:
	@if [ -z "$(EMAIL)" ]; then \
		echo "❌ ERROR: EMAIL variable required"; \
		echo "Usage: make generate-jwt EMAIL=your.email@example.com"; \
		exit 1; \
	fi
	@echo "🔑 Generating JWT token for: $(EMAIL)"
	@echo "Header (base64):"
	@echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='
	@echo ""
	@echo "Payload (base64):"
	@echo -n '{"email":"$(EMAIL)"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='
	@echo ""
	@echo "Signature:"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"$(EMAIL)"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "$$SIGNATURE"
	@echo ""
	@echo "🎯 Complete JWT Token:"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"$(EMAIL)"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "$$HEADER.$$PAYLOAD.$$SIGNATURE"
	@echo ""
	@echo "📋 Test with curl:"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"$(EMAIL)"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "curl -H \"Authorization: $$HEADER.$$PAYLOAD.$$SIGNATURE\" \"http://localhost:$(API_PORT)/api/v1/users/current\""

# Generate JWT tokens for Empire test users
test-empire-data:
	@echo "🏴 Imperial Test Users and JWT Tokens"
	@echo ""
	@echo "👑 ADMIN - Grand Moff Tarkin:"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"Grand.Moff@DeathStar.Empire"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "  EMAIL: Grand.Moff@DeathStar.Empire"; \
	echo "  TOKEN: $$HEADER.$$PAYLOAD.$$SIGNATURE"
	@echo ""
	@echo "🚢 ISSO - Admiral Piett (Executor Systems):"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"Admiral.Piett@executor.empire"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "  EMAIL: Admiral.Piett@executor.empire"; \
	echo "  TOKEN: $$HEADER.$$PAYLOAD.$$SIGNATURE"
	@echo ""
	@echo "❄️  ISSO - General Veers (Death Star Systems):"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"Commander.Veers@hoth.empire"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "  EMAIL: Commander.Veers@hoth.empire"; \
	echo "  TOKEN: $$HEADER.$$PAYLOAD.$$SIGNATURE"
	@echo ""
	@echo "🛡️  ISSO - Director Krennic (Shield Generator Systems):"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"Director.Krennic@scarif.empire"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "  EMAIL: Director.Krennic@scarif.empire"; \
	echo "  TOKEN: $$HEADER.$$PAYLOAD.$$SIGNATURE"
	@echo ""
	@echo "📋 Test with pillar scores:"
	@HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n '{"email":"Grand.Moff@DeathStar.Empire"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	echo "curl -H \"Authorization: $$HEADER.$$PAYLOAD.$$SIGNATURE\" \"http://localhost:$(API_PORT)/api/v1/scores/aggregate?include_pillars=true\""

# Clean up generated files
clean:
	@echo "🧹 Cleaning up generated files..."
	@rm -f backend/compose-dev.yml backend/dev.compose.env backend/dev-postgres.crt backend/dev-postgres.key backend/dev-postgres-certs backend/dev-postgres-init.sh
	@echo "✅ Clean complete"

# Test targets
test:
	@echo "🧪 Running all tests..."
	cd backend && go test ./...

test-unit:
	@echo "🧪 Running unit tests (fast)..."
	cd backend && go test -short ./...

test-integration:
	@echo "🧪 Running integration tests..."
	cd backend && go test -run Integration ./...

test-coverage:
	@echo "Running tests with coverage..."
	cd backend && go test -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

test-coverage-view:
	@if [ ! -f backend/coverage.html ]; then \
		echo "Coverage report not found. Run 'make test-coverage' first."; \
		exit 1; \
	fi
	@echo "Opening coverage report..."
	@if command -v xdg-open >/dev/null 2>&1; then \
		xdg-open backend/coverage.html; \
	elif command -v open >/dev/null 2>&1; then \
		open backend/coverage.html; \
	else \
		echo "Coverage report location: backend/coverage.html"; \
	fi

test-coverage-text:
	@echo "Running tests with coverage..."
	@cd backend && go test -cover ./...

test-e2e:
	@echo "🧪 Running Emberfall E2E tests (isolated environment)..."
	@if ! command -v emberfall >/dev/null 2>&1; then \
		echo "❌ Emberfall not installed"; \
		echo "   Install: curl -L https://github.com/aquia-inc/emberfall/releases/download/v0.3.1/emberfall_Linux_x86_64.tar.gz | tar -xz && mv emberfall ~/.local/bin/"; \
		exit 1; \
	fi
	@echo "🧹 Cleaning up any existing test containers..."
	@cd backend && docker compose -f compose-test.yml down -v 2>/dev/null || true
	@echo "🚀 Starting fresh test environment (port 8090)..."
	@cd backend && docker compose -f compose-test.yml up -d --build
	@echo "⏳ Waiting for API to be ready..."
	@sleep 15
	@echo "🔥 Running Emberfall tests..."
	@sed 's/localhost:8080/localhost:8090/g' backend/emberfall_tests.yml > /tmp/emberfall_test_isolated.yml
	@emberfall --config /tmp/emberfall_test_isolated.yml || (echo "❌ Emberfall tests failed"; cd backend && docker compose -f compose-test.yml down -v; exit 1)
	@echo "🧹 Cleaning up test environment..."
	@cd backend && docker compose -f compose-test.yml down -v
	@rm /tmp/emberfall_test_isolated.yml
	@echo "✅ E2E tests passed!"

test-full:
	@echo "Running comprehensive test suite..."
	@echo ""
	@echo "1/3 Running unit tests..."
	@cd backend && go test -short ./...
	@echo ""
	@echo "2/3 Generating coverage report..."
	@cd backend && go test -cover ./...
	@echo ""
	@echo "3/3 Running Emberfall E2E tests (isolated containers)..."
	@make test-e2e
	@echo ""
	@echo "✅ All tests complete"

# Generate frontend .env.development.local with a working admin token
# Uses Grand Moff Tarkin (ADMIN) from seed data by default
# Override with: make frontend-env EMAIL=your.email@example.com
frontend-env:
	@if [ ! -d "../ztmf-ui" ]; then \
		echo "❌ Frontend not found at ../ztmf-ui"; \
		echo "   Clone ztmf-ui repo at same level as ztmf:"; \
		echo "   git clone <ztmf-ui-repo-url> ../ztmf-ui"; \
		exit 1; \
	fi
	@DEV_EMAIL=$${EMAIL:-Grand.Moff@DeathStar.Empire}; \
	HEADER=$$(echo -n '{"alg":"HS256"}' | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	PAYLOAD=$$(echo -n "{\"email\":\"$$DEV_EMAIL\"}" | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	SIGNATURE=$$(echo -n "$$HEADER.$$PAYLOAD" | openssl dgst -sha256 -hmac 'zeroTrust' -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='); \
	TOKEN="$$HEADER.$$PAYLOAD.$$SIGNATURE"; \
	echo "# Local development configuration" > ../ztmf-ui/.env.development.local; \
	echo "# Generated by: make frontend-env" >> ../ztmf-ui/.env.development.local; \
	echo "# Auth token for: $$DEV_EMAIL" >> ../ztmf-ui/.env.development.local; \
	echo "# Re-run 'make frontend-env' or 'make frontend-env EMAIL=you@example.com' to update" >> ../ztmf-ui/.env.development.local; \
	echo "" >> ../ztmf-ui/.env.development.local; \
	echo "# Enable local dev mode (bypasses OIDC - NEVER set in AWS builds)" >> ../ztmf-ui/.env.development.local; \
	echo "VITE_LOCAL_DEV=true" >> ../ztmf-ui/.env.development.local; \
	echo "" >> ../ztmf-ui/.env.development.local; \
	echo "# Backend API running locally" >> ../ztmf-ui/.env.development.local; \
	echo "VITE_CF_DOMAIN=http://localhost:$(API_PORT)" >> ../ztmf-ui/.env.development.local; \
	echo "" >> ../ztmf-ui/.env.development.local; \
	echo "# Disable IDP for local dev" >> ../ztmf-ui/.env.development.local; \
	echo "VITE_IDP_ENABLED=false" >> ../ztmf-ui/.env.development.local; \
	echo "" >> ../ztmf-ui/.env.development.local; \
	echo "# Auto-generated auth token ($$DEV_EMAIL)" >> ../ztmf-ui/.env.development.local; \
	echo "VITE_AUTH_TOKEN3=$$TOKEN" >> ../ztmf-ui/.env.development.local; \
	echo "" >> ../ztmf-ui/.env.development.local; \
	echo "# Legacy settings" >> ../ztmf-ui/.env.development.local; \
	echo "TOKEN=" >> ../ztmf-ui/.env.development.local; \
	echo "PROXY_URI=" >> ../ztmf-ui/.env.development.local; \
	echo "✅ Frontend configured for: $$DEV_EMAIL"; \
	echo "   File: ../ztmf-ui/.env.development.local"; \
	echo ""; \
	echo "   To use a different user:"; \
	echo "   make frontend-env EMAIL=your.email@example.com"

# Full stack development (backend + frontend)
full-stack-up:
	@echo "Starting ZTMF full stack development environment..."
	@if [ ! -d "../ztmf-ui" ]; then \
		echo "❌ Frontend not found at ../ztmf-ui"; \
		echo "   Clone ztmf-ui repo at same level as ztmf"; \
		exit 1; \
	fi
	@echo ""
	@echo "1/3 Starting backend..."
	@make dev-up
	@sleep 3
	@echo ""
	@echo "2/3 Configuring frontend auth..."
	@make frontend-env
	@echo ""
	@echo "3/3 Starting frontend..."
	@rm -rf ../ztmf-ui/node_modules/.vite
	@cd ../ztmf-ui && npm run dev &
	@sleep 2
	@echo ""
	@echo "✅ Full stack started!"
	@echo ""
	@echo "Services:"
	@echo "  Frontend UI:  http://localhost:5174"
	@echo "  Backend API:  http://localhost:$(API_PORT)"
	@echo "  Database:     localhost:54321"
	@echo ""
	@echo "Logged in as: Grand Moff Tarkin (ADMIN)"
	@echo "To switch user: make frontend-env EMAIL=Admiral.Piett@executor.empire"
	@echo ""
	@echo "Stop with: make full-stack-down"

full-stack-down:
	@echo "Stopping ZTMF full stack..."
	@make dev-down
	@pkill -f "vite" || true
	@echo "✅ Full stack stopped"
