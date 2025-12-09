package pipeline

import (
	"fmt"

	"github.com/jbcom/secretsync/api/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// GenerateConfigs generates SecretSync configs without executing them
// Useful for GitOps workflows or Kubernetes CRD generation
func (p *Pipeline) GenerateConfigs(opts Options) ([]v1alpha1.SecretSync, error) {
	var configs []v1alpha1.SecretSync

	if p.config.MergeStore.Vault == nil {
		log.Warn("GenerateConfigs only supports Vault merge store; S3 merge store operations are handled inline")
	}

	targets := p.resolveTargets(opts.Targets)

	// Generate merge configs (only for Vault merge store)
	if (opts.Operation == OperationMerge || opts.Operation == OperationPipeline) && p.config.MergeStore.Vault != nil {
		for _, targetName := range targets {
			target := p.config.Targets[targetName]
			mergePath := fmt.Sprintf("%s/%s", p.config.MergeStore.Vault.Mount, targetName)

			for _, importName := range target.Imports {
				sourcePath := p.config.GetSourcePath(importName)
				cfg := p.createMergeSync(importName, targetName, sourcePath, mergePath, opts.DryRun)
				configs = append(configs, cfg)
			}
		}
	}

	// Generate sync configs
	if opts.Operation == OperationSync || opts.Operation == OperationPipeline {
		for _, targetName := range targets {
			target := p.config.Targets[targetName]
			roleARN := p.config.GetRoleARN(target.AccountID)

			var sourcePath string
			if p.config.MergeStore.Vault != nil {
				sourcePath = fmt.Sprintf("%s/%s", p.config.MergeStore.Vault.Mount, targetName)
			} else if p.config.MergeStore.S3 != nil {
				log.WithField("target", targetName).Warn("S3 merge store sync requires custom handling")
				continue
			}

			region := target.Region
			if region == "" {
				region = p.config.AWS.Region
			}

			cfg := p.createAWSSync(targetName, sourcePath, roleARN, region, opts.DryRun)
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

// boolPtr is a helper to create a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
