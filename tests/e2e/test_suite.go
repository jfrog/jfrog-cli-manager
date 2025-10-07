package e2e

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSuite holds the test environment
type TestSuite struct {
	jfcmPath    string
	TestDir     string
	OriginalPWD string
}

// findjfcmBinary searches upwards from the current directory for the jfcm binary
func findjfcmBinary() (string, error) {
	// Check jfcm_PATH env var first
	if envPath := os.Getenv("jfcm_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}
	// Start from current dir and walk up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "jfcm")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// SetupTestSuite initializes the test environment
func SetupTestSuite(t *testing.T) *TestSuite {
	// Find the jfcm binary robustly
	jfcmSrc, err := findjfcmBinary()
	if err != nil {
		t.Fatalf("jfcm binary not found in any parent directory or jfcm_PATH. Please build it before running tests.")
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", "jfcm-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Copy jfcm binary into testDir
	jfcmDst := filepath.Join(testDir, "jfcm")
	srcFile, err := os.Open(jfcmSrc)
	if err != nil {
		t.Fatalf("Failed to open jfcm binary: %v", err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(jfcmDst)
	if err != nil {
		t.Fatalf("Failed to create jfcm binary in test dir: %v", err)
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("Failed to copy jfcm binary: %v", err)
	}
	os.Chmod(jfcmDst, 0755) // Ensure it's executable

	// Store original working directory
	originalPWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to test directory
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	return &TestSuite{
		jfcmPath:    "./jfcm",
		TestDir:     testDir,
		OriginalPWD: originalPWD,
	}
}

// CleanupTestSuite cleans up the test environment
func (ts *TestSuite) CleanupTestSuite(t *testing.T) {
	// Change back to original directory
	if err := os.Chdir(ts.OriginalPWD); err != nil {
		t.Logf("Warning: Failed to change back to original directory: %v", err)
	}

	// Clean up test directory
	if err := os.RemoveAll(ts.TestDir); err != nil {
		t.Logf("Warning: Failed to remove test directory: %v", err)
	}
}

// RunCommand executes a jfcm command and returns the output
func (ts *TestSuite) RunCommand(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command(ts.jfcmPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommandWithTimeout executes a jfcm command with timeout
func (ts *TestSuite) RunCommandWithTimeout(t *testing.T, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ts.jfcmPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// AssertContains checks if output contains expected text
func (ts *TestSuite) AssertContains(t *testing.T, output, expected string) {
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
	}
}

// AssertNotContains checks if output doesn't contain unexpected text
func (ts *TestSuite) AssertNotContains(t *testing.T, output, unexpected string) {
	if strings.Contains(output, unexpected) {
		t.Errorf("Expected output to not contain '%s', but got: %s", unexpected, output)
	}
}

// AssertSuccess checks if command executed successfully
func (ts *TestSuite) AssertSuccess(t *testing.T, output string, err error) {
	if err != nil {
		t.Errorf("Expected command to succeed, but got error: %v\nOutput: %s", err, output)
	}
}

// AssertFailure checks if command failed as expected
func (ts *TestSuite) AssertFailure(t *testing.T, output string, err error) {
	if err == nil {
		t.Errorf("Expected command to fail, but it succeeded\nOutput: %s", output)
	}
}

// WaitForFile waits for a file to exist
func (ts *TestSuite) WaitForFile(t *testing.T, filepath string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath); err == nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// CreateTestFile creates a test file with content
func (ts *TestSuite) CreateTestFile(t *testing.T, filename, content string) {
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filename, err)
	}
}

// TestCoreVersionManagement tests basic version management features
func TestCoreVersionManagement(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Install Version", func(t *testing.T) {
		output, err := ts.RunCommand(t, "install", "2.74.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("List Installed Versions", func(t *testing.T) {
		output, err := ts.RunCommand(t, "list")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Use Specific Version", func(t *testing.T) {
		output, err := ts.RunCommand(t, "use", "2.74.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Use Latest Version", func(t *testing.T) {
		output, err := ts.RunCommandWithTimeout(t, 30*time.Second, "use", "latest")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Latest version:")
	})

	t.Run("Remove Version", func(t *testing.T) {
		output, err := ts.RunCommand(t, "remove", "2.74.0")
		ts.AssertSuccess(t, output, err)
	})

	t.Run("Clear All Versions", func(t *testing.T) {
		// First install a version to clear
		ts.RunCommand(t, "install", "2.74.0")

		output, err := ts.RunCommand(t, "clear")
		ts.AssertSuccess(t, output, err)

		// Verify it's cleared
		listOutput, err := ts.RunCommand(t, "list")
		ts.AssertSuccess(t, listOutput, err)
		ts.AssertNotContains(t, listOutput, "2.74.0")
	})
}

// TestAliasManagement tests alias functionality
func TestAliasManagement(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install a version first
	ts.RunCommand(t, "install", "2.74.0")

	t.Run("Set Alias", func(t *testing.T) {
		output, err := ts.RunCommand(t, "alias", "set", "prod", "2.74.0")
		ts.AssertSuccess(t, output, err)
	})

	t.Run("Get Alias", func(t *testing.T) {
		output, err := ts.RunCommand(t, "alias", "get", "prod")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Use Alias", func(t *testing.T) {
		output, err := ts.RunCommand(t, "use", "prod")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Block Latest Alias", func(t *testing.T) {
		output, err := ts.RunCommand(t, "alias", "set", "latest", "2.74.0")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "reserved keyword")
	})

	t.Run("Remove Alias", func(t *testing.T) {
		output, err := ts.RunCommand(t, "alias", "remove", "prod")
		ts.AssertSuccess(t, output, err)

		// Verify it's removed
		_, err = ts.RunCommand(t, "alias", "get", "prod")
		ts.AssertFailure(t, "", err)
	})
}

// TestProjectSpecificVersion tests .jfrog-version file functionality
func TestProjectSpecificVersion(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install a version first
	ts.RunCommand(t, "install", "2.74.0")

	t.Run("Use Project Version File", func(t *testing.T) {
		// Create .jfrog-version file
		ts.CreateTestFile(t, ".jfrog-version", "2.74.0")

		output, err := ts.RunCommand(t, "use")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Use Without Project File", func(t *testing.T) {
		// Remove .jfrog-version file
		os.Remove(".jfrog-version")

		output, err := ts.RunCommand(t, "use")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "No version provided")
	})
}

// TestLinkLocalBinary tests linking local binaries
func TestLinkLocalBinary(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Link Local Binary", func(t *testing.T) {
		// Create a dummy binary for testing
		dummyBinary := filepath.Join(ts.TestDir, "dummy-jf")
		ts.CreateTestFile(t, dummyBinary, "#!/bin/bash\necho 'dummy jf binary'")
		os.Chmod(dummyBinary, 0755)

		output, err := ts.RunCommand(t, "link", "--from", dummyBinary, "--name", "test-local")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Linked")
	})

	t.Run("Use Linked Binary", func(t *testing.T) {
		output, err := ts.RunCommand(t, "use", "test-local")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "test-local")
	})

	t.Run("Link Non-existent Binary", func(t *testing.T) {
		output, err := ts.RunCommand(t, "link", "--from", "/non/existent/path", "--name", "invalid")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "no such file")
	})
}

