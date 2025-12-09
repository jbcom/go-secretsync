package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

// Build information set via ldflags at build time
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "secretsync",
	Short: "SecretSync - Multi-account secrets management",
	Long: `SecretSync synchronizes secrets from Vault to AWS across multiple accounts.

It supports:
- AWS Control Tower / Organizations for multi-account management
- Inheritance hierarchies (dev → staging → prod)
- Dynamic target discovery via Identity Center / Organizations
- Merge stores for centralized secret aggregation

Configuration via environment variables:
  SECRETSYNC_CONFIG       - Path to config file (default: config.yaml)
  SECRETSYNC_LOG_LEVEL    - Log level: debug, info, warn, error (default: info)
  SECRETSYNC_LOG_FORMAT   - Log format: text, json (default: text)
  SECRETSYNC_TARGETS      - Comma-separated list of targets
  SECRETSYNC_DRY_RUN      - Dry run mode (default: false)
  SECRETSYNC_MERGE_ONLY   - Only run merge phase (default: false)
  SECRETSYNC_SYNC_ONLY    - Only run sync phase (default: false)
  SECRETSYNC_DISCOVER     - Enable dynamic target discovery (default: false)
  SECRETSYNC_OUTPUT       - Output format: human, json, github, compact (default: human)
  SECRETSYNC_DIFF         - Show diff even without dry-run (default: false)
  SECRETSYNC_EXIT_CODE    - Use exit codes for CI (default: false)

Examples:
  # Run full pipeline
  secretsync pipeline --config config.yaml

  # Dry run for specific targets
  secretsync pipeline --config config.yaml --targets Serverless_Stg --dry-run

  # Merge only (no AWS sync)
  secretsync pipeline --config config.yaml --merge-only

  # Using environment variables (no shell script needed)
  SECRETSYNC_CONFIG=config.yaml SECRETSYNC_DRY_RUN=true secretsync pipeline

  # Validate configuration
  secretsync validate --config config.yaml

  # Show dependency graph
  secretsync graph --config config.yaml`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set log level from viper (supports both flag and env var)
		logLevel := viper.GetString("log-level")
		level, err := log.ParseLevel(logLevel)
		if err != nil {
			level = log.InfoLevel
		}
		log.SetLevel(level)

		// Set log format from viper
		if viper.GetString("log-format") == "json" {
			log.SetFormatter(&log.JSONFormatter{})
		}
	},
}

// Execute runs the root command
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags - all bound to viper for env var support
	rootCmd.PersistentFlags().String("config", "config.yaml", "config file path")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "text", "log format (text, json)")

	// Bind all flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))
}

func initConfig() {
	// Environment variables with SECRETSYNC_ prefix
	viper.SetEnvPrefix("SECRETSYNC")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Config file handling
	cfgFile := viper.GetString("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config locations
		viper.AddConfigPath(".")
		viper.AddConfigPath("/config")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
}
