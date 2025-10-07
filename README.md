# jfcm - JFrog CLI Version Manager

[![CI](https://github.com/jfrog/jfrog-cli-vm/actions/workflows/release.yml/badge.svg)](https://github.com/jfrog/jfrog-cli-vm/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/jfrog/jfrog-cli-vm)](https://github.com/jfrog/jfrog-cli-vm/releases)
[![License](https://img.shields.io/github/license/jfrog/jfrog-cli-vm)](https://github.com/jfrog/jfrog-cli-vm/blob/main/LICENSE)
[![homebrew installs](https://img.shields.io/badge/homebrew-installs-brightgreen?logo=homebrew)](https://github.com/jfrog/homebrew-jfrog-cli-vm)

**jfcm** is a powerful CLI tool that helps you manage multiple versions of the [JFrog CLI](https://jfrog.com/getcli/) on your system. It supports auto-installation, version switching, project-specific defaults, local binary linking, aliasing, parallel version comparison, performance benchmarking, and usage analytics — all inspired by tools like `nvm`, `sdkman`, and `volta`.

## 🎥 Demo

https://github.com/user-attachments/assets/6984077c-72ab-4f8c-a11c-671e72870efe

https://github.com/user-attachments/assets/32ce3eb1-4f69-49bd-bdc7-9f95cd9ead34


## 🚀 Why jfcm?

Managing different versions of the JFrog CLI across multiple projects and environments can be challenging. `jfcm` simplifies this by:

- Installing any released version of the `jf` binary
- Automatically fetching and using the latest version with `jfcm use latest`
- Allowing you to link locally built versions
- Automatically switching versions based on a `.jfrog-version` file
- Letting you define named aliases (e.g., `prod`, `dev`)
- Providing a smooth `jf` shim for command redirection
- Parallel command comparison between versions with diff visualization
- Performance benchmarking across multiple versions
- Usage history tracking and analytics

No more symlink hacking or hardcoded paths.

---

## 🛠️ Installation

### Via Homebrew (with tap):
```bash
brew tap jfrog/jfrog-cli-vm
brew install jfcm
```

### Via one-liner:
```bash
brew install https://raw.githubusercontent.com/jfrog/homebrew-jfrog-cli-vm/main/Formula/jfcm.rb
```

### Or Build From Source:
```bash
git clone https://github.com/jfrog/jfrog-cli-vm.git
cd jfrog-cli-vm
make install
```

**Note**: Use `make build` instead of `go build` to ensure the executable is named `jfcm` (not `jfrog-cli-vm`).

---

## 📦 Commands

### Core Version Management

#### `jfcm install <version>`
Installs the specified version of JFrog CLI (`jf`) from JFrog's public release server.
```bash
jfcm install 2.74.0
```

#### `jfcm use <version or alias>`
Activates the given version or alias. If `.jfrog-version` exists in the current directory, that will be used if no argument is passed. Use `latest` to automatically fetch and activate the most recent JFrog CLI version (downloads if not already installed). Automatically sets up PATH priority so jfcm-managed `jf` takes precedence over system-installed versions.
```bash
jfcm use 2.74.0
jfcm use latest
jfcm use prod
```

#### `jfcm list`
Shows all installed versions and the currently active one.
```bash
jfcm list
```

#### `jfcm remove <version>`
Removes a specific version of `jf`.
```bash
jfcm remove 2.72.1
```

#### `jfcm clear`
Removes **all** installed versions.
```bash
jfcm clear
```

#### `jfcm alias <n> <version>`
Defines an alias for a specific version.
```bash
jfcm alias dev 2.74.0
```

#### `jfcm link --from <path> --name <n>`
Links a **locally built `jf` binary** to be used via `jfcm`.
```bash
jfcm link --from /Users/Jfrog/go/bin/jf --name local-dev
jfcm use local-dev
```

#### `jfcm health-check`
Performs comprehensive health check of jfcm installation with various options.
```bash
# Basic health check
jfcm health-check

# Detailed health check with verbose output
jfcm health-check --verbose

# Health check with automatic fixes
jfcm health-check --fix

# Include performance and security checks
jfcm health-check --performance --security

# All options combined
jfcm health-check --verbose --fix --performance --security
```



### Advanced Features

#### `jfcm compare <subcommand>`
Compare JFrog CLI versions with specialized subcommands for different comparison types.

##### CLI Command Comparison (`jfcm compare cli`)
Compare JFrog CLI command output between two versions in parallel with git-like diff visualization.

```bash
# Compare version output
jfcm compare cli 2.74.0 2.73.0 -- --version

# Compare command outputs with side-by-side diff
jfcm compare cli prod dev -- rt ping

# Show unified diff format
jfcm compare cli 2.74.0 2.73.0 --unified -- config show

# Disable colored output and timing
jfcm compare cli old new --no-color --timing -- rt search "*.jar"
```

##### Server Comparison (`jfcm compare rt`)
Compare JFrog CLI command execution between different server configurations.

```bash
# Compare rt ping command across two servers
jfcm compare rt server1 server2 -- rt ping

# Compare search results across server configurations
jfcm compare rt prod dev -- rt search "*.jar"

# Show unified diff format for server comparison
jfcm compare rt server1 server2 -- config show --unified

# Server comparison with custom timeout
jfcm compare rt main backup -- rt repos show --timeout 60
```

##### Changelog Comparison (`jfcm compare changelog`)
Compare release notes and changelogs between two JFrog CLI versions.

```bash
# Compare release notes between versions
jfcm compare changelog v2.75.1 v2.76.0

# Compare changelogs with custom options
jfcm compare changelog v2.74.0 v2.73.0 --no-color --timeout 60
```

**Features:**
- Parallel execution for faster results
- Side-by-side and unified diff formats
- Colored output highlighting differences
- Execution timing comparison
- Exit code and error output comparison

#### `jfcm benchmark <versions> -- <command>`
Run performance benchmarks across multiple JFrog CLI versions with detailed statistics.

```bash
# Benchmark across multiple versions
jfcm benchmark 2.74.0,2.73.0,2.72.0 -- --version

# Custom iterations and detailed output
jfcm benchmark prod,dev,latest -- rt ping --iterations 10 --detailed

# Export results as JSON or CSV
jfcm benchmark 2.74.0,2.73.0 -- config show --format json
jfcm benchmark 2.74.0,2.73.0 -- rt search "*.jar" --format csv
```

**Features:**
- Configurable iteration counts
- Statistical analysis (min, max, average, success rate)
- Multiple output formats (table, JSON, CSV)
- Parallel execution across versions
- Detailed execution logs
- Performance ranking and speed comparisons

#### `jfcm history`
Track and analyze version usage patterns with comprehensive statistics.

```bash
# Show recent usage history
jfcm history

# Show detailed statistics
jfcm history --stats

# Filter by specific version
jfcm history --version 2.74.0

# Limit number of entries
jfcm history --limit 20

# Export as JSON
jfcm history --format json

# Clear history (cannot be undone)
jfcm history --clear

# Reexecute a specific history entry by ID
jfcm history '!2'  # Reexecute history entry with ID 2
jfcm history '!5'  # Reexecute history entry with ID 5
```

**Features:**
- Automatic usage tracking through the shim
- Command execution timing
- Most used versions and commands
- Usage trends and timeline analysis
- Configurable history limits
- **History replay**: Reexecute any previous command using `!{id}` syntax

---

## 📁 Project-specific Version

Add a `.jfrog-version` file to your repo:
```bash
echo "2.74.0" > .jfrog-version
```
Then run:
```bash
jfcm use
```

---

## ⚙️ Shell Integration & Priority Management
jfcm automatically configures your shell to ensure jfcm-managed `jf` binaries have **highest priority** over system-installed versions. When you run `jfcm use <version>`, it:

1. **Creates a shim** at `~/.jfcm/shim/jf` that redirects to the active version
2. **Updates your PATH** to prioritize the jfcm shim directory (prepends to PATH)
3. **Adds a shell function** for enhanced priority handling (similar to nvm)
4. **Verifies priority** to ensure jfcm-managed versions take precedence over Homebrew or system-installed jf

The configuration is automatically added to your shell profile (`.zshrc`, `.bashrc`, etc.):
```bash
# jfcm PATH configuration - ensures jfcm-managed jf takes highest priority
export PATH="$HOME/.jfcm/shim:$PATH"

# jfcm shell function for enhanced priority (similar to nvm approach)
jf() {
    # Check if jfcm shim exists and is executable
    if [ -x "$HOME/.jfcm/shim/jf" ]; then
        # Execute jfcm-managed jf with highest priority
        "$HOME/.jfcm/shim/jf" "$@"
    else
        # Fallback to system jf if jfcm shim not available
        command jf "$@"
    fi
}
```



### Debug Mode
Set `jfcm_DEBUG=1` to see detailed shim execution information:
```bash
export jfcm_DEBUG=1
# Will show which version is being executed
jf --version
```

### Troubleshooting PATH Issues

If `jf` is still using the system version instead of jfcm-managed version:

1. **Run the health check command:**
   ```bash
   jfcm health-check --fix
   # This will verify all aspects of jfcm setup and attempt to fix issues
   ```

2. **Check which jf is being used:**
   ```bash
   which jf
   # Should show: /Users/username/.jfcm/shim/jf
   ```

3. **Verify PATH order:**
   ```bash
   echo $PATH
   # ~/.jfcm/shim should appear before /usr/local/bin or /opt/homebrew/bin
   ```

4. **Re-run use command:**
   ```bash
   jfcm use <version>
   source ~/.zshrc  # or ~/.bashrc
   ```

5. **Manual PATH fix:**
   ```bash
   # Add this to your shell profile
   export PATH="$HOME/.jfcm/shim:$PATH"
   ```

6. **Check for shell function conflicts:**
   ```bash
   type jf
   # Should show the jfcm shell function, not a system binary
```

---

## 🧪 Advanced Examples

### Comparing Configuration Changes
```bash
# Compare configuration differences between versions
jfcm compare cli 2.74.0 2.73.0 -- config show --format json

# Check if a specific feature works across versions
jfcm compare cli old new -- rt search "libs-release-local/*.jar" --limit 5

# Compare release notes and changelogs
jfcm compare changelog v2.75.1 v2.76.0
```

### Performance Analysis
```bash
# Benchmark search performance across versions
jfcm benchmark 2.74.0,2.73.0,2.72.0 -- rt search "*" --limit 100 --iterations 3

# Test upload performance
jfcm benchmark prod,dev -- rt upload test.txt my-repo/ --iterations 5 --detailed
```

### Usage Analytics
```bash
# See your most used JFrog CLI commands
jfcm history --stats

# Track version adoption over time
jfcm history --version 2.74.0
```

### Automation and CI/CD
```bash
# Export benchmark results for CI analysis
jfcm benchmark $OLD_VERSION,$NEW_VERSION -- rt ping --format json > performance.json

# Compare outputs in automated testing
jfcm compare cli baseline canary --unified --no-color -- rt search "*.jar"

# Always use the latest version in CI/CD pipelines
jfcm use latest
jf --version
```

---

## 🧼 Uninstall
```bash
rm -rf ~/.jfcm
 # if installed via Homebrew
brew uninstall jfcm
```

---

## 🔧 Advanced Configuration

### History Management
- History is automatically tracked in `~/.jfcm/history.json`
- Limited to 1000 entries to prevent unlimited growth
- Includes command execution timing and metadata

### Health Check Features
- **System Environment**: OS compatibility, architecture support, shell detection
- **Installation Status**: jfcm directories, shim setup, PATH configuration
- **Priority Verification**: Ensures jfcm-managed `jf` has highest priority
- **Binary Execution**: Tests both `jfcm` and `jf` command execution
- **Network Connectivity**: GitHub API and JFrog releases connectivity
- **Performance Benchmarks**: Command execution timing and performance analysis
- **Security Checks**: File permissions and suspicious file detection
- **Auto-Fix Capability**: Automatically fixes common configuration issues
- **JSON Output**: Machine-readable output for CI/CD integration

### Performance Optimization
- Commands run in parallel when possible
- Configurable timeouts for long-running operations
- Efficient diff algorithms for large outputs

---

## 📝 Use Cases

### Development Teams
- **Version Testing**: Compare behavior across JFrog CLI versions before upgrading
- **Performance Monitoring**: Track performance regressions between releases
- **Usage Analytics**: Understand which commands and versions are used most
- **Latest Features**: Easily switch to the latest version with `jfcm use latest` to test new features

### DevOps Engineers
- **CI/CD Integration**: Automate version comparison in deployment pipelines
- **Performance Benchmarks**: Ensure new versions meet performance requirements
- **Migration Planning**: Analyze compatibility before major version upgrades
- **Automated Updates**: Use `jfcm use latest` in deployment scripts to always use the most recent stable version

### Enterprise Environments
- **Compliance Tracking**: Monitor which versions are being used across teams
- **Performance Optimization**: Identify and optimize slow operations
- **Training Insights**: Understand which commands teams use most frequently

---

## 📬 Feedback / Contributions
PRs and issues welcome! Open source, MIT licensed.

**GitHub:** https://github.com/jfrog/jfrog-cli-vm