// TestCompareVersions tests version comparison functionality
func TestCompareVersions(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install multiple versions
	ts.RunCommand(t, "install", "2.74.0")
	ts.RunCommand(t, "install", "2.73.0")

	t.Run("Compare CLI Version Output", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "cli", "2.74.0", "2.73.0", "--", "--version")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
		ts.AssertContains(t, output, "2.73.0")
	})

	t.Run("Compare CLI With Unified Diff", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "cli", "2.74.0", "2.73.0", "--unified", "--", "--version")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "unified")
	})
}

// TestBenchmarkVersions tests benchmarking functionality
func TestBenchmarkVersions(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install multiple versions
	ts.RunCommand(t, "install", "2.74.0")
	ts.RunCommand(t, "install", "2.73.0")

	t.Run("Benchmark Versions", func(t *testing.T) {
		output, err := ts.RunCommand(t, "benchmark", "2.74.0,2.73.0", "--", "--version")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
		ts.AssertContains(t, output, "2.73.0")
	})

	t.Run("Benchmark With JSON Output", func(t *testing.T) {
		output, err := ts.RunCommand(t, "benchmark", "2.74.0,2.73.0", "--", "--version", "--format", "json")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "json")
	})
}

// TestHistoryTracking tests history functionality
func TestHistoryTracking(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install and use a version to generate history
	ts.RunCommand(t, "install", "2.74.0")
	ts.RunCommand(t, "use", "2.74.0")

	t.Run("Show History", func(t *testing.T) {
		output, err := ts.RunCommand(t, "history")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
	})

	t.Run("Show History Stats", func(t *testing.T) {
		output, err := ts.RunCommand(t, "history", "--stats")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "statistics")
	})

	t.Run("Show History With Limit", func(t *testing.T) {
		output, err := ts.RunCommand(t, "history", "--limit", "5")
		ts.AssertSuccess(t, output, err)
	})

	t.Run("Clear History", func(t *testing.T) {
		output, err := ts.RunCommand(t, "history", "--clear")
		ts.AssertSuccess(t, output, err)

		// Verify history is cleared
		historyOutput, err := ts.RunCommand(t, "history")
		ts.AssertSuccess(t, historyOutput, err)
		ts.AssertNotContains(t, historyOutput, "2.74.0")
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Use Non-existent Version", func(t *testing.T) {
		output, err := ts.RunCommand(t, "use", "999.999.999")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "not found")
	})

	t.Run("Remove Non-existent Version", func(t *testing.T) {
		output, err := ts.RunCommand(t, "remove", "999.999.999")
		ts.AssertFailure(t, output, err)
	})

	t.Run("Invalid Command", func(t *testing.T) {
		output, err := ts.RunCommand(t, "invalid-command")
		ts.AssertFailure(t, output, err)
	})

	t.Run("Missing Required Arguments", func(t *testing.T) {
		output, err := ts.RunCommand(t, "install")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "Usage")
	})
}

