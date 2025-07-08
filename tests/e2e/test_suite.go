package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSuite holds the test environment
type TestSuite struct {
	JfvmPath    string
	TestDir     string
	OriginalPWD string
}

// SetupTestSuite initializes the test environment
func SetupTestSuite(t *testing.T) *TestSuite {
	// Get the path to the jfvm binary
	jfvmPath := os.Getenv("JFVM_PATH")
	if jfvmPath == "" {
		// Default to current directory if not set
		jfvmPath = "./jfvm"
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", "jfvm-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

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
		JfvmPath:    jfvmPath,
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

// RunCommand executes a jfvm command and returns the output
func (ts *TestSuite) RunCommand(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command(ts.JfvmPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommandWithTimeout executes a jfvm command with timeout
func (ts *TestSuite) RunCommandWithTimeout(t *testing.T, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ts.JfvmPath, args...)
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

	t.Run("Compare Version Output", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "2.74.0", "2.73.0", "--", "--version")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "2.74.0")
		ts.AssertContains(t, output, "2.73.0")
	})

	t.Run("Compare With Unified Diff", func(t *testing.T) {
		output, err := ts.RunCommand(t, "compare", "2.74.0", "2.73.0", "--", "--version", "--unified")
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
		ts.RunCommand(t, "compare", "prod", "dev", "--", "--version")

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
		ts.AssertContains(t, output, "jfvm")
	})
}

// TestSecurity tests security-related functionality
func TestSecurity(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Binary Permissions", func(t *testing.T) {
		ts.RunCommand(t, "install", "2.74.0")

		// Check that binary has correct permissions
		binaryPath := filepath.Join(os.Getenv("HOME"), ".jfvm", "versions", "2.74.0", "jf")
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
		shimPath := filepath.Join(os.Getenv("HOME"), ".jfvm", "shim", "jf")
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Errorf("Shim file should exist at %s", shimPath)
		}
	})

	t.Run("Shim is Executable", func(t *testing.T) {
		shimPath := filepath.Join(os.Getenv("HOME"), ".jfvm", "shim", "jf")
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
		ts.AssertContains(t, output, "jfvm Health Check")
		ts.AssertContains(t, output, "Overall Status")
	})

	t.Run("Health Check with Fix", func(t *testing.T) {
		output, err := ts.RunCommand(t, "health-check", "--fix")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfvm Health Check")
	})

	t.Run("Health Check Verbose", func(t *testing.T) {
		output, err := ts.RunCommand(t, "health-check", "--verbose")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfvm Health Check")
		ts.AssertContains(t, output, "Details:")
	})
}
