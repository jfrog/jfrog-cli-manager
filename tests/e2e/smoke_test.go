package e2e

import (
	"testing"
)

// TestSmoke is a simple smoke test to verify the test infrastructure works
func TestSmoke(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.CleanupTestSuite(t)

	t.Run("Basic Smoke Test", func(t *testing.T) {
		// Test that jfcm binary exists and is executable
		output, err := ts.RunCommand(t, "--help")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm")
	})

	t.Run("Version Command", func(t *testing.T) {
		// Test that version command works
		output, err := ts.RunCommand(t, "--version")
		ts.AssertSuccess(t, output, err)
		ts.AssertContains(t, output, "jfcm")
	})
}
