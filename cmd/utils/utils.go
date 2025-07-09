package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ToolName    = "jfvm"
	ConfigFile  = "config"
	VersionsDir = "versions"
	BinaryName  = "jf"
	ProjectFile = ".jfrog-version"
	AliasesDir  = "aliases"
	ShimDir     = "shim"
)

var (
	HomeDir      = os.Getenv("HOME")
	JfvmRoot     = filepath.Join(HomeDir, "."+ToolName)
	JfvmConfig   = filepath.Join(JfvmRoot, ConfigFile)
	JfvmVersions = filepath.Join(JfvmRoot, VersionsDir)
	JfvmAliases  = filepath.Join(JfvmRoot, AliasesDir)
	JfvmShim     = filepath.Join(JfvmRoot, ShimDir)
)

// InitializeJfvmDirectories creates the necessary jfvm directories if they don't exist
func InitializeJfvmDirectories() error {
	directories := []string{
		JfvmRoot,
		JfvmVersions,
		JfvmAliases,
		JfvmShim,
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func GetVersionFromProjectFile() (string, error) {
	fmt.Println("Attempting to read .jfrog-version file...")
	data, err := os.ReadFile(ProjectFile)
	if err != nil {
		fmt.Printf("Failed to read .jfrog-version file: %v\n", err)
		return "", err
	}
	version := strings.TrimSpace(string(data))
	fmt.Printf(".jfrog-version content: %s\n", version)
	return version, nil
}

func ResolveAlias(name string) (string, error) {
	path := filepath.Join(JfvmAliases, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ResolveVersionOrAlias attempts to resolve an alias first, then falls back to the original name
func ResolveVersionOrAlias(name string) (string, error) {
	// Try to resolve as alias first
	resolved, err := ResolveAlias(name)
	if err == nil {
		return strings.TrimSpace(resolved), nil
	}

	// If not an alias, return the original name
	return name, nil
}

// CheckVersionExists verifies that a version directory and binary exist
func CheckVersionExists(version string) error {
	versionDir := filepath.Join(JfvmVersions, version)
	binaryPath := filepath.Join(versionDir, BinaryName)

	// Check if version directory exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version directory does not exist")
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found in version directory")
	}

	return nil
}

// GetLatestVersion fetches the latest version from GitHub API
func GetLatestVersion() (string, error) {
	// Use GitHub API to get the latest release
	url := "https://api.github.com/repos/jfrog/jfrog-cli/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest version: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	content := string(body)
	tagNameIndex := strings.Index(content, `"tag_name":"`)
	if tagNameIndex == -1 {
		return "", fmt.Errorf("could not find tag_name in response")
	}

	// Extract the version starting after "tag_name":"
	startIndex := tagNameIndex + len(`"tag_name":"`)
	endIndex := strings.Index(content[startIndex:], `"`)
	if endIndex == -1 {
		return "", fmt.Errorf("could not parse tag_name value")
	}

	version := content[startIndex : startIndex+endIndex]
	if !strings.HasPrefix(version, "v2.") {
		return "", fmt.Errorf("invalid version format: %s", version)
	}
	version = strings.TrimPrefix(version, "v")

	return version, nil
}

// SetupShim creates the jf shim that will redirect to the active version
func SetupShim() error {
	// Create shim directory if it doesn't exist
	if err := os.MkdirAll(JfvmShim, 0755); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	shimPath := filepath.Join(JfvmShim, BinaryName)

	// Create shim script content based on platform
	var shimContent string
	if runtime.GOOS == "windows" {
		shimContent = createWindowsShim()
	} else {
		shimContent = createUnixShim()
	}

	// Write shim script
	if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
		return fmt.Errorf("failed to write shim script: %w", err)
	}

	return nil
}

// createUnixShim creates the shim script for Unix-like systems
func createUnixShim() string {
	return `#!/bin/bash
# jfvm shim - redirects jf commands to the active version

# Debug output if JFVM_DEBUG is set
if [ "$JFVM_DEBUG" = "1" ]; then
    echo "[shim] Executing jfvm shim" >&2
fi

# Get the active version from jfvm config
JFVM_ROOT="$HOME/.jfvm"
CONFIG_FILE="$JFVM_ROOT/config"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: No active jfvm version. Run 'jfvm use <version>' first." >&2
    exit 1
fi

ACTIVE_VERSION=$(cat "$CONFIG_FILE")
BINARY_PATH="$JFVM_ROOT/versions/$ACTIVE_VERSION/jf"

if [ "$JFVM_DEBUG" = "1" ]; then
    echo "[shim] Executing version: $ACTIVE_VERSION" >&2
    echo "[shim] Full binary path: $BINARY_PATH" >&2
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Active version $ACTIVE_VERSION not found. Run 'jfvm use <version>' to fix." >&2
    exit 1
fi

# Execute the binary with all arguments
exec "$BINARY_PATH" "$@"
`
}

// createWindowsShim creates the shim script for Windows
func createWindowsShim() string {
	return `@echo off
REM jfvm shim - redirects jf commands to the active version

REM Get the active version from jfvm config
set JFVM_ROOT=%USERPROFILE%\.jfvm
set CONFIG_FILE=%JFVM_ROOT%\config

if not exist "%CONFIG_FILE%" (
    echo Error: No active jfvm version. Run 'jfvm use ^<version^>' first.
    exit /b 1
)

for /f "delims=" %%i in (%CONFIG_FILE%) do set ACTIVE_VERSION=%%i
set BINARY_PATH=%JFVM_ROOT%\versions\%ACTIVE_VERSION%\jf.exe

if not exist "%BINARY_PATH%" (
    echo Error: Active version %ACTIVE_VERSION% not found. Run 'jfvm use ^<version^>' to fix.
    exit /b 1
)

REM Execute the binary with all arguments
"%BINARY_PATH%" %*
`
}

// UpdatePATH updates the user's shell profile to include jfvm shim in PATH with highest priority
func UpdatePATH() error {
	// First, clean up the old bin directory if it exists
	oldBinDir := filepath.Join(JfvmRoot, "bin")
	if _, err := os.Stat(oldBinDir); err == nil {
		fmt.Printf("Removing old bin directory: %s\n", oldBinDir)
		if err := os.RemoveAll(oldBinDir); err != nil {
			fmt.Printf("Warning: Failed to remove old bin directory: %v\n", err)
		}
	}

	// Get the current shell and clean up relevant profile files
	shell := GetCurrentShell()
	var profileFiles []string

	switch shell {
	case "zsh":
		profileFiles = []string{
			filepath.Join(HomeDir, ".zshrc"),
			filepath.Join(HomeDir, ".profile"), // zsh also loads .profile
		}
	case "bash":
		profileFiles = []string{
			filepath.Join(HomeDir, ".bashrc"),
			filepath.Join(HomeDir, ".bash_profile"),
			filepath.Join(HomeDir, ".profile"),
		}
	default:
		// For other shells, just clean the primary profile
		primaryProfile := GetShellProfile(shell)
		if primaryProfile != "" {
			profileFiles = []string{primaryProfile}
		}
	}

	for _, profileFile := range profileFiles {
		if err := cleanupProfileFile(profileFile); err != nil {
			fmt.Printf("Warning: Failed to cleanup %s: %v\n", profileFile, err)
		}
	}

	// Get the primary shell profile for adding the new configuration
	primaryProfileFile := GetShellProfile(shell)

	if primaryProfileFile == "" {
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Read current primary profile
	content, err := os.ReadFile(primaryProfileFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read profile file: %w", err)
	}

	profileContent := string(content)

	// Check if jfvm shim PATH is already added with correct priority
	if strings.Contains(profileContent, JfvmShim) {
		// Verify it's at the beginning of PATH (highest priority)
		if strings.Contains(profileContent, fmt.Sprintf("export PATH=\"%s:$PATH\"", JfvmShim)) {
			fmt.Printf("jfvm PATH already configured with highest priority in %s\n", primaryProfileFile)
			return nil
		} else {
			// Remove old incorrect PATH entry
			fmt.Printf("Updating jfvm PATH configuration to ensure highest priority...\n")
		}
	}

	// Add jfvm shim PATH to profile with highest priority (prepend to PATH)
	// Also add shell function for better priority handling (similar to nvm)
	pathLine := fmt.Sprintf(`

# jfvm PATH configuration - ensures jfvm-managed jf takes highest priority
export PATH="%s:$PATH"

# jfvm shell function for enhanced priority (similar to nvm approach)
jf() {
    # Check if jfvm shim exists and is executable
    if [ -x "%s/jf" ]; then
        # Execute jfvm-managed jf with highest priority
        "%s/jf" "$@"
    else
        # Fallback to system jf if jfvm shim not available
        command jf "$@"
    fi
}

`, JfvmShim, JfvmShim, JfvmShim)

	// Append to profile
	if err := os.WriteFile(primaryProfileFile, []byte(profileContent+pathLine), 0644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}

	fmt.Printf("✅ Added jfvm shim to PATH with highest priority in %s\n", primaryProfileFile)
	fmt.Printf("🔧 jfvm-managed jf will now take precedence over system installations\n")
	fmt.Printf("📝 Please restart your terminal or run: source %s\n", primaryProfileFile)

	return nil
}

// cleanupProfileFile removes old jfvm PATH entries from a profile file
func cleanupProfileFile(profileFile string) error {
	// Check if file exists
	if _, err := os.Stat(profileFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to clean
	}

	// Read current profile
	content, err := os.ReadFile(profileFile)
	if err != nil {
		return fmt.Errorf("failed to read profile file: %w", err)
	}

	profileContent := string(content)
	originalContent := profileContent

	// Remove any old jfvm PATH entries (both bin and shim)
	lines := strings.Split(profileContent, "\n")
	var newLines []string

	for _, line := range lines {
		// Skip lines that contain old jfvm PATH entries
		if strings.Contains(line, "$HOME/.jfvm/bin") ||
			strings.Contains(line, "~/.jfvm/bin") ||
			strings.Contains(line, JfvmShim) ||
			strings.Contains(line, "jfvm PATH") ||
			strings.Contains(line, "jf() {") {
			continue
		}
		newLines = append(newLines, line)
	}

	profileContent = strings.Join(newLines, "\n")

	// Only write if content changed
	if profileContent != originalContent {
		if err := os.WriteFile(profileFile, []byte(profileContent), 0644); err != nil {
			return fmt.Errorf("failed to write profile file: %w", err)
		}
		fmt.Printf("🧹 Cleaned up old jfvm entries from %s\n", profileFile)
	}

	return nil
}

// GetCurrentShell determines the current shell
func GetCurrentShell() string {
	// Try to get shell from environment
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}

	// Fallback based on OS
	if runtime.GOOS == "windows" {
		return "cmd"
	}

	// Default to bash for Unix-like systems
	return "bash"
}

// GetShellProfile returns the profile file path for the given shell
func GetShellProfile(shell string) string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE") // Windows fallback
	}

	switch shell {
	case "bash":
		// Check for .bash_profile first, then .bashrc
		if _, err := os.Stat(filepath.Join(homeDir, ".bash_profile")); err == nil {
			return filepath.Join(homeDir, ".bash_profile")
		}
		return filepath.Join(homeDir, ".bashrc")
	case "zsh":
		return filepath.Join(homeDir, ".zshrc")
	case "fish":
		return filepath.Join(homeDir, ".config/fish/config.fish")
	case "cmd":
		// Windows doesn't use profile files in the same way
		return ""
	default:
		return ""
	}
}

