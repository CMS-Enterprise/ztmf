#!/bin/bash

# Setup git hooks for ZTMF project
# Run this script to install local git hooks

HOOKS_DIR=".git/hooks"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT" || exit 1

echo "Setting up git hooks..."

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Create pre-push hook
cat > "$HOOKS_DIR/pre-push" << 'EOF'
#!/bin/bash

# Pre-push hook to run tests before pushing to remote

echo "Running pre-push checks..."

# Run unit tests
echo "Running unit tests..."
cd backend && go test -short ./... 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Unit tests failed. Push aborted."
    exit 1
fi
cd ..

# Check if dev environment is running
if ! docker ps | grep -q backend-api-1; then
    echo "⚠️  Dev environment not running. Starting..."
    make dev-up >/dev/null 2>&1
    sleep 5
fi

# Check if emberfall is installed
if ! command -v emberfall >/dev/null 2>&1; then
    echo "⚠️  Emberfall not installed, skipping E2E tests"
    echo "   Install with: curl -sSL https://raw.githubusercontent.com/aquia-inc/emberfall/main/install.sh | bash"
else
    # Run Emberfall E2E tests
    echo "Running Emberfall E2E tests..."
    emberfall ./backend/emberfall_tests.yml 2>&1
    if [ $? -ne 0 ]; then
        echo "❌ E2E tests failed. Push aborted."
        echo "   Fix the failing tests or skip this hook with: git push --no-verify"
        exit 1
    fi
fi

echo "✅ All checks passed. Proceeding with push..."
exit 0
EOF

# Make hook executable
chmod +x "$HOOKS_DIR/pre-push"

echo "✅ Git hooks installed successfully!"
echo ""
echo "Installed hooks:"
echo "  - pre-push: Runs unit tests and Emberfall E2E tests before push"
echo ""
echo "To bypass hooks temporarily, use: git push --no-verify"
