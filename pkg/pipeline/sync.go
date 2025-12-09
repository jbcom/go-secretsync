package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/jbcom/secretsync/api/v1alpha1"
	"github.com/jbcom/secretsync/internal/backend"
	"github.com/jbcom/secretsync/pkg/client/aws"
	"github.com/jbcom/secretsync/pkg/client/vault"
	log "github.com/sirupsen/logrus"
)

// syncTarget syncs merged secrets to AWS for a single target
func (p *Pipeline) syncTarget(ctx context.Context, targetName string, dryRun bool) Result {
	start := time.Now()
	l := log.WithFields(log.Fields{
		"action": "syncTarget",
		"target": targetName,
		"dryRun": dryRun,
	})

	target, ok := p.config.Targets[targetName]
	if !ok {
		return Result{
			Target:   targetName,
			Phase:    "sync",
			Success:  false,
			Error:    fmt.Errorf("target not found"),
			Duration: time.Since(start),
		}
	}

	roleARN := p.config.GetRoleARN(target.AccountID)

	// Determine source path based on merge store type
	var sourcePath string
	if p.config.MergeStore.Vault != nil {
		sourcePath = fmt.Sprintf("%s/%s", p.config.MergeStore.Vault.Mount, targetName)
	} else if p.s3Store != nil {
		sourcePath = p.s3Store.GetMergePath(targetName)
	} else {
		return Result{
			Target:   targetName,
			Phase:    "sync",
			Success:  false,
			Error:    fmt.Errorf("no merge store configured"),
			Duration: time.Since(start),
		}
	}

	region := target.Region
	if region == "" {
		region = p.config.AWS.Region
	}

	l.WithFields(log.Fields{
		"accountID":  target.AccountID,
		"roleARN":    roleARN,
		"sourcePath": sourcePath,
		"region":     region,
	}).Info("Starting sync to AWS")

	// Create and execute sync
	syncConfig := p.createAWSSync(targetName, sourcePath, roleARN, region, dryRun)

	if err := backend.AddSyncConfig(syncConfig); err != nil {
		return Result{
			Target:   targetName,
			Phase:    "sync",
			Success:  false,
			Error:    fmt.Errorf("failed to add sync config: %w", err),
			Duration: time.Since(start),
		}
	}

	if err := backend.ManualTrigger(ctx, syncConfig, logical.UpdateOperation); err != nil {
		return Result{
			Target:   targetName,
			Phase:    "sync",
			Success:  false,
			Error:    fmt.Errorf("failed to trigger sync: %w", err),
			Duration: time.Since(start),
		}
	}

	// TODO: The backend.ManualTrigger is async and doesn't provide a completion signal.
	// This sleep is a temporary workaround until we implement proper synchronization.
	// Future improvement: make backend operations synchronous or return a completion channel.
	time.Sleep(500 * time.Millisecond)

	l.WithField("duration", time.Since(start)).Info("Sync completed")

	result := Result{
		Target:    targetName,
		Phase:     "sync",
		Operation: string(OperationSync),
		Success:   true,
		Duration:  time.Since(start),
		Details: ResultDetails{
			SourcePaths:     []string{sourcePath},
			DestinationPath: fmt.Sprintf("aws:%s", target.AccountID),
			RoleARN:         roleARN,
		},
	}

	// Compute diff if tracking is enabled
	if p.pipelineDiff != nil {
		targetDiff, err := p.computeSyncDiff(ctx, targetName, roleARN, region)
		if err != nil {
			l.WithError(err).Debug("Failed to compute sync diff")
		} else {
			result.Diff = targetDiff
			p.addTargetDiff(*targetDiff)
			l.WithFields(log.Fields{
				"added":    targetDiff.Summary.Added,
				"removed":  targetDiff.Summary.Removed,
				"modified": targetDiff.Summary.Modified,
			}).Debug("Diff computed for sync")
		}
	}

	return result
}

// createAWSSync creates a SecretSync for syncing to AWS
func (p *Pipeline) createAWSSync(targetName, sourcePath, roleARN, region string, dryRun bool) v1alpha1.SecretSync {
	sync := v1alpha1.SecretSync{
		Spec: v1alpha1.SecretSyncSpec{
			DryRun:     boolPtr(dryRun),
			SyncDelete: boolPtr(p.config.Pipeline.Sync.DeleteOrphans),
			Source: &vault.VaultClient{
				Address:   p.config.Vault.Address,
				Namespace: p.config.Vault.Namespace,
				Path:      fmt.Sprintf("%s/(.*)", sourcePath),
			},
			Dest: []*v1alpha1.StoreConfig{
				{
					AWS: &aws.AwsClient{
						Name:    "$1",
						Region:  region,
						RoleArn: roleARN,
					},
				},
			},
		},
	}
	sync.Name = fmt.Sprintf("sync-%s", targetName)
	sync.Namespace = "pipeline"
	return sync
}
