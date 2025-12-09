// Package pipeline provides a unified secrets synchronization pipeline that works
// identically across all execution modes: CLI, Kubernetes, and Vault event-driven.
//
// Architecture:
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                         Pipeline Configuration                          │
//	│  (YAML file, Kubernetes CRD, ConfigMap, or programmatic)               │
//	└─────────────────────────────────────────────────────────────────────────┘
//	                                    │
//	                                    ▼
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                           Pipeline Engine                               │
//	│  • Dependency graph resolution                                          │
//	│  • Topological ordering                                                 │
//	│  • Parallel execution within levels                                     │
//	│  • Operation modes: merge, sync, pipeline (merge+sync)                  │
//	└─────────────────────────────────────────────────────────────────────────┘
//	                                    │
//	          ┌─────────────────────────┼─────────────────────────┐
//	          ▼                         ▼                         ▼
//	┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
//	│   CLI Runner    │       │  K8s Operator   │       │  Vault Events   │
//	│  (secretsync pipeline) │       │  (CRD watch)    │       │  (webhook)      │
//	└─────────────────┘       └─────────────────┘       └─────────────────┘
//
// Operations:
//   - merge:    Source stores → Merge store (with inheritance resolution)
//   - sync:     Merge store → Destination stores (AWS, etc.)
//   - pipeline: merge + sync in correct dependency order
package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jbcom/secretsync/api/v1alpha1"
	"github.com/jbcom/secretsync/internal/backend"
	"github.com/jbcom/secretsync/internal/queue"
	internalSync "github.com/jbcom/secretsync/internal/sync"
	"github.com/jbcom/secretsync/pkg/client/aws"
	"github.com/jbcom/secretsync/pkg/client/vault"
	"github.com/jbcom/secretsync/pkg/diff"
	log "github.com/sirupsen/logrus"
)

// Operation defines what the pipeline should do
type Operation string

const (
	// OperationMerge only performs the merge phase (sources → merge store)
	OperationMerge Operation = "merge"
	// OperationSync only performs the sync phase (merge store → destinations)
	OperationSync Operation = "sync"
	// OperationPipeline performs both merge and sync in order
	OperationPipeline Operation = "pipeline"
)

// Pipeline is the main orchestrator for secrets synchronization
type Pipeline struct {
	config      *Config
	graph       *Graph
	initialized bool
	mu          sync.Mutex

	awsCtx  *AWSExecutionContext
	s3Store *S3MergeStore

	results   []Result
	resultsMu sync.Mutex

	pipelineDiff *diff.PipelineDiff
	diffMu       sync.Mutex
}

// Options configures pipeline execution
type Options struct {
	Operation       Operation
	Targets         []string
	DryRun          bool
	ContinueOnError bool
	Parallelism     int
	ComputeDiff     bool
	OutputFormat    diff.OutputFormat
}

// DefaultOptions returns sensible default options
func DefaultOptions() Options {
	return Options{
		Operation:       OperationPipeline,
		DryRun:          false,
		ContinueOnError: false,
		Parallelism:     4,
		ComputeDiff:     false,
	}
}

// Result represents the outcome of a single target operation
type Result struct {
	Target    string           `json:"target"`
	Phase     string           `json:"phase"`
	Operation string           `json:"operation"`
	Success   bool             `json:"success"`
	Error     error            `json:"error,omitempty"`
	Duration  time.Duration    `json:"duration"`
	Details   ResultDetails    `json:"details,omitempty"`
	Diff      *diff.TargetDiff `json:"diff,omitempty"`
}

// ResultDetails contains additional information about the operation
type ResultDetails struct {
	SecretsProcessed int      `json:"secrets_processed,omitempty"`
	SecretsAdded     int      `json:"secrets_added,omitempty"`
	SecretsModified  int      `json:"secrets_modified,omitempty"`
	SecretsRemoved   int      `json:"secrets_removed,omitempty"`
	SecretsUnchanged int      `json:"secrets_unchanged,omitempty"`
	SourcePaths      []string `json:"source_paths,omitempty"`
	DestinationPath  string   `json:"destination_path,omitempty"`
	RoleARN          string   `json:"role_arn,omitempty"`
	FailedImports    []string `json:"failed_imports,omitempty"`
}

