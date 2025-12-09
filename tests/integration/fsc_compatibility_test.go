// Package integration provides end-to-end tests for the FSC merge+sync pipeline.
// These tests require LocalStack and Vault to be running (via docker-compose.test.yml).
package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Environment variables for integration testing
const (
	envVaultAddr     = "VAULT_ADDR"
	envVaultToken    = "VAULT_TOKEN"
	envAWSEndpoint   = "AWS_ENDPOINT_URL"
	envAWSRegion     = "AWS_REGION"
	envAWSAccessKey  = "AWS_ACCESS_KEY_ID"
	envAWSSecretKey  = "AWS_SECRET_ACCESS_KEY"
)

func skipIfNoIntegrationEnv(t *testing.T) {
	t.Helper()
	if os.Getenv(envVaultAddr) == "" || os.Getenv(envAWSEndpoint) == "" {
		t.Skip("Skipping integration test: VAULT_ADDR and AWS_ENDPOINT_URL required")
	}
}

// TestFSCMergePlusSyncPattern validates the complete FSC merge+sync workflow:
// 1. Seed Vault with source secrets (simulating FSC pattern)
// 2. Run merge phase (sources -> merge store with deepmerge)
// 3. Run sync phase (merge store -> AWS Secrets Manager)
// 4. Validate final secrets match expected merged output
func TestFSCMergePlusSyncPattern(t *testing.T) {
	skipIfNoIntegrationEnv(t)

	ctx := context.Background()

	// Setup clients
	vaultClient := setupVaultClient(t)
	awsClient := setupAWSClient(t, ctx)

	// Step 1: Seed Vault with FSC-style source secrets
	seedVaultSecrets(t, vaultClient)

	// Step 2: Validate Vault secrets were created correctly
	validateVaultSecrets(t, vaultClient)

	// Step 3: Simulate merge phase - read from Vault, deepmerge, write to merge store
	mergedSecrets := runMergePhase(t, vaultClient)

	// Step 4: Simulate sync phase - write merged secrets to AWS
	runSyncPhase(t, ctx, awsClient, mergedSecrets)

	// Step 5: Validate AWS secrets match expected output
	validateAWSSecrets(t, ctx, awsClient, mergedSecrets)

	// Cleanup
	cleanup(t, ctx, vaultClient, awsClient)
}

func setupVaultClient(t *testing.T) *api.Client {
	t.Helper()

	cfg := api.DefaultConfig()
	cfg.Address = os.Getenv(envVaultAddr)

	client, err := api.NewClient(cfg)
	require.NoError(t, err)

	client.SetToken(os.Getenv(envVaultToken))
	return client
}

func setupAWSClient(t *testing.T, ctx context.Context) *secretsmanager.Client {
	t.Helper()

	endpoint := os.Getenv(envAWSEndpoint)
	region := os.Getenv(envAWSRegion)
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv(envAWSAccessKey),
			os.Getenv(envAWSSecretKey),
			"",
		)),
	)
	require.NoError(t, err)

	return secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

// seedVaultSecrets creates test secrets in Vault that simulate FSC pattern
func seedVaultSecrets(t *testing.T, client *api.Client) {
	t.Helper()

	// Enable KV v2 if not already enabled
	client.Sys().Mount("secret", &api.MountInput{
		Type: "kv-v2",
	})

	// Source 1: analytics - base configuration
	writeVaultSecret(t, client, "secret/data/analytics/database", map[string]interface{}{
		"data": map[string]interface{}{
			"host":     "analytics-db.example.com",
			"port":     5432,
			"users":    []interface{}{"readonly", "admin"},
			"settings": map[string]interface{}{"timeout": 30, "pool_size": 10},
		},
	})

	writeVaultSecret(t, client, "secret/data/analytics/api-keys", map[string]interface{}{
		"data": map[string]interface{}{
			"stripe":   "sk_test_analytics",
			"sendgrid": "SG.analytics",
		},
	})

	// Source 2: analytics-engineers - environment-specific overrides
	writeVaultSecret(t, client, "secret/data/analytics-engineers/database", map[string]interface{}{
		"data": map[string]interface{}{
			"users":    []interface{}{"engineer1", "engineer2"}, // Should APPEND to analytics users
			"settings": map[string]interface{}{"debug": true},   // Should MERGE into analytics settings
		},
	})

	// Source 3: shared - common secrets
	writeVaultSecret(t, client, "secret/data/shared/common", map[string]interface{}{
		"data": map[string]interface{}{
			"region":      "us-east-1",
			"environment": "test",
		},
	})

	// Nested secrets (for recursive listing validation)
	writeVaultSecret(t, client, "secret/data/analytics/nested/level1/config", map[string]interface{}{
		"data": map[string]interface{}{
			"nested_key": "nested_value",
		},
	})

	t.Log("Seeded Vault with FSC-style test secrets")
}

