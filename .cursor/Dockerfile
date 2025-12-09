# Go development environment for SecretSync
FROM golang:1-bookworm

WORKDIR /workspace

# =============================================================================
# SHELL CONFIGURATION
# =============================================================================
# dump_bash_state - dumps bash command history from an active shell process
# Useful for debugging. See: https://www.commandlinefu.com/commands/view/11427
# Requires gdb to be installed
RUN echo 'dump_bash_state() { \
  local pid="${1:-$$}"; \
  local outfile="/tmp/bash_history-${pid}.txt"; \
  if command -v gdb &>/dev/null; then \
    gdb -batch --eval "attach $pid" --eval "call write_history(\"$outfile\")" --eval "detach" --eval "q" 2>/dev/null && \
    echo "History dumped to $outfile"; \
  else \
    echo "gdb not installed - cannot dump bash state"; \
  fi; \
}' >> /etc/bash.bashrc

# =============================================================================
# SYSTEM DEPENDENCIES
# =============================================================================
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Version control
    git \
    git-lfs \
    # GitHub CLI
    gh \
    # Task automation
    just \
    # Essential CLI tools
    ripgrep \
    jq \
    curl \
    wget \
    # Debugging
    gdb \
    && rm -rf /var/lib/apt/lists/*

# =============================================================================
# GOLANGCI-LINT
# =============================================================================
# Install latest golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin latest && \
    golangci-lint --version

# =============================================================================
# GIT LFS CONFIGURATION
# =============================================================================
# --skip-smudge prevents workspace crashes with large LFS repos
RUN git lfs install --system --skip-smudge

# Verify installations
RUN go version && golangci-lint --version && gh --version && rg --version && jq --version
