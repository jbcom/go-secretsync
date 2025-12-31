# Testing Requirements

## Testing Philosophy

**Every feature must be tested.** Tests are not optional—they are the specification of how the code works and the safety net for future changes.

## Test Coverage Goals

- **Unit Tests:** 80%+ coverage for business logic
- **Integration Tests:** All critical workflows
- **Race Detection:** All tests pass with `-race`
- **Edge Cases:** Explicit tests for error conditions

## Types of Tests

### 1. Unit Tests

Test individual functions and methods in isolation.

**Location:** `*_test.go` files alongside source files

**Example:**
```go
// pkg/utils/deepmerge_test.go
func TestDeepMerge_ListAppend(t *testing.T) {
    base := map[string]interface{}{
        "items": []interface{}{"a", "b"},
    }
    overlay := map[string]interface{}{
        "items": []interface{}{"c"},
    }
    
    result := DeepMerge(base, overlay)
    
    expected := []interface{}{"a", "b", "c"}
    assert.Equal(t, expected, result["items"])
}
```

### 2. Integration Tests

Test multiple components working together with real external services (in test containers).

**Location:** `tests/integration/`

**Example:**
```go
// tests/integration/pipeline_test.go
func TestPipeline_VaultToAWS(t *testing.T) {
    // Setup: docker-compose provides Vault + LocalStack
    
    // Seed Vault with test secrets
    seedVault(t, vaultClient)
    
    // Run pipeline
    err := pipeline.Execute(ctx, config)
    require.NoError(t, err)
    
    // Verify secrets in AWS
    secrets := listAWSSecrets(t, awsClient)
    assert.Len(t, secrets, 5)
}
```

### 3. Table-Driven Tests

Test multiple scenarios with the same logic.

**Example:**
```go
func TestVaultClient_ListSecrets(t *testing.T) {
    tests := []struct {
        name        string
        vaultData   map[string]interface{}
        path        string
        want        []string
        wantErr     bool
        errContains string
    }{
        {
            name: "single level secrets",
            vaultData: map[string]interface{}{
                "secret/data/app": map[string]interface{}{
                    "keys": []interface{}{"api-key", "db-pass"},
                },
            },
            path: "secret/data/app",
            want: []string{"api-key", "db-pass"},
        },
        {
            name: "nested directories",
            vaultData: map[string]interface{}{
                "secret/metadata/": map[string]interface{}{
                    "keys": []interface{}{"app/", "db/"},
                },
                "secret/metadata/app": map[string]interface{}{
                    "keys": []interface{}{"api-key"},
                },
            },
            path: "secret/",
            want: []string{"app/api-key"},
        },
        {
            name:        "invalid path",
            path:        "secret/../etc/passwd",
            wantErr:     true,
            errContains: "path traversal",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock or test server
            client := setupTestClient(t, tt.vaultData)
            
            // Execute
            got, err := client.ListSecrets(context.Background(), tt.path)
            
            // Assert
            if tt.wantErr {
                require.Error(t, err)
                if tt.errContains != "" {
                    assert.Contains(t, err.Error(), tt.errContains)
                }
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 4. Mock-Based Tests

Use mocks to isolate components from external dependencies.

**Example:**
```go
type mockVaultAPI struct {
    listFunc func(context.Context, string) (*vault.Secret, error)
}

func (m *mockVaultAPI) List(ctx context.Context, path string) (*vault.Secret, error) {
    if m.listFunc != nil {
        return m.listFunc(ctx, path)
    }
    return nil, errors.New("not implemented")
}

func TestVaultClient_WithMock(t *testing.T) {
    mock := &mockVaultAPI{
        listFunc: func(ctx context.Context, path string) (*vault.Secret, error) {
            return &vault.Secret{
                Data: map[string]interface{}{
                    "keys": []interface{}{"secret1", "secret2"},
                },
            }, nil
        },
    }
    
    client := &VaultClient{api: mock}
    secrets, err := client.ListSecrets(context.Background(), "test/")
    
    require.NoError(t, err)
    assert.Len(t, secrets, 2)
}
```

### 5. Race Condition Tests

Verify thread safety with concurrent access.

**Example:**
```go
func TestCache_ConcurrentAccess(t *testing.T) {
    cache := NewCache()
    
    var wg sync.WaitGroup
    
    // Concurrent writers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                cache.Set(fmt.Sprintf("key-%d", id), value)
            }
        }(i)
    }
    
    // Concurrent readers
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                cache.Get(fmt.Sprintf("key-%d", id%10))
            }
        }(i)
    }
    
    wg.Wait()
    
    // Verify no race conditions occurred
    // Run with: go test -race
}
```

## Test Organization

### Test File Structure
```go
package pipeline_test  // Use _test package for black-box testing