// TestConcurrentOperations tests concurrent operations
func TestConcurrentOperations(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Concurrent Installs", func(t *testing.T) {
		// This test would need to be implemented with goroutines
		// For now, we'll test that basic operations work
		output, err := ts.RunCommand(t, "install", "2.74.0")
		ts.AssertSuccess(t, output, err)
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Install Performance", func(t *testing.T) {
		start := time.Now()
		output, err := ts.RunCommandWithTimeout(t, 60*time.Second, "install", "2.74.0")
		duration := time.Since(start)

		ts.AssertSuccess(t, output, err)
		if duration > 30*time.Second {
			t.Errorf("Install took too long: %v", duration)
		}
	})

	t.Run("List Performance", func(t *testing.T) {
		start := time.Now()
		output, err := ts.RunCommand(t, "list")
		duration := time.Since(start)

		ts.AssertSuccess(t, output, err)
		if duration > 5*time.Second {
			t.Errorf("List took too long: %v", duration)
		}
	})
}

// TestIntegration tests integration scenarios
func TestIntegration(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Full Workflow", func(t *testing.T) {
		// Install multiple versions
		ts.RunCommand(t, "install", "2.74.0")
		ts.RunCommand(t, "install", "2.73.0")

		// Set up aliases
		ts.RunCommand(t, "alias", "set", "prod", "2.73.0")
		ts.RunCommand(t, "alias", "set", "dev", "2.74.0")

		// Use aliases
		ts.RunCommand(t, "use", "prod")
		ts.RunCommand(t, "use", "dev")

		// Compare versions
		ts.RunCommand(t, "compare", "cli", "prod", "dev", "--", "--version")

		// Benchmark
		ts.RunCommand(t, "benchmark", "prod,dev", "--", "--version")

		// Check history
		ts.RunCommand(t, "history")

		// Clean up
		ts.RunCommand(t, "clear")
	})
}

// TestPlatformSpecific tests platform-specific functionality
func TestPlatformSpecific(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Platform Detection", func(t *testing.T) {
		output, err := ts.RunCommand(t, "list")
		ts.AssertSuccess(t, output, err)

		// Should work on all platforms
		ts.AssertContains(t, output, "jfcm")
	})
}

// TestSecurity tests security-related functionality
func TestSecurity(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Binary Permissions", func(t *testing.T) {
		ts.RunCommand(t, "install", "2.74.0")

		// Check that binary has correct permissions
		binaryPath := filepath.Join(os.Getenv("HOME"), ".jfcm", "versions", "2.74.0", "jf")
		if info, err := os.Stat(binaryPath); err == nil {
			mode := info.Mode()
			if mode&0111 == 0 {
				t.Errorf("Binary should be executable")
			}
		}
	})
}

