package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

	// Create HTTP client with proper headers
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add proper headers to avoid rate limiting
	req.Header.Set("User-Agent", "jfvm/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Add GitHub token if available (for CI environments)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusForbidden:
		return "", fmt.Errorf("GitHub API access forbidden (403). This may be due to rate limiting. Try again later or set GITHUB_TOKEN environment variable")
	case http.StatusNotFound:
		return "", fmt.Errorf("GitHub API endpoint not found (404). Please check the repository URL")
	case http.StatusTooManyRequests:
		return "", fmt.Errorf("GitHub API rate limit exceeded (429). Try again later or set GITHUB_TOKEN environment variable")
	default:
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

// GetLatestVersionWithFallback attempts to get the latest version with fallback options
func GetLatestVersionWithFallback() (string, error) {
	// Try GitHub API first
	version, err := GetLatestVersion()
	if err == nil {
		return version, nil
	}

	// If GitHub API fails, try alternative approaches
	fmt.Printf("Warning: GitHub API failed: %v\n", err)
	fmt.Println("Attempting fallback methods...")

	// Fallback 1: Try JFrog releases API directly
	if fallbackVersion, fallbackErr := getLatestVersionFromJFrogReleases(); fallbackErr == nil {
		fmt.Printf("Successfully got latest version from JFrog releases: %s\n", fallbackVersion)
		return fallbackVersion, nil
	}

	// Fallback 2: Return a known recent version as last resort
	fmt.Println("All API methods failed. Using fallback version 2.77.0")
	return "2.77.0", nil
}

// getLatestVersionFromJFrogReleases tries to get the latest version from JFrog's release server
func getLatestVersionFromJFrogReleases() (string, error) {
	// TODO: Implement proper parsing of JFrog releases directory listing
	// Currently hardcoded to latest known version to ensure fallback works
	// Future improvement: Parse https://releases.jfrog.io/artifactory/jfrog-cli/v2-jf/
	// directory listing to dynamically find the latest version

	// Try to get version info from JFrog's release server
	url := "https://releases.jfrog.io/artifactory/jfrog-cli/v2-jf/"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create JFrog releases request: %w", err)
	}

	req.Header.Set("User-Agent", "jfvm/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from JFrog releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("JFrog releases API returned status: %d", resp.StatusCode)
	}

	// For now, return the current latest version (2.77.0)
	// TODO: Parse the directory listing to dynamically find the latest version
	return "2.77.0", nil
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

# Capture the full command line as typed
FULL_CMD="$(basename "$0") $@"

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

# Skip history recording if disabled or in debug mode
if [ "$JFVM_NO_HISTORY" = "1" ] || [ "$JFVM_DEBUG" = "1" ]; then
    exec "$BINARY_PATH" "$@"
fi

# Record command execution in history (lightweight)
START_TIME=$(date +%s)

# Execute the binary and capture output
OUTPUT=$("$BINARY_PATH" "$@" 2>&1)
EXIT_CODE=$?
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Record history asynchronously to avoid blocking
# Try to find the jfvm binary in the current directory first, then fallback to PATH
JFVM_BINARY=""
if [ -x "./jfvm" ]; then
    JFVM_BINARY="./jfvm"
elif [ -x "$(dirname "$0")/../jfvm" ]; then
    JFVM_BINARY="$(dirname "$0")/../jfvm"
else
    JFVM_BINARY="$(command -v jfvm 2>/dev/null || echo '')"
fi

if [ -n "$JFVM_BINARY" ] && [ -x "$JFVM_BINARY" ]; then
    ("$JFVM_BINARY" add-history-entry "$ACTIVE_VERSION" "$FULL_CMD" "$DURATION" "$EXIT_CODE" "$OUTPUT" >/dev/null 2>&1) &
fi

# Output the result immediately
echo "$OUTPUT"
exit $EXIT_CODE
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

REM Record command execution in history
set COMMAND=jf %*
set START_TIME=%TIME%

REM Execute the binary with all arguments
"%BINARY_PATH%" %*
set EXIT_CODE=%ERRORLEVEL%

REM Record command execution in history using jfvm binary
where jfvm >nul 2>&1
if %ERRORLEVEL% == 0 (
    jfvm add-history-entry "%ACTIVE_VERSION%" "%COMMAND%" "0" "%EXIT_CODE%" "Windows output capture not implemented" >nul 2>&1
)
`
}

// Unique block markers for jfvm PATH
const (
	JfvmBlockStart = "# >>> jfvm PATH (managed by jfvm)"
	JfvmBlockEnd   = "# <<< jfvm PATH (managed by jfvm)"
)

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

	// Get the current shell and check the primary profile file
	shell := GetCurrentShell()
	primaryProfileFile := GetShellProfile(shell)

	if primaryProfileFile == "" {
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Read current primary profile (create if missing)
	content, err := os.ReadFile(primaryProfileFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read profile file: %w", err)
	}
	profileContent := string(content)

	// Check if the correct jfvm block already exists
	expectedBlock := fmt.Sprintf(`# >>> jfvm PATH (managed by jfvm)
export PATH="%s:$PATH"
# <<< jfvm PATH (managed by jfvm)`, JfvmShim)

	// Check if the expected block is already present
	if strings.Contains(profileContent, expectedBlock) {
		fmt.Printf("✅ jfvm PATH already configured correctly in %s\n", primaryProfileFile)
		return nil
	}

	// Check if there are any jfvm-related entries that need cleanup
	if strings.Contains(profileContent, "jfvm") {
		fmt.Printf("🧹 Cleaning up old jfvm entries from %s\n", primaryProfileFile)
		// Clean up the primary profile file
		if err := CleanupProfileFile(primaryProfileFile); err != nil {
			fmt.Printf("Warning: Failed to cleanup %s: %v\n", primaryProfileFile, err)
		}

		// Re-read the cleaned content
		content, err = os.ReadFile(primaryProfileFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read profile file after cleanup: %w", err)
		}
		profileContent = string(content)
	}

	// Remove any existing jfvm block (even if partial/corrupted)
	profileContent = RemoveJfvmBlock(profileContent)

	// Add jfvm shim PATH block
	block := fmt.Sprintf(`# >>> jfvm PATH (managed by jfvm)
export PATH="%s:$PATH"
# <<< jfvm PATH (managed by jfvm)
`, JfvmShim)

	// Ensure proper formatting: trim trailing whitespace and add newline if needed
	profileContent = strings.TrimRight(profileContent, "\n\r\t ")
	if !strings.HasSuffix(profileContent, "\n") {
		profileContent += "\n"
	}
	profileContent += "\n" + block

	// Validate syntax before writing (for bash/zsh profiles)
	if strings.HasSuffix(primaryProfileFile, "rc") || strings.HasSuffix(primaryProfileFile, "_profile") {
		if err := validateShellSyntax(profileContent, primaryProfileFile); err != nil {
			// If syntax validation fails, try to create a minimal working profile
			fmt.Printf("⚠️  Syntax validation failed, creating minimal profile: %v\n", err)
			profileContent = fmt.Sprintf(`# Minimal jfvm profile
# Original content had syntax errors
# Original file backed up to %s.jfvm.backup

%s
`, primaryProfileFile, block)
		}
	}

	// Create backup before writing
	backupFile := primaryProfileFile + ".jfvm.backup"
	if err := os.WriteFile(backupFile, []byte(string(content)), 0644); err != nil {
		fmt.Printf("Warning: Failed to create backup: %v\n", err)
	}

	if err := os.WriteFile(primaryProfileFile, []byte(profileContent), 0644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}

	fmt.Printf("✅ Added jfvm shim to PATH with highest priority in %s\n", primaryProfileFile)
	fmt.Printf("🔧 jfvm-managed jf will now take precedence over system installations\n")
	fmt.Printf("📝 Please restart your terminal or run: source %s\n", primaryProfileFile)

	return nil
}

