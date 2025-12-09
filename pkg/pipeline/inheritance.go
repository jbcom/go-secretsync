package pipeline

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ValidateTargetInheritance checks for circular dependencies in target inheritance chains
func (c *Config) ValidateTargetInheritance() error {
	for name := range c.Targets {
		visited := make(map[string]bool)
		recursionStack := make(map[string]bool)
		if err := c.detectCycle(name, visited, recursionStack); err != nil {
			return err
		}
	}
	return nil
}

// detectCycle performs DFS to detect circular dependencies in target inheritance
func (c *Config) detectCycle(targetName string, visited, recursionStack map[string]bool) error {
	visited[targetName] = true
	recursionStack[targetName] = true

	if target, ok := c.Targets[targetName]; ok {
		for _, imp := range target.Imports {
			if _, isTarget := c.Targets[imp]; isTarget {
				if !visited[imp] {
					if err := c.detectCycle(imp, visited, recursionStack); err != nil {
						return err
					}
				} else if recursionStack[imp] {
					return fmt.Errorf("circular dependency detected in target inheritance: %s -> %s", targetName, imp)
				}
			}
		}
	}

	recursionStack[targetName] = false
	return nil
}

// IsInheritedTarget checks if a target inherits from another target
func (c *Config) IsInheritedTarget(targetName string) bool {
	target, ok := c.Targets[targetName]
	if !ok {
		return false
	}
	for _, imp := range target.Imports {
		if _, isTarget := c.Targets[imp]; isTarget {
			return true
		}
	}
	return false
}

// GetSourcePath returns the full path for a source or inherited target
func (c *Config) GetSourcePath(importName string) string {
	if src, ok := c.Sources[importName]; ok {
		if src.Vault != nil {
			return src.Vault.Mount
		}
	}

	if _, ok := c.Targets[importName]; ok {
		if c.MergeStore.Vault != nil {
			return fmt.Sprintf("%s/%s", c.MergeStore.Vault.Mount, importName)
		}
	}

	log.WithField("import", importName).Warn("Unknown import - not found in sources or targets, using import name as path")
	return importName
}

// GetRoleARN returns the role ARN for a target account
func (c *Config) GetRoleARN(accountID string) string {
	for _, target := range c.Targets {
		if target.AccountID == accountID && target.RoleARN != "" {
			return target.RoleARN
		}
	}

	if c.AWS.ControlTower.Enabled {
		roleName := c.AWS.ControlTower.ExecutionRole.Name
		if roleName == "" {
			roleName = "AWSControlTowerExecution"
		}
		path := c.AWS.ControlTower.ExecutionRole.Path
		if path == "" {
			path = "/"
		} else {
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}
		}
		return fmt.Sprintf("arn:aws:iam::%s:role%s%s", accountID, path, roleName)
	}

	if c.AWS.ExecutionContext.CustomRolePattern != "" {
		return strings.ReplaceAll(c.AWS.ExecutionContext.CustomRolePattern, "{{.AccountID}}", accountID)
	}

	return fmt.Sprintf("arn:aws:iam::%s:role/AWSControlTowerExecution", accountID)
}
