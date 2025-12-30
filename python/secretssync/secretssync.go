// Package secretssync provides Python bindings for the secrets-sync pipeline.
//
// This package exposes the core secrets synchronization functionality
// for use from Python via gopy-generated bindings.
package secretssync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/extended-data-library/secrets-sync/pkg/diff"
	"github.com/extended-data-library/secrets-sync/pkg/pipeline"
)

// Version of the Python bindings
const Version = "0.1.0"

// PipelineConfig represents the pipeline configuration in a Python-friendly format
type PipelineConfig struct {
	Path string // Path to YAML configuration file
}

// SyncOptions configures pipeline execution
type SyncOptions struct {
	DryRun          bool   // If true, don't make actual changes
	Operation       string // "merge", "sync", or "pipeline"
	Targets         string // Comma-separated list of targets (empty for all)
	ContinueOnError bool   // Continue on errors
	Parallelism     int    // Number of parallel operations
	ComputeDiff     bool   // Compute and return diff
	OutputFormat    string // "human", "json", "github", "compact", "side-by-side"
}

// SyncResult represents the outcome of a sync operation
type SyncResult struct {
	Success          bool   // Overall success status
	TargetCount      int    // Number of targets processed
	SecretsProcessed int    // Total secrets processed
	SecretsAdded     int    // Secrets added
	SecretsModified  int    // Secrets modified
	SecretsRemoved   int    // Secrets removed
	SecretsUnchanged int    // Secrets unchanged
	DurationMs       int64  // Duration in milliseconds
	ErrorMessage     string // Error message if failed
	ResultsJSON      string // Full results as JSON
	DiffOutput       string // Diff output if computed
}

// DefaultSyncOptions returns sensible default options
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		DryRun:          false,
		Operation:       "pipeline",
		Targets:         "",
		ContinueOnError: false,
		Parallelism:     4,
		ComputeDiff:     false,
		OutputFormat:    "human",
	}
}

// NewPipelineConfig creates a new pipeline configuration from a file path
func NewPipelineConfig(path string) *PipelineConfig {
	return &PipelineConfig{Path: path}
}

// ValidateConfig validates a pipeline configuration file
func ValidateConfig(configPath string) (bool, string) {
	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return false, fmt.Sprintf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		return false, fmt.Sprintf("Invalid configuration: %v", err)
	}

	return true, "Configuration is valid"
}

