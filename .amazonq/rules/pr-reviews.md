# Amazon Q PR Review Rules for secretsync

## ⛔ CRITICAL: Version Validation Policy

**DO NOT** make claims about software version validity based on training data.

### Forbidden Actions

- Claiming any software version "doesn't exist"
- Suggesting version downgrades because training data is stale
- Asserting "latest stable version" without live API verification
- Making compatibility claims based on version assumptions

### Required Behavior

1. **Trust project files**: The versions in `go.mod`, `Dockerfile`, and config files are intentional
2. **No stale data claims**: If you cannot verify via Docker Hub API, GitHub API, or official sources, DO NOT comment on version validity
3. **Acceptable feedback**: "Consider verifying compatibility" or "Pin to patch version for reproducibility"

### Why This Matters

Training data cutoffs mean you may not know about Go 1.25, LocalStack 3.8, Vault 1.17, etc. Making claims that these "don't exist" wastes maintainer time and demonstrates unfitness for review.

### Consequence

Reviews claiming version invalidity based on training data are **auto-rejected** regardless of whether the claim happens to be correct.