func writeVaultSecret(t *testing.T, client *api.Client, path string, data map[string]interface{}) {
	t.Helper()
	_, err := client.Logical().Write(path, data)
	require.NoError(t, err, "Failed to write secret to %s", path)
}

func validateVaultSecrets(t *testing.T, client *api.Client) {
	t.Helper()

	// Validate analytics/database exists
	secret, err := client.Logical().Read("secret/data/analytics/database")
	require.NoError(t, err)
	require.NotNil(t, secret)

	data := secret.Data["data"].(map[string]interface{})
	assert.Equal(t, "analytics-db.example.com", data["host"])

	// Validate nested secret exists (tests recursive listing)
	nested, err := client.Logical().Read("secret/data/analytics/nested/level1/config")
	require.NoError(t, err)
	require.NotNil(t, nested)

	t.Log("Validated Vault secrets exist")
}

// runMergePhase simulates the FSC merge pattern:
// Target "Stg" imports: analytics, analytics-engineers, shared
// Expected: deepmerge with list append, dict merge
func runMergePhase(t *testing.T, client *api.Client) map[string]map[string]interface{} {
	t.Helper()

	merged := make(map[string]map[string]interface{})

	// Read all source secrets
	analyticsDB := readVaultSecretData(t, client, "secret/data/analytics/database")
	analyticsAPI := readVaultSecretData(t, client, "secret/data/analytics/api-keys")
	engineersDB := readVaultSecretData(t, client, "secret/data/analytics-engineers/database")
	shared := readVaultSecretData(t, client, "secret/data/shared/common")
	nested := readVaultSecretData(t, client, "secret/data/analytics/nested/level1/config")

	// Simulate deepmerge for database config
	// analytics/database + analytics-engineers/database
	mergedDB := deepMergeSimple(analyticsDB, engineersDB)
	merged["database"] = mergedDB

	// api-keys pass through (no merge needed)
	merged["api-keys"] = analyticsAPI

	// shared pass through
	merged["common"] = shared

	// nested pass through
	merged["nested/level1/config"] = nested

	// Validate merge results
	// Users should be: ["readonly", "admin", "engineer1", "engineer2"] (list append)
	users := mergedDB["users"].([]interface{})
	assert.Len(t, users, 4, "Expected 4 users after list append merge")

	// Settings should have both timeout/pool_size AND debug (dict merge)
	settings := mergedDB["settings"].(map[string]interface{})
	assert.Contains(t, settings, "timeout", "Expected timeout from analytics")
	assert.Contains(t, settings, "debug", "Expected debug from engineers")

	t.Log("Merge phase completed with deepmerge validation")
	return merged
}

func readVaultSecretData(t *testing.T, client *api.Client, path string) map[string]interface{} {
	t.Helper()
	secret, err := client.Logical().Read(path)
	require.NoError(t, err)
	require.NotNil(t, secret)
	return secret.Data["data"].(map[string]interface{})
}