// TestShimAndPATH tests shim and PATH functionality
func TestShimAndPATH(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	// Install a version first
	ts.RunCommand(t, "install", "2.74.0")

	t.Run("Use Version with Automatic Shim Setup", func(t *testing.T) {
		output, err := ts.RunCommand(t, "use", "2.74.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Successfully activated")
		ts.AssertContains(t, output, "takes priority over system jf")
		ts.AssertContains(t, output, "Setting up jf shim")
		ts.AssertContains(t, output, "Updating PATH")
	})

	t.Run("Shim File Exists", func(t *testing.T) {
		shimPath := filepath.Join(os.Getenv("HOME"), ".jfcm", "shim", "jf")
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Errorf("Shim file should exist at %s", shimPath)
		}
	})

	t.Run("Shim is Executable", func(t *testing.T) {
		shimPath := filepath.Join(os.Getenv("HOME"), ".jfcm", "shim", "jf")
		if info, err := os.Stat(shimPath); err == nil {
			mode := info.Mode()
			if mode&0111 == 0 {
				t.Errorf("Shim should be executable")
			}
		}
	})

	t.Run("Use Latest with Shim Setup", func(t *testing.T) {
		output, err := ts.RunCommandWithTimeout(t, 30*time.Second, "use", "latest")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Successfully activated")
		ts.AssertContains(t, output, "takes priority over system jf")
	})

	t.Run("Health Check", func(t *testing.T) {
		output, err := ts.RunCommand(t, "health-check")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm Health Check")
		ts.AssertContains(t, output, "Overall Status")
	})

	t.Run("Health Check with Fix", func(t *testing.T) {
		output, err := ts.RunCommand(t, "health-check", "--fix")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm Health Check")
	})

	t.Run("Health Check Verbose", func(t *testing.T) {
		output, err := ts.RunCommand(t, "health-check", "--verbose")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm Health Check")
		ts.AssertContains(t, output, "Details:")
	})
}

// TestChangelogFunctionality tests changelog functionality
func TestChangelogFunctionality(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Compare Command Structure", func(t *testing.T) {
		// Test main compare command shows subcommands
		output, err := ts.RunCommand(t, "compare", "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "changelog")
		ts.AssertContains(t, output, "cli")
		ts.AssertContains(t, output, "subcommands")
	})

	t.Run("Changelog Subcommand Help", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "changelog", "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Compare release notes between two versions")
		ts.AssertContains(t, output, "<version1> <version2>")
	})

	t.Run("CLI Subcommand Help", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "cli", "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Compare JFrog CLI command execution between two versions")
		ts.AssertContains(t, output, "<version1> <version2> -- <jf-command>")
		ts.AssertContains(t, output, "--unified")
	})

	t.Run("Fetch Release Notes Between Versions", func(t *testing.T) {
		// Test fetching changelog between two JFrog CLI versions using compare changelog
		output, err := ts.RunCommandWithTimeout(t, 60*time.Second, "compare", "changelog", "v2.50.0", "v2.52.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Release Notes")
	})

	t.Run("Fetch Release Notes With Aliases", func(t *testing.T) {
		// Test fetching changelog with version aliases
		output, err := ts.RunCommandWithTimeout(t, 60*time.Second, "compare", "changelog", "v2.50.0", "v2.51.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Release Notes")
	})

	t.Run("Invalid Version Tags", func(t *testing.T) {
		output, err := ts.RunCommandWithTimeout(t, 30*time.Second, "compare", "changelog", "v999.999.999", "v999.999.998")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "error")
	})

	t.Run("Missing Arguments for Changelog", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "changelog", "v2.50.0")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "Usage")
	})

	t.Run("Filtered Release Notes", func(t *testing.T) {
		// Test that release notes are properly filtered (removing "New Contributors" etc.)
		output, err := ts.RunCommandWithTimeout(t, 60*time.Second, "compare", "changelog", "v2.50.0", "v2.51.0")
		ts.AssertSuccess(t, output, err)
		// Should not contain "New Contributors" section
		ts.AssertNotContains(t, output, "## New Contributors")
	})

	t.Run("Changelog With Same Version", func(t *testing.T) {
		// Test edge case where from and to versions are the same
		output, err := ts.RunCommandWithTimeout(t, 30*time.Second, "compare", "changelog", "v2.50.0", "v2.50.0")
		// This should either work (showing just that version) or fail gracefully
		if err != nil {
			ts.AssertContains(t, output, "same version")
		} else {
			ts.AssertContains(t, output, "v2.50.0")
		}
	})

	t.Run("Network Timeout Handling", func(t *testing.T) {
		// Test with a very short timeout to simulate network issues
		output, err := ts.RunCommandWithTimeout(t, 1*time.Second, "compare", "changelog", "v2.50.0", "v2.52.0")
		// Should either succeed quickly or fail with timeout
		if err != nil {
			// Timeout or network error is acceptable for this test
			t.Logf("Expected timeout or network error: %v, output: %s", err, output)
		}
	})

	t.Run("Large Version Range", func(t *testing.T) {
		// Test fetching changelog across many versions (should be limited to 5)
		output, err := ts.RunCommandWithTimeout(t, 90*time.Second, "compare", "changelog", "v2.40.0", "v2.52.0")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Release Notes")
		// Should limit to maximum 5 releases
		releaseCount := strings.Count(output, "## ")
		if releaseCount > 5 {
			t.Errorf("Expected maximum 5 releases, but found %d", releaseCount)
		}
	})
}

