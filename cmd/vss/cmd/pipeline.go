package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/robertlestak/vault-secret-sync/pkg/pipeline"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

var (
	targets   string
	mergeOnly bool
	syncOnly  bool
	dryRun    bool
)

// pipelineCmd runs the full merge-then-sync pipeline
var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Run the full secrets pipeline (merge → sync)",
	Long: `Runs the complete secrets synchronization pipeline:

1. MERGE PHASE: Aggregate secrets from sources into the merge store
   - Processes targets in dependency order (base before derived)
   - Supports inheritance (Prod inherits from Stg)
   - Uses Vault merge mode for aggregation

2. SYNC PHASE: Sync merged secrets to target AWS accounts
   - Assumes Control Tower execution role in each account
   - Runs in parallel (respects --parallel setting)

Examples:
  # Full pipeline
  vss pipeline --config config.yaml

  # Dry run
  vss pipeline --config config.yaml --dry-run

  # Specific targets only
  vss pipeline --config config.yaml --targets "Serverless_Stg,Serverless_Prod"

  # Merge only (no AWS sync)
  vss pipeline --config config.yaml --merge-only`,
	RunE: runPipeline,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)

	pipelineCmd.Flags().StringVar(&targets, "targets", "", "comma-separated list of targets (default: all)")
	pipelineCmd.Flags().BoolVar(&mergeOnly, "merge-only", false, "only run merge phase")
	pipelineCmd.Flags().BoolVar(&syncOnly, "sync-only", false, "only run sync phase")
	pipelineCmd.Flags().BoolVar(&dryRun, "dry-run", false, "dry run mode (no changes)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	l := log.WithFields(log.Fields{
		"action": "runPipeline",
	})

	// Create pipeline from config file
	p, err := pipeline.NewFromFile(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		l.Warn("Received shutdown signal")
		cancel()
	}()

	// Parse targets
	var targetList []string
	if targets != "" {
		targetList = strings.Split(targets, ",")
		for i := range targetList {
			targetList[i] = strings.TrimSpace(targetList[i])
		}
	}

	// Determine operation
	op := pipeline.OperationPipeline
	if mergeOnly {
		op = pipeline.OperationMerge
	} else if syncOnly {
		op = pipeline.OperationSync
	}

	// Run options
	opts := pipeline.Options{
		Operation:       op,
		Targets:         targetList,
		DryRun:          dryRun,
		ContinueOnError: true,
	}

	l.WithFields(log.Fields{
		"config":    cfgFile,
		"targets":   targetList,
		"operation": op,
		"dryRun":    dryRun,
	}).Info("Starting pipeline")

	// Run pipeline
	results, err := p.Run(ctx, opts)

	// Print summary
	printResults(results)

	if err != nil {
		return err
	}

	// Check for any failures
	for _, r := range results {
		if !r.Success {
			return fmt.Errorf("pipeline completed with errors")
		}
	}

	l.Info("Pipeline completed successfully")
	return nil
}

func printResults(results []pipeline.Result) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Pipeline Results")
	fmt.Println(strings.Repeat("=", 60))

	var mergeResults, syncResults []pipeline.Result
	for _, r := range results {
		if r.Phase == "merge" {
			mergeResults = append(mergeResults, r)
		} else {
			syncResults = append(syncResults, r)
		}
	}

	if len(mergeResults) > 0 {
		fmt.Println("\nMerge Phase:")
		for _, r := range mergeResults {
			status := "✅"
			if !r.Success {
				status = "❌"
			}
			fmt.Printf("  %s %s (%.2fs)\n", status, r.Target, r.Duration.Seconds())
			if r.Error != nil {
				fmt.Printf("      Error: %v\n", r.Error)
			}
		}
	}

	if len(syncResults) > 0 {
		fmt.Println("\nSync Phase:")
		for _, r := range syncResults {
			status := "✅"
			if !r.Success {
				status = "❌"
			}
			fmt.Printf("  %s %s (%.2fs)\n", status, r.Target, r.Duration.Seconds())
			if r.Error != nil {
				fmt.Printf("      Error: %v\n", r.Error)
			}
		}
	}

	// Count successes/failures
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	fmt.Printf("\nTotal: %d/%d succeeded\n", successCount, len(results))
	fmt.Println(strings.Repeat("=", 60))
}