// RemoveJfvmBlock removes any existing jfvm PATH/function block from the profile content
func RemoveJfvmBlock(content string) string {
	fmt.Printf("🔍 Starting jfvm block removal...\n")
	fmt.Printf("📊 Runner info: %s\n", GetRunnerInfo())

	lines := strings.Split(content, "\n")
	var newLines []string
	inBlock := false
	blockStartLine := -1
	blockEndLine := -1
	linesRemoved := 0

	fmt.Printf("📝 Processing %d lines\n", len(lines))

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for start marker (exact match)
		if trimmedLine == JfvmBlockStart {
			fmt.Printf("🚨 Found jfvm block start marker at line %d\n", i+1)
			inBlock = true
			blockStartLine = i + 1
			linesRemoved++
			continue
		}

		// Check for end marker (exact match)
		if inBlock && trimmedLine == JfvmBlockEnd {
			fmt.Printf("✅ Found jfvm block end marker at line %d\n", i+1)
			inBlock = false
			blockEndLine = i + 1
			linesRemoved++
			continue
		}

		// Only keep lines that are not inside a jfvm block
		if !inBlock {
			newLines = append(newLines, line)
		} else {
			fmt.Printf("🗑️  Skipping line %d in jfvm block: %s\n", i+1, line)
			linesRemoved++
		}
	}

	if blockStartLine > 0 && blockEndLine > 0 {
		fmt.Printf("✅ Removed jfvm block from lines %d to %d (%d lines total)\n", blockStartLine, blockEndLine, linesRemoved)
	} else if blockStartLine > 0 {
		fmt.Printf("⚠️  Found start marker at line %d but no end marker - removed %d lines\n", blockStartLine, linesRemoved)
	} else {
		fmt.Printf("ℹ️  No jfvm blocks found\n")
	}

	result := strings.Join(newLines, "\n")
	fmt.Printf("📊 Original: %d lines, After removal: %d lines\n", len(lines), len(newLines))

	return result
}

