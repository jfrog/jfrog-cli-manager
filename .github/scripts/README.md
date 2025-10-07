# GitHub Actions Scripts

Essential scripts for the jfcm release workflow.

## Scripts

### `validate-tag.sh`
Validates semantic version tag format.

```bash
env GITHUB_EVENT_NAME="workflow_dispatch" INPUT_TAG="v1.0.0" ./validate-tag.sh
```

### `build-binary.sh`
Builds jfcm binary for specific platform/architecture.

```bash
./build-binary.sh <version> <goos> <goarch>
```

### `package-artifacts.sh`
Creates archives and SHA256 checksums.

```bash
./package-artifacts.sh <version> <goos> <goarch> <binary_path>
```

### `generate-homebrew-formula.sh`
Generates Homebrew formula with checksums.

```bash
./generate-homebrew-formula.sh <tag> <repo> <darwin_amd_sha> <darwin_arm_sha> <linux_amd_sha> <linux_arm_sha>
```

## Usage

These scripts are used by:
- **Release workflow**: Automatic releases from tags
- **Publish Homebrew Formula**: Manual Homebrew publishing

All scripts use `set -euo pipefail` for safety and provide clear error messages.