// RunPipeline executes the secrets synchronization pipeline
func RunPipeline(configPath string, opts *SyncOptions) *SyncResult {
	result := &SyncResult{}
	startTime := time.Now()

	ctx := context.Background()

	// Load and validate configuration
	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to load config: %v", err)
		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Create pipeline
	p, err := pipeline.NewWithContext(ctx, cfg)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create pipeline: %v", err)
		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Parse options
	pipelineOpts := pipeline.DefaultOptions()
	pipelineOpts.DryRun = opts.DryRun
	pipelineOpts.ContinueOnError = opts.ContinueOnError
	pipelineOpts.ComputeDiff = opts.ComputeDiff

	if opts.Parallelism > 0 {
		pipelineOpts.Parallelism = opts.Parallelism
	}

	switch opts.Operation {
	case "merge":
		pipelineOpts.Operation = pipeline.OperationMerge
	case "sync":
		pipelineOpts.Operation = pipeline.OperationSync
	case "pipeline", "":
		pipelineOpts.Operation = pipeline.OperationPipeline
	default:
		result.ErrorMessage = fmt.Sprintf("Unknown operation: %s", opts.Operation)
		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Parse output format
	switch opts.OutputFormat {
	case "json":
		pipelineOpts.OutputFormat = diff.OutputFormatJSON
	case "github":
		pipelineOpts.OutputFormat = diff.OutputFormatGitHub
	case "compact":
		pipelineOpts.OutputFormat = diff.OutputFormatCompact
	case "side-by-side":
		pipelineOpts.OutputFormat = diff.OutputFormatSideBySide
	default:
		pipelineOpts.OutputFormat = diff.OutputFormatHuman
	}

	// Parse targets
	if opts.Targets != "" {
		targets := splitTargets(opts.Targets)
		pipelineOpts.Targets = targets
	}

	// Run pipeline
	results, err := p.Run(ctx, pipelineOpts)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Pipeline execution failed: %v", err)
	}

	// Process results
	result.TargetCount = len(results)
	result.Success = err == nil

	for _, r := range results {
		result.SecretsProcessed += r.Details.SecretsProcessed
		result.SecretsAdded += r.Details.SecretsAdded
		result.SecretsModified += r.Details.SecretsModified
		result.SecretsRemoved += r.Details.SecretsRemoved
		result.SecretsUnchanged += r.Details.SecretsUnchanged

		if !r.Success && result.Success {
			result.Success = false
			if r.Error != nil && result.ErrorMessage == "" {
				result.ErrorMessage = r.Error.Error()
			}
		}
	}

	// Serialize results to JSON
	if jsonBytes, err := json.Marshal(results); err == nil {
		result.ResultsJSON = string(jsonBytes)
	}

	// Get diff output if computed
	if opts.ComputeDiff {
		pipelineDiff := p.Diff()
		if pipelineDiff != nil {
			result.DiffOutput = pipelineDiff.Format(pipelineOpts.OutputFormat, false)
		}
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	return result
}

// DryRun performs a dry run of the pipeline and returns the diff
func DryRun(configPath string) *SyncResult {
	opts := DefaultSyncOptions()
	opts.DryRun = true
	opts.ComputeDiff = true
	return RunPipeline(configPath, opts)
}

// Merge runs only the merge phase of the pipeline
func Merge(configPath string, dryRun bool) *SyncResult {
	opts := DefaultSyncOptions()
	opts.Operation = "merge"
	opts.DryRun = dryRun
	opts.ComputeDiff = dryRun
	return RunPipeline(configPath, opts)
}

// Sync runs only the sync phase of the pipeline
func Sync(configPath string, dryRun bool) *SyncResult {
	opts := DefaultSyncOptions()
	opts.Operation = "sync"
	opts.DryRun = dryRun
	opts.ComputeDiff = dryRun
	return RunPipeline(configPath, opts)
}

// GetTargets returns the list of targets from a configuration
func GetTargets(configPath string) ([]string, string) {
	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Sprintf("Failed to load config: %v", err)
	}

	var targets []string
	for name := range cfg.Targets {
		targets = append(targets, name)
	}
	return targets, ""
}

// GetSources returns the list of sources from a configuration
func GetSources(configPath string) ([]string, string) {
	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Sprintf("Failed to load config: %v", err)
	}

	var sources []string
	for name := range cfg.Sources {
		sources = append(sources, name)
	}
	return sources, ""
}

// Helper to split comma-separated targets
func splitTargets(targets string) []string {
	if targets == "" {
		return nil
	}

	var result []string
	current := ""
	for _, c := range targets {
		if c == ',' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// ConfigInfo returns information about a configuration file
type ConfigInfo struct {
	Valid        bool     // Whether the configuration is valid
	ErrorMessage string   // Error message if invalid
	SourceCount  int      // Number of sources
	TargetCount  int      // Number of targets
	Sources      []string // List of source names
	Targets      []string // List of target names
	HasMergeStore bool    // Whether a merge store is configured
	VaultAddress string   // Vault address if configured
	AWSRegion    string   // AWS region if configured
}

// GetConfigInfo returns detailed information about a configuration
func GetConfigInfo(configPath string) *ConfigInfo {
	info := &ConfigInfo{}

	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		info.ErrorMessage = fmt.Sprintf("Failed to load config: %v", err)
		return info
	}

	if err := cfg.Validate(); err != nil {
		info.ErrorMessage = fmt.Sprintf("Invalid configuration: %v", err)
		return info
	}

	info.Valid = true
	info.SourceCount = len(cfg.Sources)
	info.TargetCount = len(cfg.Targets)
	info.VaultAddress = cfg.Vault.Address
	info.AWSRegion = cfg.AWS.Region
	info.HasMergeStore = cfg.MergeStore.Vault != nil || cfg.MergeStore.S3 != nil

	for name := range cfg.Sources {
		info.Sources = append(info.Sources, name)
	}
	for name := range cfg.Targets {
		info.Targets = append(info.Targets, name)
	}

	return info
}