// GetActiveVersion returns the currently active version
func GetActiveVersion() (string, error) {
	if _, err := os.Stat(JfvmConfig); os.IsNotExist(err) {
		return "", fmt.Errorf("no active version set")
	}

	content, err := os.ReadFile(JfvmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// GetActiveBinaryPath returns the path to the active jf binary
func GetActiveBinaryPath() (string, error) {
	version, err := GetActiveVersion()
	if err != nil {
		return "", err
	}

	binaryPath := filepath.Join(JfvmVersions, version, BinaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("active version %s not found", version)
	}

	return binaryPath, nil
}

// CheckShimSetup checks if the shim is properly set up
func CheckShimSetup() error {
	shimPath := filepath.Join(JfvmShim, BinaryName)
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		return fmt.Errorf("shim not found at %s", shimPath)
	}

	// Check if shim is executable
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(shimPath); err == nil {
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("shim is not executable")
			}
		}
	}

	return nil
}

// VerifyPriority checks if jfvm-managed jf has highest priority
func VerifyPriority() error {
	// Check if shim exists
	if err := CheckShimSetup(); err != nil {
		return fmt.Errorf("shim setup issue: %w", err)
	}

	// Check PATH order
	path := os.Getenv("PATH")
	pathDirs := strings.Split(path, string(os.PathListSeparator))

	// Find jfvm shim in PATH
	shimIndex := -1
	systemJfIndex := -1

	for i, dir := range pathDirs {
		if strings.Contains(dir, ".jfvm/shim") {
			shimIndex = i
		}
		// Check for common system jf locations
		if strings.Contains(dir, "/usr/local/bin") || strings.Contains(dir, "/opt/homebrew/bin") || strings.Contains(dir, "/usr/bin") {
			if systemJfIndex == -1 {
				systemJfIndex = i
			}
		}
	}

	if shimIndex == -1 {
		return fmt.Errorf("jfvm shim not found in PATH")
	}

	if systemJfIndex != -1 && shimIndex > systemJfIndex {
		return fmt.Errorf("jfvm shim is not first in PATH (index %d vs system index %d)", shimIndex, systemJfIndex)
	}

	return nil
}
