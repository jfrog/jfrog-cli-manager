package e2e

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

type historyEntry struct {
	ID      int    `json:"id"`
	Version string `json:"version"`
	Command string `json:"command"`
}

func TestHistoryShowsIDs(t *testing.T) {
	// Run a jf command
	cmd := exec.Command("jf", "--version")
	_ = cmd.Run()
	time.Sleep(1 * time.Second) // ensure history is written

	// Run jfvm history
	out, err := exec.Command("jfvm", "history").CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history failed: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "ID") {
		t.Errorf("history output missing ID column:\n%s", output)
	}
	if !strings.Contains(output, "jf --version") {
		t.Errorf("history output missing command:\n%s", output)
	}
}

func TestHistoryReplayByID(t *testing.T) {
	// Run a jf command
	cmd := exec.Command("jf", "--version")
	_ = cmd.Run()
	time.Sleep(1 * time.Second)

	// Get the latest history entry's ID
	out, err := exec.Command("jfvm", "history", "--format", "json").CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history --format json failed: %v", err)
	}
	var entries []historyEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		t.Fatalf("failed to parse history json: %v\n%s", err, string(out))
	}
	if len(entries) == 0 {
		t.Fatal("no history entries found")
	}
	id := entries[0].ID // newest is first

	// Switch to a different version (simulate, if available)
	_ = exec.Command("jfvm", "use", entries[0].Version).Run()

	// Replay the command
	replayCmd := exec.Command("jfvm", "history", "!"+strconv.Itoa(id))
	replayOut, err := replayCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history !%d failed: %v\nOutput: %s", id, err, string(replayOut))
	}
	if !strings.Contains(string(replayOut), "jf version") && !strings.Contains(string(replayOut), "JFrog CLI version") {
		t.Errorf("Replay output missing expected content:\n%s", string(replayOut))
	}
}

func TestHistoryReplayNonExistentID(t *testing.T) {
	out, err := exec.Command("jfvm", "history", "!999999").CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for non-existent ID, got none. Output: %s", string(out))
	}
	if !strings.Contains(string(out), "not found") {
		t.Errorf("Expected 'not found' error, got: %s", string(out))
	}
}

func TestHistoryReplayZeroOrNegativeID(t *testing.T) {
	for _, id := range []string{"!0", "!-1"} {
		out, err := exec.Command("jfvm", "history", id).CombinedOutput()
		if err == nil {
			t.Errorf("Expected error for ID %s, got none. Output: %s", id, string(out))
		}
		if !strings.Contains(string(out), "not found") {
			t.Errorf("Expected 'not found' error for ID %s, got: %s", id, string(out))
		}
	}
}

func TestHistoryReplayNonIntegerID(t *testing.T) {
	out, err := exec.Command("jfvm", "history", "!abc").CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for non-integer ID, got none. Output: %s", string(out))
	}
	if !strings.Contains(string(out), "not found") && !strings.Contains(string(out), "invalid") {
		t.Errorf("Expected error for non-integer ID, got: %s", string(out))
	}
}

func TestHistoryReplayMissingVersion(t *testing.T) {
	// Run a jf command to ensure at least one entry
	cmd := exec.Command("jf", "--version")
	_ = cmd.Run()
	time.Sleep(1 * time.Second)

	out, err := exec.Command("jfvm", "history", "--format", "json").CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history --format json failed: %v", err)
	}
	var entries []historyEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		t.Fatalf("failed to parse history json: %v\n%s", err, string(out))
	}
	if len(entries) == 0 {
		t.Fatal("no history entries found")
	}
	id := entries[0].ID
	versionDir := "/Users/runner/.jfvm/versions/" + entries[0].Version // adjust path as needed for CI
	_ = exec.Command("rm", "-rf", versionDir).Run()

	replayCmd := exec.Command("jfvm", "history", "!"+strconv.Itoa(id))
	replayOut, err := replayCmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for missing version, got none. Output: %s", string(replayOut))
	}
	if !strings.Contains(string(replayOut), "not found") {
		t.Errorf("Expected 'not found' error for missing version, got: %s", string(replayOut))
	}
}

func TestHistoryNoEntries(t *testing.T) {
	_ = exec.Command("jfvm", "history", "--clear").Run()
	out, err := exec.Command("jfvm", "history").CombinedOutput()
	if err != nil {
		t.Fatalf("jfvm history failed: %v", err)
	}
	if !strings.Contains(string(out), "No history entries found") {
		t.Errorf("Expected 'No history entries found', got: %s", string(out))
	}
}