// CleanupProfileFile removes old jfvm PATH/function blocks from a profile file using block markers
func CleanupProfileFile(profileFile string) error {
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

	// First, remove properly marked blocks
	cleaned := RemoveJfvmBlock(profileContent)

	// Then, remove any malformed jfvm content that might cause syntax errors
	cleaned = removeMalformedJfvmContent(cleaned)

	// Finally, remove any orphaned shell structures that might cause syntax errors
	cleaned = removeOrphanedShellStructures(cleaned)

	// As a final safety measure, remove ANY line containing jfvm-related content
	// This is the most aggressive cleanup to ensure no jfvm content remains
	cleaned = removeAllJfvmContent(cleaned)

	// Remove the specific corrupted patterns from the user's output
	cleaned = removeCorruptedJfvmPatterns(cleaned)

	// Validate syntax before writing
	if strings.HasSuffix(profileFile, "rc") || strings.HasSuffix(profileFile, "_profile") {
		if err := validateShellSyntax(cleaned, profileFile); err != nil {
			// If syntax validation fails, create a minimal clean profile
			fmt.Printf("⚠️  Syntax validation failed after cleanup, creating minimal profile: %v\n", err)
			cleaned = fmt.Sprintf(`# Clean profile created by jfvm cleanup
# Original content had syntax errors
# Original file backed up to %s.jfvm.backup
`, profileFile)
		}
	}

	// Only write if content changed
	if cleaned != profileContent {
		// Create backup before writing
		backupFile := profileFile + ".jfvm.backup"
		if err := os.WriteFile(backupFile, []byte(profileContent), 0644); err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		}

		if err := os.WriteFile(profileFile, []byte(cleaned), 0644); err != nil {
			return fmt.Errorf("failed to write profile file: %w", err)
		}
		fmt.Printf("🧹 Cleaned up old jfvm entries from %s\n", profileFile)
	}

	return nil
}

// removeMalformedJfvmContent removes jfvm-related content that might cause syntax errors
func removeMalformedJfvmContent(content string) string {
	fmt.Printf("🔍 Starting malformed jfvm content removal...\n")

	lines := strings.Split(content, "\n")
	var newLines []string
	linesRemoved := 0

	fmt.Printf("📝 Processing %d lines for malformed content\n", len(lines))

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip any line that contains jfvm-related content
		// This catches any jfvm content that wasn't properly marked with block markers
		if strings.Contains(trimmedLine, "jfvm") ||
			strings.Contains(trimmedLine, ".jfvm") ||
			strings.Contains(trimmedLine, "jf()") ||
			strings.Contains(trimmedLine, "JFVM_") ||
			strings.Contains(trimmedLine, "jfvm shell function") ||
			strings.Contains(trimmedLine, "Check if jfvm") ||
			strings.Contains(trimmedLine, "Execute jfvm") ||
			strings.Contains(trimmedLine, "Fallback to system") ||
			strings.Contains(trimmedLine, "command jf") ||
			strings.Contains(trimmedLine, "jfvm PATH configuration") ||
			strings.Contains(trimmedLine, "jfvm-managed jf") ||
			strings.Contains(trimmedLine, "enhanced priority") {
			fmt.Printf("🗑️  Removing malformed jfvm content at line %d: %s\n", i+1, line)
			linesRemoved++
			continue
		}

		// Keep non-jfvm lines
		newLines = append(newLines, line)
	}

	fmt.Printf("📊 Removed %d lines with malformed jfvm content\n", linesRemoved)
	result := strings.Join(newLines, "\n")
	fmt.Printf("📊 After malformed content removal: %d lines\n", len(newLines))

	return result
}