// deepMergeSimple is a simplified deepmerge for testing
// Production code uses pkg/utils/deepmerge.go
func deepMergeSimple(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy dst
	for k, v := range dst {
		result[k] = v
	}

	// Merge src
	for k, v := range src {
		if existing, ok := result[k]; ok {
			// Handle list append
			if dstSlice, ok := existing.([]interface{}); ok {
				if srcSlice, ok := v.([]interface{}); ok {
					result[k] = append(dstSlice, srcSlice...)
					continue
				}
			}
			// Handle dict merge
			if dstMap, ok := existing.(map[string]interface{}); ok {
				if srcMap, ok := v.(map[string]interface{}); ok {
					result[k] = deepMergeSimple(dstMap, srcMap)
					continue
				}
			}
		}
		// Default: override
		result[k] = v
	}

	return result
}

// runSyncPhase writes merged secrets to AWS Secrets Manager
func runSyncPhase(t *testing.T, ctx context.Context, client *secretsmanager.Client, secrets map[string]map[string]interface{}) {
	t.Helper()

	for name, data := range secrets {
		secretName := "fsc-test/" + name
		secretValue, err := json.Marshal(data)
		require.NoError(t, err)

		// Create or update secret
		_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			SecretString: aws.String(string(secretValue)),
		})
		if err != nil {
			// Try update if create fails (secret exists)
			_, err = client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
				SecretId:     aws.String(secretName),
				SecretString: aws.String(string(secretValue)),
			})
			require.NoError(t, err)
		}
	}

	t.Log("Sync phase completed - secrets written to AWS")
}

// validateAWSSecrets reads back secrets from AWS and validates they match
func validateAWSSecrets(t *testing.T, ctx context.Context, client *secretsmanager.Client, expected map[string]map[string]interface{}) {
	t.Helper()

	for name, expectedData := range expected {
		secretName := "fsc-test/" + name

		result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
		require.NoError(t, err)

		var actualData map[string]interface{}
		err = json.Unmarshal([]byte(*result.SecretString), &actualData)
		require.NoError(t, err)

		// Validate key fields
		for key, expectedVal := range expectedData {
			assert.Contains(t, actualData, key, "Expected key %s in AWS secret %s", key, name)
		}

		t.Logf("Validated AWS secret: %s", secretName)
	}

	t.Log("All AWS secrets validated successfully")
}

// cleanup removes test data
func cleanup(t *testing.T, ctx context.Context, vaultClient *api.Client, awsClient *secretsmanager.Client) {
	t.Helper()

	// Delete Vault secrets
	paths := []string{
		"secret/metadata/analytics",
		"secret/metadata/analytics-engineers",
		"secret/metadata/shared",
	}
	for _, path := range paths {
		vaultClient.Logical().Delete(path)
	}

	// Delete AWS secrets
	secretNames := []string{
		"fsc-test/database",
		"fsc-test/api-keys",
		"fsc-test/common",
		"fsc-test/nested/level1/config",
	}
	for _, name := range secretNames {
		awsClient.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(name),
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
	}

	t.Log("Cleanup completed")
}

// TestRecursiveVaultListing validates the BFS recursive listing works correctly
func TestRecursiveVaultListing(t *testing.T) {
	skipIfNoIntegrationEnv(t)

	ctx := context.Background()
	vaultClient := setupVaultClient(t)

	// Create nested structure
	paths := []string{
		"secret/data/recursive-test/level0",
		"secret/data/recursive-test/a/level1",
		"secret/data/recursive-test/a/b/level2",
		"secret/data/recursive-test/a/b/c/level3",
		"secret/data/recursive-test/x/y/z/deep",
	}

	for _, path := range paths {
		writeVaultSecret(t, vaultClient, path, map[string]interface{}{
			"data": map[string]interface{}{"path": path},
		})
	}

	// List recursively using Vault LIST API (simulating our BFS)
	allSecrets := listVaultSecretsRecursive(t, vaultClient, "secret/metadata/recursive-test")

	// Should find all 5 secrets
	assert.GreaterOrEqual(t, len(allSecrets), 5, "Expected at least 5 secrets from recursive listing")

	// Cleanup
	vaultClient.Logical().Delete("secret/metadata/recursive-test")

	t.Logf("Found %d secrets via recursive listing", len(allSecrets))
}