import (
    "testing"
    "context"
    
    "github.com/extended-data-library/secretssync/pkg/pipeline"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Test fixtures and helpers at the top
func setupTestPipeline(t *testing.T) *pipeline.Pipeline {
    t.Helper()
    // Setup code...
}

// Test functions alphabetically
func TestPipeline_Execute(t *testing.T) { }
func TestPipeline_Merge(t *testing.T) { }
func TestPipeline_Sync(t *testing.T) { }
```

### Helper Functions
```go
// Mark helpers with t.Helper()
func setupVault(t *testing.T, data map[string]interface{}) *vault.Client {
    t.Helper()
    
    client, err := vault.NewClient(testConfig)
    require.NoError(t, err, "failed to create vault client")
    
    // Seed data...
    return client
}

// Cleanup with t.Cleanup()
func createTempConfig(t *testing.T) string {
    t.Helper()
    
    file, err := os.CreateTemp("", "config-*.yaml")
    require.NoError(t, err)
    
    t.Cleanup(func() {
        os.Remove(file.Name())
    })
    
    return file.Name()
}
```

## Integration Testing with Docker Compose

### Test Environment Setup

**File:** `docker-compose.test.yml`

```yaml
services:
  vault:
    image: hashicorp/vault:1.17.6
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: test-token
    ports:
      - "8200:8200"

  localstack:
    image: localstack/localstack:3.8.1
    environment:
      SERVICES: secretsmanager,s3,sts,organizations
    ports:
      - "4566:4566"

  vault-seeder:
    image: hashicorp/vault:1.17.6
    volumes:
      - ./tests/integration/scripts:/scripts
    command: /scripts/seed-vault.sh
    depends_on:
      - vault
```

### Running Integration Tests

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Wait for services
sleep 5

# Run integration tests
go test ./tests/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### Integration Test Example

```go
func TestIntegration_FullPipeline(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Connect to test Vault
    vaultClient := vault.NewClient(&vault.Config{
        Address: "http://localhost:8200",
        Token:   "test-token",
    })
    
    // Connect to LocalStack
    awsClient := aws.NewClient(&aws.Config{
        Endpoint: "http://localhost:4566",
        Region:   "us-east-1",
    })
    
    // Load test configuration
    config := loadTestConfig(t, "fixtures/pipeline-config.yaml")
    
    // Execute pipeline
    pipeline := pipeline.New(vaultClient, awsClient)
    err := pipeline.Execute(context.Background(), config)
    require.NoError(t, err)
    
    // Verify results
    secrets, err := awsClient.ListSecrets(context.Background())
    require.NoError(t, err)
    assert.Len(t, secrets, 10, "expected 10 secrets synced")
}
```

## Test Data and Fixtures

### Fixture Files
```
tests/integration/
├── fixtures/
│   ├── pipeline-config.yaml     # Test pipeline configuration
│   ├── secrets.yaml              # Test secret definitions
│   └── targets.yaml              # Test target definitions
├── scripts/
│   ├── seed-vault.sh            # Vault seeding script
│   └── seed-aws.sh              # AWS seeding script
└── testdata/
    ├── accounts.json             # Test AWS account data
    └── secrets_seed.json         # Test secret data
```

### Loading Fixtures
```go
func loadFixture(t *testing.T, filename string) []byte {
    t.Helper()
    
    data, err := os.ReadFile(filepath.Join("fixtures", filename))
    require.NoError(t, err, "failed to load fixture %s", filename)
    
    return data
}
```

## Assertions and Requirements

### Use Testify Effectively

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require: Fail fast on critical assertions
require.NoError(t, err, "setup failed")
require.NotNil(t, client, "client must not be nil")

// assert: Continue test after failure
assert.Equal(t, expected, actual)
assert.Contains(t, list, item)
assert.Len(t, result, 5)
```

### Custom Assertions
```go
func assertSecretsEqual(t *testing.T, expected, actual []Secret) {
    t.Helper()
    
    require.Len(t, actual, len(expected), "secret count mismatch")
    
    for i, exp := range expected {
        assert.Equal(t, exp.Path, actual[i].Path, "secret %d path mismatch", i)
        assert.Equal(t, exp.Data, actual[i].Data, "secret %d data mismatch", i)
    }
}
```

## Running Tests

### Standard Test Run
```bash
go test ./...
```

### Verbose Output
```bash
go test ./... -v
```

### With Race Detection
```bash
go test ./... -race
```

### Coverage Report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests Only
```bash
go test ./tests/integration/... -v
```

### Skip Integration Tests
```bash
go test ./... -short
```

## Test Checklist

Before marking a feature complete:

- [ ] Unit tests for all new functions
- [ ] Table-driven tests for multiple scenarios
- [ ] Error cases explicitly tested
- [ ] Edge cases covered (empty input, nil values, etc.)
- [ ] Integration test for end-to-end workflow
- [ ] Race detector passes (`-race`)
- [ ] Coverage meets 80% threshold
- [ ] Tests are deterministic (no flaky tests)
- [ ] Test names are descriptive
- [ ] Helper functions use `t.Helper()`
- [ ] Cleanup uses `t.Cleanup()` or `defer`

## Anti-Patterns to Avoid

### ❌ Silent Test Failures
```go
// BAD - Ignoring error
client.Connect()

// GOOD - Explicit assertion
err := client.Connect()
require.NoError(t, err)
```

### ❌ Non-Deterministic Tests
```go
// BAD - Random or time-dependent
time.Sleep(time.Second)
if time.Now().Unix()%2 == 0 { }

// GOOD - Deterministic, use mocks/fakes
mockClock.SetTime(fixedTime)
```

### ❌ Testing Implementation Details
```go
// BAD - Testing internal state
assert.Equal(t, 5, obj.internalCounter)

// GOOD - Testing observable behavior
result := obj.GetCount()
assert.Equal(t, 5, result)
```

### ❌ Shared Mutable State
```go
// BAD - Global test state
var testCache = NewCache()

// GOOD - Fresh state per test
func TestSomething(t *testing.T) {
    cache := NewCache()
    // Test with isolated cache
}
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

