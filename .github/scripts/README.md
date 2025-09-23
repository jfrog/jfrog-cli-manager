# GitHub Actions Scripts

This directory contains reusable scripts extracted from GitHub Actions workflows to improve maintainability and reduce code duplication.

## Scripts

### `validate-tag.sh`
**Purpose**: Validates and determines the release tag from workflow inputs or git refs.

**Usage**: 
```bash
env GITHUB_EVENT_NAME="workflow_dispatch" INPUT_TAG="v1.0.0" ./validate-tag.sh
```

**Environment Variables**:
- `GITHUB_EVENT_NAME`: Type of GitHub event (workflow_dispatch or push)
- `INPUT_TAG`: Manual tag input (for workflow_dispatch)
- `GITHUB_REF`: Git reference (for push events)

**Outputs**: Sets `tag` in `$GITHUB_OUTPUT`

### `build-binary.sh`
**Purpose**: Builds the jfvm binary for a specific platform and architecture.

**Usage**: 
```bash
./build-binary.sh <version> <goos> <goarch>
```

**Arguments**:
- `version`: Release version (e.g., v1.0.0)
- `goos`: Target OS (linux, darwin, windows)
- `goarch`: Target architecture (amd64, arm64)

**Outputs**: Sets `binary_path` and `binary_size` in `$GITHUB_OUTPUT`

### `package-artifacts.sh`
**Purpose**: Packages binaries into archives and generates checksums.

**Usage**: 
```bash
./package-artifacts.sh <version> <goos> <goarch> <binary_path>
```

**Arguments**:
- `version`: Release version
- `goos`: Target OS
- `goarch`: Target architecture  
- `binary_path`: Path to the built binary

**Outputs**: Creates packaged files and SHA256 checksums

### `generate-homebrew-formula.sh`
**Purpose**: Generates a Homebrew formula with platform-specific checksums.

**Usage**: 
```bash
./generate-homebrew-formula.sh <tag> <repo> <darwin_amd_sha> <darwin_arm_sha> <linux_amd_sha> <linux_arm_sha>
```

**Arguments**:
- `tag`: Release tag
- `repo`: GitHub repository name
- Platform-specific SHA256 checksums

**Outputs**: Creates `jfvm.rb` formula file, sets `formula_size` in `$GITHUB_OUTPUT`

### `add-summary.sh`
**Purpose**: Adds formatted content to GitHub Actions Summary.

**Usage**: 
```bash
./add-summary.sh <type> <args...>
```

**Types**:
- `build <goos> <goarch> <size> <version> <date> <commit>`: Build results summary
- `release <version> <total_files> <release_url>`: Release summary
- `homebrew <version> <formula_size>`: Homebrew formula summary

## Benefits

1. **Maintainability**: Scripts are easier to test and modify independently
2. **Reusability**: Can be used across multiple workflows
3. **Clarity**: Workflow files focus on orchestration, not implementation
4. **Testing**: Scripts can be tested locally without running full workflows
5. **Version Control**: Changes to scripts are tracked separately from workflow changes

## Development

All scripts follow these conventions:
- Use `#!/bin/bash` and `set -euo pipefail` for safety
- Include descriptive comments and error messages
- Use consistent parameter validation
- Output status information with emojis for visibility
- Set appropriate outputs in `$GITHUB_OUTPUT` when needed
