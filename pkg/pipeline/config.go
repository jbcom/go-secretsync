// Package pipeline provides unified configuration and orchestration for secrets syncing pipelines.
// It supports AWS Control Tower / Organizations patterns for multi-account secrets management.
package pipeline

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.applyDefaults()
	cfg.expandEnvVars()

	// Also load via Viper for env var override support
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("SECRETSYNC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if v.IsSet("log.level") {
		cfg.Log.Level = v.GetString("log.level")
	}
	if v.IsSet("aws.region") {
		cfg.AWS.Region = v.GetString("aws.region")
	}

	return &cfg, nil
}

// applyDefaults sets default values for unset fields
func (c *Config) applyDefaults() {
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}
	if c.AWS.Region == "" {
		c.AWS.Region = "us-east-1"
	}
	if c.AWS.ControlTower.ExecutionRole.Name == "" {
		c.AWS.ControlTower.ExecutionRole.Name = "AWSControlTowerExecution"
	}
	if c.Pipeline.Merge.Parallel <= 0 {
		c.Pipeline.Merge.Parallel = 4
	}
	if c.Pipeline.Sync.Parallel <= 0 {
		c.Pipeline.Sync.Parallel = 4
	}
}

// expandEnvVars expands ${VAR} patterns in config values
func (c *Config) expandEnvVars() {
	envPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	const maxEnvValueLength = 10000

	expand := func(s string) string {
		return envPattern.ReplaceAllStringFunc(s, func(match string) string {
			varName := match[2 : len(match)-1]
			if val := os.Getenv(varName); val != "" {
				if len(val) > maxEnvValueLength {
					log.WithField("variable", varName).Warn("Environment variable value exceeds maximum length, keeping placeholder")
					return match
				}
				return val
			}
			return match
		})
	}

	if c.Vault.Auth.AppRole != nil {
		c.Vault.Auth.AppRole.RoleID = expand(c.Vault.Auth.AppRole.RoleID)
		c.Vault.Auth.AppRole.SecretID = expand(c.Vault.Auth.AppRole.SecretID)
	}
	if c.Vault.Auth.Token != nil {
		c.Vault.Auth.Token.Token = expand(c.Vault.Auth.Token.Token)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Vault.Address == "" {
		return fmt.Errorf("vault.address is required")
	}

	if c.MergeStore.Vault == nil && c.MergeStore.S3 == nil {
		return fmt.Errorf("merge_store must specify either vault or s3")
	}

	if c.MergeStore.S3 != nil {
		if c.MergeStore.S3.Bucket == "" {
			return fmt.Errorf("merge_store.s3.bucket is required")
		}
	}

	if len(c.Targets) == 0 && len(c.DynamicTargets) == 0 {
		return fmt.Errorf("at least one target or dynamic_target is required")
	}

	for name, target := range c.Targets {
		if target.AccountID == "" {
			return fmt.Errorf("target %q: account_id is required", name)
		}
		if !isValidAWSAccountID(target.AccountID) {
			return fmt.Errorf("target %q: invalid account_id format %q (must be 12 digits)", name, target.AccountID)
		}
		for _, imp := range target.Imports {
			if _, ok := c.Sources[imp]; !ok {
				if _, ok := c.Targets[imp]; !ok {
					return fmt.Errorf("target %q: import %q not found in sources or targets", name, imp)
				}
			}
		}
	}

	if err := c.ValidateTargetInheritance(); err != nil {
		return err
	}

	for name, dt := range c.DynamicTargets {
		if dt.Discovery.IdentityCenter == nil && dt.Discovery.Organizations == nil && dt.Discovery.AccountsList == nil {
			return fmt.Errorf("dynamic_target %q: must specify identity_center, organizations, or accounts_list discovery", name)
		}
	}

	return nil
}

// WriteConfig writes the configuration to a file
func (c *Config) WriteConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// isValidAWSAccountID validates that an AWS account ID is exactly 12 digits
func isValidAWSAccountID(accountID string) bool {
	if len(accountID) != 12 {
		return false
	}
	for _, c := range accountID {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
