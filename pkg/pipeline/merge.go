package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/jbcom/secretsync/api/v1alpha1"
	"github.com/jbcom/secretsync/internal/backend"
	"github.com/jbcom/secretsync/pkg/client/vault"
	log "github.com/sirupsen/logrus"
)

// mergeTarget executes merge operations for a single target
func (p *Pipeline) mergeTarget(ctx context.Context, targetName string, dryRun bool) Result {
	start := time.Now()
	l := log.WithFields(log.Fields{
		"action": "mergeTarget",
		"target": targetName,
		"dryRun": dryRun,
	})

	target, ok := p.config.Targets[targetName]
	if !ok {
		return Result{
			Target:   targetName,
			Phase:    "merge",
			Success:  false,
			Error:    fmt.Errorf("target not found"),
			Duration: time.Since(start),
		}
	}

	// Determine merge path based on merge store type
	var mergePath string
	if p.config.MergeStore.Vault != nil {
		mergePath = fmt.Sprintf("%s/%s", p.config.MergeStore.Vault.Mount, targetName)
	} else if p.s3Store != nil {
		mergePath = p.s3Store.GetMergePath(targetName)
	} else {
		return Result{
			Target:   targetName,
			Phase:    "merge",
			Success:  false,
			Error:    fmt.Errorf("no merge store configured"),
			Duration: time.Since(start),
		}
	}
	l.WithField("mergePath", mergePath).Info("Starting merge")

	var sourcePaths []string
	var failedImports []string
	var lastErr error
	successCount := 0

	for _, importName := range target.Imports {
		sourcePath := p.config.GetSourcePath(importName)
		sourcePaths = append(sourcePaths, sourcePath)

		l.WithFields(log.Fields{
			"import":     importName,
			"sourcePath": sourcePath,
		}).Debug("Processing import")

		// Use Vault merge store (standard path)
		if p.config.MergeStore.Vault != nil {
			syncConfig := p.createMergeSync(importName, targetName, sourcePath, mergePath, dryRun)

			if err := backend.AddSyncConfig(syncConfig); err != nil {
				l.WithError(err).WithField("import", importName).Error("Failed to add sync config")
				failedImports = append(failedImports, importName)
				lastErr = err
				continue
			}

			if err := backend.ManualTrigger(ctx, syncConfig, logical.UpdateOperation); err != nil {
				l.WithError(err).WithField("import", importName).Error("Failed to trigger merge")
				failedImports = append(failedImports, importName)
				lastErr = err
				continue
			}
		}

		// Use S3 merge store
		if p.s3Store != nil && !dryRun {
			secretData := map[string]interface{}{
				"_source":    importName,
				"_target":    targetName,
				"_timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if err := p.s3Store.WriteSecret(ctx, targetName, importName, secretData); err != nil {
				l.WithError(err).WithField("import", importName).Error("Failed to write to S3 merge store")
				failedImports = append(failedImports, importName)
				lastErr = err
				continue
			}
		}

		successCount++
	}

	// TODO: The backend.ManualTrigger is async and doesn't provide a completion signal.
	// This sleep is a temporary workaround until we implement proper synchronization.
	// The duration is proportional to the number of imports to account for processing time.
	// Future improvement: make backend operations synchronous or return a completion channel.
	if p.config.MergeStore.Vault != nil && successCount > 0 {
		waitDuration := time.Duration(successCount*300) * time.Millisecond
		if waitDuration > 5*time.Second {
			waitDuration = 5 * time.Second // Cap at 5 seconds
		}
		time.Sleep(waitDuration)
	}

	success := lastErr == nil
	l.WithFields(log.Fields{
		"duration":      time.Since(start),
		"success":       success,
		"failedImports": failedImports,
	}).Info("Merge completed")

	result := Result{
		Target:    targetName,
		Phase:     "merge",
		Operation: string(OperationMerge),
		Success:   success,
		Error:     lastErr,
		Duration:  time.Since(start),
		Details: ResultDetails{
			SecretsProcessed: successCount,
			SourcePaths:      sourcePaths,
			DestinationPath:  mergePath,
			FailedImports:    failedImports,
		},
	}

	// Compute diff if tracking is enabled
	if p.pipelineDiff != nil {
		targetDiff, err := p.computeMergeDiff(ctx, targetName, sourcePaths)
		if err != nil {
			l.WithError(err).Debug("Failed to compute merge diff")
		} else {
			result.Diff = targetDiff
			p.addTargetDiff(*targetDiff)
			l.WithFields(log.Fields{
				"added":    targetDiff.Summary.Added,
				"removed":  targetDiff.Summary.Removed,
				"modified": targetDiff.Summary.Modified,
			}).Debug("Diff computed for merge")
		}
	}

	return result
}

// createMergeSync creates a SecretSync for merging sources
func (p *Pipeline) createMergeSync(importName, targetName, sourcePath, mergePath string, dryRun bool) v1alpha1.SecretSync {
	sync := v1alpha1.SecretSync{
		Spec: v1alpha1.SecretSyncSpec{
			DryRun:     boolPtr(dryRun),
			SyncDelete: boolPtr(false),
			Source: &vault.VaultClient{
				Address:   p.config.Vault.Address,
				Namespace: p.config.Vault.Namespace,
				Path:      fmt.Sprintf("%s/(.*)", sourcePath),
			},
			Dest: []*v1alpha1.StoreConfig{
				{
					Vault: &vault.VaultClient{
						Address:   p.config.Vault.Address,
						Namespace: p.config.Vault.Namespace,
						Path:      fmt.Sprintf("%s/$1", mergePath),
						Merge:     true,
					},
				},
			},
		},
	}
	sync.Name = fmt.Sprintf("merge-%s-to-%s", importName, targetName)
	sync.Namespace = "pipeline"
	return sync
}
