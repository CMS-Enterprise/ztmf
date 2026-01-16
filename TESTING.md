# ZTMF Testing Guide

## Testing Philosophy

This project is adopting a "new code first" testing strategy:
- **All new code** should include tests
- **Bug fixes** should include regression tests
- **Gradually add tests** to existing code during refactoring
- **Target 70%+ coverage** for new features

## Current Testing Stack (2026)

### Core Tools

1. **Standard Library** (`testing`) - Primary test framework
   - Zero dependencies
   - Fast, reliable, built into Go
   - Full tooling support

2. **Testify** (`github.com/stretchr/testify`) - Assertion library
   - Cleaner assertions with `assert` and `require`
   - Mocking capabilities with `mock`
   - Suite support for complex test setups

3. **Emberfall** - E2E API testing (already in use!)
   - Runs in CI/CD pipeline
   - Tests full request/response flows
   - Located: `backend/emberfall_tests.yml`

### Future Considerations

- **testcontainers-go** - For spinning up real PostgreSQL in tests
- **go-sqlmock** - For mocking database interactions
- **httptest** - For testing HTTP handlers (built into stdlib)

## Running Tests

### Quick Commands

```bash
# Run unit tests only (fast)
make test-unit

# Run all tests with coverage
make test-coverage

# Run comprehensive test suite (unit + coverage + E2E)
make test-full
```

### Manual Go Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run only unit tests (skip integration tests)
go test -short ./...

# Run specific package
go test ./internal/model/...

# Run specific test
go test -run TestDeleteFismaSystem ./internal/model/

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...

# Run tests with race detection
go test -race ./...
```

## Test Organization

### File Naming Convention

```
code_file.go       -> code_file_test.go
fismasystems.go    -> fismasystems_test.go
users.go           -> users_test.go
```

### Test Categories

#### 1. Unit Tests
Test individual functions in isolation.

```go
func TestDeleteFismaSystem(t *testing.T) {
    // Use t.Skip() if database not available
    if testing.Short() {
        t.Skip("Skipping database test in short mode")
    }

    // Test implementation...
}
```

#### 2. Integration Tests
Test interactions between components.

```go
func TestFismaSystemsEndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Create, update, delete flow...
}
```

#### 3. E2E Tests
Use Emberfall for API-level testing (already implemented).

## Testing Patterns

### Table-Driven Tests (Recommended)

```go
func TestFismaSystem_Validate(t *testing.T) {
    tests := []struct {
        name    string
        system  FismaSystem
        wantErr bool
    }{
        {
            name:    "ValidSystem",
            system:  FismaSystem{/* valid data */},
            wantErr: false,
        },
        {
            name:    "InvalidEmail",
            system:  FismaSystem{/* invalid email */},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.system.validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Using Testify

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDeleteFismaSystem(t *testing.T) {
    err := DeleteFismaSystem(ctx, 0)

    // assert - test continues on failure
    assert.Error(t, err)
    assert.Equal(t, ErrNoData, err)

    // require - test stops immediately on failure
    require.Error(t, err)
}
```

### Testing HTTP Handlers

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestListFismaSystems(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/fismasystems", nil)
    req.Header.Set("Authorization", testToken)

    w := httptest.NewRecorder()

    ListFismaSystems(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Mocking Database Queries

For database-heavy functions, consider:

1. **Acceptance tests** with real DB (testcontainers)
2. **Mock interfaces** for unit tests
3. **Repository pattern** to make mocking easier

Example refactor:
```go
// Current: Hard to test
func DeleteFismaSystem(ctx context.Context, id int32) error {
    sqlb := stmntBuilder.Update("fismasystems")...
    // Direct DB call
}

// Better: Testable with interface
type FismaSystemRepository interface {
    Delete(ctx context.Context, id int32) error
}

// Mock for testing
type MockFismaSystemRepo struct {
    DeleteFunc func(ctx context.Context, id int32) error
}
```

## CI/CD Integration

Tests run automatically in GitHub Actions (`.github/workflows/backend.yml`):

```yaml
- name: Run Unit Tests  # ADD THIS
  run: |
    cd backend
    go test -short -cover ./...

- name: Emberfall Smoke Tests  # ALREADY EXISTS
  uses: aquia-inc/emberfall@main
  with:
    version: 0.3.1
    file: ./backend/emberfall_tests.yml
```

## Testing Checklist for New Features

When adding a new feature (like FISMA system decommission):

- [ ] **Unit tests** for model functions
  - [ ] Valid input cases
  - [ ] Invalid input cases
  - [ ] Edge cases (empty strings, null values, etc.)

- [ ] **Controller tests** for HTTP handlers
  - [ ] Successful requests
  - [ ] Authorization failures
  - [ ] Validation errors

- [ ] **Integration tests** if needed
  - [ ] Database operations
  - [ ] External service calls

- [ ] **Emberfall E2E tests**
  - [ ] Happy path flow
  - [ ] Error scenarios

## Example: Testing the Decommission Feature

### Unit Test (Model)
```go
// backend/internal/model/fismasystems_test.go
func TestDeleteFismaSystem(t *testing.T) {
    // Test invalid ID
    // Test non-existent system
    // Test successful decommission
}
```

### Controller Test
```go
// backend/cmd/api/internal/controller/fismasystems_test.go
func TestDeleteFismaSystemHandler(t *testing.T) {
    // Test admin can decommission
    // Test non-admin gets 403
    // Test invalid ID returns 404
}
```

### E2E Test (Emberfall)
```yaml
# backend/emberfall_tests.yml
- name: Decommission FISMA System
  request:
    method: DELETE
    url: http://localhost:8080/api/v1/fismasystems/{{fismasystemid}}
    headers:
      <<: *commonHeaders
  response:
    statusCode: 204
```

## Coverage Goals

- **New features**: 80%+ coverage
- **Bug fixes**: Include regression test
- **Overall project**: Gradually increase from 0% → 50% → 70%

## Best Practices

1. **Write tests first** for new features (TDD)
2. **Keep tests simple** - one assertion per test when possible
3. **Use descriptive names** - `TestDeleteFismaSystem_WithInvalidID_ReturnsError`
4. **Don't test the framework** - focus on your business logic
5. **Mock external dependencies** - databases, APIs, time, etc.
6. **Test error paths** - not just happy paths
7. **Use table-driven tests** for multiple similar cases
8. **Skip slow tests** with `-short` flag for quick feedback

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Emberfall](https://github.com/aquia-inc/emberfall)

## Getting Started

Start testing your decommission feature:

```bash
cd backend
go test ./internal/model/ -v -run TestDeleteFismaSystem
```

Then gradually expand test coverage to other areas of the codebase!
