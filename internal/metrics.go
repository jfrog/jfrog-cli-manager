package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AppendLocalJFcmMetric writes a minimal metric event for a jfcm command
// The event is appended as a single line to ~/.jfrog/jfcm/metrics.log in the form:
// {product_id="jfcm",feature_id="<command>"}
func AppendLocalJFcmMetric(commandName string) {
	if commandName == "" {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	baseDir := filepath.Join(home, ".jfrog", "jfcm")
	_ = os.MkdirAll(baseDir, 0700)

	filePath := filepath.Join(baseDir, "metrics.log")

	// RFC3339 UTC timestamp
	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("{product_id=\"jfcm\",feature_id=\"%s\",timestamp=\"%s\"}\n", commandName, ts)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.WriteString(line)
}