// TestRTCompareFunctionality tests the RT compare functionality with --server-id flag
func TestRTCompareFunctionality(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("RT Compare Command Structure", func(t *testing.T) {
		// Test RT compare command shows proper help
		output, err := ts.RunCommand(t, "compare", "rt", "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "Compare JFrog CLI command execution between two servers")
		ts.AssertContains(t, output, "<server1> <server2> -- <jf-command>")
		ts.AssertContains(t, output, "--unified")
		ts.AssertContains(t, output, "--timeout")
	})

	t.Run("RT Compare Missing Arguments", func(t *testing.T) {
		// Test missing server arguments
		output, err := ts.RunCommand(t, "compare", "rt", "server1")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "insufficient arguments")

		// Test missing separator
		output, err = ts.RunCommand(t, "compare", "rt", "server1", "server2", "rt", "ping")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "missing '--' separator")

		// Test missing command after separator
		output, err = ts.RunCommand(t, "compare", "rt", "server1", "server2", "--")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "no command specified after '--'")
	})

	t.Run("RT Compare Invalid Separator Position", func(t *testing.T) {
		// Test separator in wrong position
		output, err := ts.RunCommand(t, "compare", "rt", "server1", "--", "server2", "rt", "ping")
		ts.AssertFailure(t, output, err)
		ts.AssertContains(t, output, "'--' separator must come after <server1> <server2>")
	})

	t.Run("RT Compare Command Execution", func(t *testing.T) {
		// Test that the command structure is correct (we can't test actual execution without real servers)
		// This test verifies the command parsing and argument validation works correctly
		output, err := ts.RunCommand(t, "compare", "rt", "test-server1", "test-server2", "--", "rt", "ping")
		// We expect this to fail because the servers don't exist, but the parsing should work
		// The error should be about server connectivity, not argument parsing
		if err == nil {
			t.Error("Expected command to fail due to non-existent servers, but it succeeded")
		}
		// The output should not contain argument parsing errors
		if strings.Contains(output, "insufficient arguments") ||
			strings.Contains(output, "missing '--' separator") ||
			strings.Contains(output, "no command specified") {
			t.Errorf("Unexpected argument parsing error: %s", output)
		}
	})

	t.Run("RT Compare With Complex Command", func(t *testing.T) {
		// Test with a more complex command that has multiple arguments
		output, err := ts.RunCommand(t, "compare", "rt", "server1", "server2", "--", "rt", "search", "*.jar", "--limit", "10")
		// Again, we expect this to fail due to non-existent servers, not argument parsing
		if err == nil {
			t.Error("Expected command to fail due to non-existent servers, but it succeeded")
		}
		// Should not have argument parsing errors
		if strings.Contains(output, "insufficient arguments") ||
			strings.Contains(output, "missing '--' separator") ||
			strings.Contains(output, "no command specified") {
			t.Errorf("Unexpected argument parsing error: %s", output)
		}
	})

	t.Run("RT Compare With Options", func(t *testing.T) {
		// Test with various command options
		output, err := ts.RunCommand(t, "compare", "rt", "server1", "server2", "--", "rt", "ping", "--timeout", "30", "--unified")
		if err == nil {
			t.Error("Expected command to fail due to non-existent servers, but it succeeded")
		}
		// Should not have argument parsing errors
		if strings.Contains(output, "insufficient arguments") ||
			strings.Contains(output, "missing '--' separator") ||
			strings.Contains(output, "no command specified") {
			t.Errorf("Unexpected argument parsing error: %s", output)
		}
	})

	t.Run("RT Compare Help Examples", func(t *testing.T) {
		// Test that help shows proper examples
		output, err := ts.RunCommand(t, "compare", "rt", "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm compare rt server1 server2 -- rt ping")
		ts.AssertContains(t, output, "Compare rt ping command across two servers")
	})
}