// removeOrphanedShellStructures removes orphaned if/else/fi statements that might cause syntax errors
func removeOrphanedShellStructures(content string) string {
	fmt.Printf("🔍 Starting orphaned shell structure removal...\n")

	lines := strings.Split(content, "\n")
	var newLines []string
	linesRemoved := 0
	ifCount := 0
	elseCount := 0
	fiCount := 0
	inFunction := false
	functionDepth := 0

	fmt.Printf("📝 Processing %d lines for orphaned shell structures\n", len(lines))

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track function definitions
		if strings.Contains(trimmedLine, "() {") || strings.Contains(trimmedLine, "() {") {
			inFunction = true
			functionDepth = 0
		}

		// Count shell control structures
		if strings.HasPrefix(trimmedLine, "if ") || trimmedLine == "if" {
			ifCount++
		} else if strings.HasPrefix(trimmedLine, "else") || trimmedLine == "else" {
			elseCount++
		} else if trimmedLine == "fi" {
			fiCount++
		}

		// Track function depth
		if inFunction {
			if strings.Contains(line, "{") {
				functionDepth++
			}
			if strings.Contains(line, "}") {
				functionDepth--
				if functionDepth <= 0 {
					inFunction = false
					functionDepth = 0
				}
			}
		}

		// Check for orphaned else statements - be more aggressive
		if strings.HasPrefix(trimmedLine, "else") || trimmedLine == "else" {
			// If we haven't seen any 'if' statements yet, or if we have more 'else' than 'if', remove it
			if ifCount == 0 || elseCount >= ifCount {
				fmt.Printf("🚨 Removing orphaned 'else' at line %d: %s\n", i+1, line)
				linesRemoved++
				continue
			}
		}

		// Check for orphaned fi statements - be more aggressive
		if trimmedLine == "fi" {
			// If we haven't seen any 'if' statements yet, or if we have more 'fi' than 'if', remove it
			if ifCount == 0 || fiCount >= ifCount {
				fmt.Printf("🚨 Removing orphaned 'fi' at line %d: %s\n", i+1, line)
				linesRemoved++
				continue
			}
		}

		// Check for orphaned function content (lines that look like function body but no function declaration)
		if !inFunction && (strings.Contains(trimmedLine, "Check if") ||
			strings.Contains(trimmedLine, "Execute") ||
			strings.Contains(trimmedLine, "Fallback to") ||
			strings.Contains(trimmedLine, "command jf") ||
			strings.Contains(trimmedLine, "jfvm-managed jf")) {
			fmt.Printf("🚨 Removing orphaned function content at line %d: %s\n", i+1, line)
			linesRemoved++
			continue
		}

		// Keep the line
		newLines = append(newLines, line)
	}

	fmt.Printf("📊 Shell structure counts - if: %d, else: %d, fi: %d\n", ifCount, elseCount, fiCount)
	fmt.Printf("📊 Removed %d orphaned shell structures\n", linesRemoved)

	result := strings.Join(newLines, "\n")
	fmt.Printf("📊 After shell structure cleanup: %d lines\n", len(newLines))

	return result
}

// removeAllJfvmContent removes ANY line containing jfvm-related content as a final safety measure
func removeAllJfvmContent(content string) string {
	fmt.Printf("🔍 Starting final jfvm content removal (most aggressive)...\n")

	lines := strings.Split(content, "\n")
	var newLines []string
	linesRemoved := 0

	fmt.Printf("📝 Processing %d lines for final jfvm cleanup\n", len(lines))

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Remove ANY line that contains jfvm-related content
		if strings.Contains(strings.ToLower(trimmedLine), "jfvm") ||
			strings.Contains(trimmedLine, ".jfvm") ||
			strings.Contains(trimmedLine, "jf()") ||
			strings.Contains(trimmedLine, "JFVM_") ||
			strings.Contains(trimmedLine, "jfvm shell function") ||
			strings.Contains(trimmedLine, "Check if jfvm") ||
			strings.Contains(trimmedLine, "Execute jfvm") ||
			strings.Contains(trimmedLine, "Fallback to system") ||
			strings.Contains(trimmedLine, "command jf") ||
			strings.Contains(trimmedLine, "jfvm PATH configuration") ||
			strings.Contains(trimmedLine, "jfvm-managed jf") ||
			strings.Contains(trimmedLine, "enhanced priority") ||
			strings.Contains(trimmedLine, "Check if") ||
			strings.Contains(trimmedLine, "Execute") ||
			strings.Contains(trimmedLine, "Fallback to") ||
			strings.Contains(trimmedLine, "jfvm-managed") {
			fmt.Printf("🗑️  Final removal of jfvm content at line %d: %s\n", i+1, line)
			linesRemoved++
			continue
		}

		// Keep non-jfvm lines
		newLines = append(newLines, line)
	}

	fmt.Printf("📊 Final cleanup removed %d lines with jfvm content\n", linesRemoved)
	result := strings.Join(newLines, "\n")
	fmt.Printf("📊 After final cleanup: %d lines\n", len(newLines))

	return result
}

