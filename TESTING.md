# SpellingClash Testing Guide

This document provides comprehensive information about the test suite for the SpellingClash application.

## Overview

The test suite covers:
- **Services**: Business logic and data processing
- **Utilities**: Password handling, validation, sanitization
- **Handlers**: Game logic and mechanics
- **Models**: Data validation and calculations
- **Database**: Dialect abstraction and integration testing

## Running Tests

### Quick Test Run
```bash
go test ./...
```

### With Coverage Report
```bash
./test.sh
```

This will:
1. Run all tests with verbose output
2. Generate coverage report
3. Create an HTML coverage visualization (`coverage.html`)
4. Display coverage summary
5. Check against minimum coverage threshold (50%)

### Run Specific Tests
```bash
# Test a specific package
go test ./internal/service

# Test a specific function
go test ./internal/utils -run TestHashPassword

# Run only short tests (skip integration tests)
go test ./... -short
```

### With Coverage for Specific Package
```bash
go test ./internal/service -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Structure

### Service Layer Tests
**File**: `internal/service/practice_service_test.go`

Tests business logic for practice sessions:
- `TestWordsToIDString`: Word ID serialization
- `TestReorderWordsByIDs`: Word reordering logic
- `TestCalculatePoints`: Points calculation formulas

### Utility Tests

**File**: `internal/utils/password_test.go`
- `TestGenerateKidPassword`: Password generation and uniqueness
- `TestHashPassword`: Bcrypt hashing with salt
- `TestCheckPassword`: Password verification

**File**: `internal/utils/validation_test.go`
- `TestValidateEmail`: Email format validation (15+ test cases)
- `TestValidateName`: Name validation rules
- `TestSanitizeInput`: XSS protection and HTML escaping

### Handler Tests
**File**: `internal/handlers/game_logic_test.go`

Tests game mechanics:
- `TestGetMissingIndices`: Missing letter position generation
- `TestCalculateHangmanPoints`: Hangman scoring formula
- `TestCalculateMissingLetterPoints`: Missing letter scoring
- `TestSessionValidation`: Session expiration and validation

### Model Tests
**File**: `internal/models/models_test.go`

Tests data models:
- `TestSessionExpiration`: Session timeout logic
- `TestKidValidation`: Kid model field validation
- `TestWordValidation`: Word model requirements
- `TestKidStatsAccuracy`: Accuracy percentage calculation
- `TestFamilyCodeGeneration`: Unique code generation

### Database Tests

**File**: `internal/database/dialect_test.go`
- `TestDialectSelection`: SQLite/PostgreSQL/MySQL selection
- `TestPlaceholderReplacement`: Query parameter formatting
- `TestConnectionString`: Database URL parsing

**File**: `internal/database/integration_test.go`
- `TestDatabaseIntegration`: Full lifecycle testing
- `TestDatabaseTransactions`: Commit/rollback behavior
- `TestConcurrentAccess`: Thread-safety verification

## Coverage Goals

| Component | Target Coverage | Current Status |
|-----------|----------------|----------------|
| Services | 80% | ⏳ Pending |
| Utilities | 90% | ⏳ Pending |
| Handlers | 70% | ⏳ Pending |
| Models | 85% | ⏳ Pending |
| Database | 75% | ⏳ Pending |

## Writing New Tests

### Table-Driven Test Example
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty string", "", "", true},
        {"special chars", "test@123", "TEST@123", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            
            if tt.wantErr && err == nil {
                t.Error("Expected error, got nil")
            }
            if !tt.wantErr && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Integration Test Example
```go
func TestDatabaseOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Test
    result, err := PerformOperation(db)
    
    // Assert
    if err != nil {
        t.Fatalf("Operation failed: %v", err)
    }
    // ... more assertions
}
```

## Best Practices

### Do:
✅ Use table-driven tests for multiple scenarios
✅ Test edge cases and error conditions
✅ Use descriptive test names
✅ Clean up resources (defer, t.Cleanup())
✅ Use `testing.Short()` for integration tests
✅ Mock external dependencies
✅ Test both success and failure paths

### Don't:
❌ Test implementation details
❌ Write tests that depend on execution order
❌ Use real databases without cleanup
❌ Ignore test failures
❌ Write tests without assertions
❌ Test third-party library code

## Continuous Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test ./... -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
```

## Troubleshooting

### Test Failures
```bash
# Run with verbose output
go test -v ./...

# Run specific test with more detail
go test -v ./internal/utils -run TestValidateEmail
```

### Coverage Issues
```bash
# See which lines are not covered
go test -coverprofile=coverage.out ./internal/service
go tool cover -html=coverage.out
```

### Integration Test Issues
```bash
# Skip integration tests
go test ./... -short

# Run only integration tests
go test ./... -run Integration
```

## Future Enhancements

- [ ] Add HTTP handler tests with httptest
- [ ] Add repository layer tests with database mocks
- [ ] Add end-to-end tests with testcontainers
- [ ] Add benchmark tests for performance-critical code
- [ ] Add mutation testing
- [ ] Integrate with CI/CD pipeline
- [ ] Add test fixtures for common scenarios
- [ ] Add property-based testing for complex validation

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://go.dev/blog/cover)
- [testify Library](https://github.com/stretchr/testify) (optional, not currently used)

## Contributing

When adding new features:
1. Write tests first (TDD approach recommended)
2. Ensure tests pass: `go test ./...`
3. Check coverage: `./test.sh`
4. Update this README if adding new test categories