// New creates a new Pipeline from configuration
func New(cfg *Config) (*Pipeline, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	graph, err := BuildGraph(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	return &Pipeline{
		config: cfg,
		graph:  graph,
	}, nil
}

// NewWithContext creates a new Pipeline with AWS execution context
func NewWithContext(ctx context.Context, cfg *Config) (*Pipeline, error) {
	p, err := New(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize AWS execution context if configured
	if cfg.AWS.ExecutionContext.Type != "" {
		awsCtx, err := NewAWSExecutionContext(ctx, &cfg.AWS)
		if err != nil {
			log.WithError(err).Warn("Failed to initialize AWS execution context")
		} else {
			p.awsCtx = awsCtx
		}
	}

	// Initialize S3 merge store if configured
	if cfg.MergeStore.S3 != nil {
		s3Store, err := NewS3MergeStore(ctx, cfg.MergeStore.S3, cfg.AWS.Region)
		if err != nil {
			log.WithError(err).Warn("Failed to initialize S3 merge store")
		} else {
			p.s3Store = s3Store
		}
	}

	return p, nil
}

// NewFromFile creates a Pipeline from a configuration file
func NewFromFile(path string) (*Pipeline, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// NewFromFileWithContext creates a Pipeline from a configuration file with context
func NewFromFileWithContext(ctx context.Context, path string) (*Pipeline, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return NewWithContext(ctx, cfg)
}

// Run executes the pipeline with the given options
func (p *Pipeline) Run(ctx context.Context, opts Options) ([]Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	l := log.WithFields(log.Fields{
		"action":    "Pipeline.Run",
		"operation": opts.Operation,
		"dryRun":    opts.DryRun,
	})

	if err := p.initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline: %w", err)
	}

	p.resultsMu.Lock()
	p.results = nil
	p.resultsMu.Unlock()

	if opts.DryRun || opts.ComputeDiff {
		p.initDiff(opts.DryRun, "")
	}

	targets := p.resolveTargets(opts.Targets)
	l.WithField("targets", targets).Info("Starting pipeline execution")

	if opts.Parallelism <= 0 {
		opts.Parallelism = p.config.Pipeline.Merge.Parallel
		if opts.Parallelism <= 0 {
			opts.Parallelism = 4
		}
	}

	switch opts.Operation {
	case OperationMerge:
		return p.runMerge(ctx, targets, opts)
	case OperationSync:
		return p.runSync(ctx, targets, opts)
	case OperationPipeline:
		return p.runPipeline(ctx, targets, opts)
	default:
		return nil, fmt.Errorf("unknown operation: %s", opts.Operation)
	}
}

// initialize sets up the sync infrastructure
func (p *Pipeline) initialize(ctx context.Context) error {
	if p.initialized {
		return nil
	}

	l := log.WithField("action", "Pipeline.initialize")
	l.Debug("Initializing pipeline infrastructure")

	backend.ManualTrigger = internalSync.ManualTrigger

	if queue.Q == nil {
		if err := queue.Init(queue.QueueTypeMemory, nil); err != nil {
			return fmt.Errorf("failed to initialize queue: %w", err)
		}
	}

	p.setDefaultStores()

	// Use a channel to signal when the event processor is ready
	ready := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		workerPoolSize := p.config.Pipeline.Merge.Parallel
		if workerPoolSize <= 0 {
			workerPoolSize = 4
		}
		// Signal ready before entering the event loop
		close(ready)
		if err := internalSync.EventProcessor(ctx, workerPoolSize, workerPoolSize); err != nil {
			l.WithError(err).Error("Event processor exited")
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for the event processor to be ready or timeout
	select {
	case <-ready:
		// Event processor goroutine has started
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for event processor to start")
	}

	p.initialized = true
	l.Info("Pipeline infrastructure initialized")
	return nil
}

// setDefaultStores configures default store settings
func (p *Pipeline) setDefaultStores() {
	stores := &v1alpha1.StoreConfig{
		Vault: &vault.VaultClient{
			Address:   p.config.Vault.Address,
			Namespace: p.config.Vault.Namespace,
		},
		AWS: &aws.AwsClient{
			Region: p.config.AWS.Region,
		},
	}
	internalSync.SetStoreDefaults(stores)
}

// resolveTargets returns the targets to process, including dependencies
func (p *Pipeline) resolveTargets(requested []string) []string {
	if len(requested) == 0 {
		return p.graph.TopologicalOrder()
	}
	return p.graph.IncludeDependencies(requested)
}

// Config returns the pipeline configuration
func (p *Pipeline) Config() *Config {
	return p.config
}

// Graph returns the dependency graph
func (p *Pipeline) Graph() *Graph {
	return p.graph
}

// Results returns the results from the last Run
func (p *Pipeline) Results() []Result {
	p.resultsMu.Lock()
	defer p.resultsMu.Unlock()
	return p.results
}

// Diff returns the computed diff from the last Run
func (p *Pipeline) Diff() *diff.PipelineDiff {
	p.diffMu.Lock()
	defer p.diffMu.Unlock()
	return p.pipelineDiff
}
