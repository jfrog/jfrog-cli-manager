package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestHistoryBasic(t *testing.T) {
	// Run a jf command through the shim to ensure history is recorded
	cmd := exec.Command(os.Getenv("HOME")+"/.jfvm/shim/jf", "--version")
	_ = cmd.Run()

	// Run jfvm history to verify it works
	out, err := exec.Command("jfvm", "history").CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history failed: %v", err)
	}
	output := string(out)

	// Just verify the command runs without error
	if strings.Contains(output, "Error") {
		t.Errorf("history command returned error: %s", output)
	}
}