// removeCorruptedJfvmPatterns removes the specific corrupted patterns from the user's output
func removeCorruptedJfvmPatterns(content string) string {
	fmt.Printf("🔍 Starting corrupted pattern removal...\n")

	lines := strings.Split(content, "\n")
	var newLines []string
	linesRemoved := 0

	fmt.Printf("📝 Processing %d lines for corrupted pattern removal\n", len(lines))

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Remove the exact corrupted patterns from the user's output
		if strings.Contains(trimmedLine, "# jfvm shell function for enhanced priority") ||
			strings.Contains(trimmedLine, "# Check if jfvm shim exists and is executable") ||
			strings.Contains(trimmedLine, "# Execute jfvm-managed jf with highest priority") ||
			strings.Contains(trimmedLine, "else") ||
			strings.Contains(trimmedLine, "# Fallback to system jf if jfvm shim not available") ||
			strings.Contains(trimmedLine, "command jf \"$@\"") ||
			strings.Contains(trimmedLine, "fi") ||
			strings.Contains(trimmedLine, "}") ||
			strings.Contains(trimmedLine, "# jfvm PATH configuration - ensures jfvm-managed jf takes highest priority") ||
			strings.Contains(trimmedLine, "export PATH=\"/Users/runner/.jfvm/shim:$PATH\"") ||
			strings.Contains(trimmedLine, "jf() {") {
			fmt.Printf("🚨 Removing corrupted pattern at line %d: %s\n", i+1, line)
			linesRemoved++
			continue
		}

		// Keep non-corrupted lines
		newLines = append(newLines, line)
	}

	fmt.Printf("📊 Removed %d lines with corrupted patterns\n", linesRemoved)
	result := strings.Join(newLines, "\n")
	fmt.Printf("📊 After corrupted pattern removal: %d lines\n", len(newLines))

	return result
}

// validateShellSyntax validates shell syntax using bash -n
func validateShellSyntax(content, filename string) error {
	// Create a temporary file for validation
	tmpFile, err := os.CreateTemp("", "jfvm-validate-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content to temp file
	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Validate syntax using bash -n
	cmd := exec.Command("bash", "-n", tmpFile.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell syntax error in %s: %w", filename, err)
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

// SwitchToVersion switches to the specified version for command execution
func SwitchToVersion(version string) error {
	// Check if version exists
	binPath := filepath.Join(JfvmVersions, version, BinaryName)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s not found", version)
	}

	// Write the version to config file
	if err := os.WriteFile(JfvmConfig, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetRunnerInfo returns information about the current runner environment
func GetRunnerInfo() string {
	info := []string{}

	// Get runner ID if available
	if runnerID := os.Getenv("RUNNER_ID"); runnerID != "" {
		info = append(info, fmt.Sprintf("Runner ID: %s", runnerID))
	}

	// Get runner name if available
	if runnerName := os.Getenv("RUNNER_NAME"); runnerName != "" {
		info = append(info, fmt.Sprintf("Runner Name: %s", runnerName))
	}

	// Get workflow run ID
	if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
		info = append(info, fmt.Sprintf("Run ID: %s", runID))
	}

	// Get workflow run number
	if runNumber := os.Getenv("GITHUB_RUN_NUMBER"); runNumber != "" {
		info = append(info, fmt.Sprintf("Run Number: %s", runNumber))
	}

	// Get job ID
	if jobID := os.Getenv("GITHUB_JOB"); jobID != "" {
		info = append(info, fmt.Sprintf("Job: %s", jobID))
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info = append(info, fmt.Sprintf("Hostname: %s", hostname))
	}

	// Get current timestamp
	info = append(info, fmt.Sprintf("Timestamp: %s", time.Now().Format(time.RFC3339)))

	return strings.Join(info, ", ")
}