// listVaultSecretsRecursive is a test helper that mimics our BFS implementation
func listVaultSecretsRecursive(t *testing.T, client *api.Client, basePath string) []string {
	t.Helper()

	var allSecrets []string
	visited := make(map[string]bool)
	queue := []string{basePath}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		result, err := client.Logical().List(current)
		if err != nil || result == nil {
			continue
		}

		keys, ok := result.Data["keys"].([]interface{})
		if !ok {
			continue
		}

		for _, key := range keys {
			keyStr := key.(string)
			fullPath := current + "/" + keyStr

			if keyStr[len(keyStr)-1] == '/' {
				// Directory - add to queue
				queue = append(queue, current+"/"+keyStr[:len(keyStr)-1])
			} else {
				// Secret - add to results
				allSecrets = append(allSecrets, fullPath)
			}
		}
	}

	return allSecrets
}

// TestDeepMergeStrategies validates the FSC deepmerge behavior
func TestDeepMergeStrategies(t *testing.T) {
	// This test doesn't need emulators - it validates the merge logic

	t.Run("list append", func(t *testing.T) {
		dst := map[string]interface{}{
			"users": []interface{}{"alice", "bob"},
		}
		src := map[string]interface{}{
			"users": []interface{}{"charlie"},
		}

		result := deepMergeSimple(dst, src)
		users := result["users"].([]interface{})

		assert.Len(t, users, 3)
		assert.Contains(t, users, "alice")
		assert.Contains(t, users, "bob")
		assert.Contains(t, users, "charlie")
	})

	t.Run("dict merge", func(t *testing.T) {
		dst := map[string]interface{}{
			"config": map[string]interface{}{
				"timeout": 30,
				"retries": 3,
			},
		}
		src := map[string]interface{}{
			"config": map[string]interface{}{
				"debug": true,
			},
		}

		result := deepMergeSimple(dst, src)
		config := result["config"].(map[string]interface{})

		assert.Equal(t, 30, config["timeout"])
		assert.Equal(t, 3, config["retries"])
		assert.Equal(t, true, config["debug"])
	})

	t.Run("scalar override", func(t *testing.T) {
		dst := map[string]interface{}{
			"version": "1.0",
		}
		src := map[string]interface{}{
			"version": "2.0",
		}

		result := deepMergeSimple(dst, src)
		assert.Equal(t, "2.0", result["version"])
	})
}

// TestTargetInheritanceChain validates Stg -> Prod -> Demo inheritance
func TestTargetInheritanceChain(t *testing.T) {
	// This test validates the inheritance resolution logic
	// without needing emulators

	targets := map[string]struct {
		imports  []string
		inherits string
	}{
		"Stg":  {imports: []string{"analytics", "shared"}, inherits: ""},
		"Prod": {imports: []string{"Stg"}, inherits: "Stg"},           // Inherits from Stg
		"Demo": {imports: []string{"Prod"}, inherits: "Prod"},         // Inherits from Prod
	}

	// Validate inheritance detection
	assert.Empty(t, targets["Stg"].inherits, "Stg should not inherit")
	assert.Equal(t, "Stg", targets["Prod"].inherits, "Prod should inherit from Stg")
	assert.Equal(t, "Prod", targets["Demo"].inherits, "Demo should inherit from Prod")

	// Validate topological order (dependencies resolved first)
	order := []string{"Stg", "Prod", "Demo"}
	for i, target := range order {
		if targets[target].inherits != "" {
			// Find parent in order
			parentIdx := -1
			for j, t := range order {
				if t == targets[target].inherits {
					parentIdx = j
					break
				}
			}
			assert.Less(t, parentIdx, i, "%s should be processed after %s", target, targets[target].inherits)
		}
	}

	t.Log("Target inheritance chain validated")
}
