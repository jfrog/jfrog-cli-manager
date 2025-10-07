package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var AddHistoryEntryCmd = &cli.Command{
	Name:        "add-history-entry",
	Usage:       "Add a history entry (internal use)",
	Description: "Internal command used by jfcm shim to record command execution history",
	Hidden:      true, // Hide this command from help output
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 5 {
			return fmt.Errorf("add-history-entry requires 5 arguments: version, command, duration_ms, exit_code, output")
		}

		version := c.Args().Get(0)
		command := c.Args().Get(1)
		durationStr := c.Args().Get(2)
		exitCodeStr := c.Args().Get(3)
		output := c.Args().Get(4)

		// Skip recording jfcm commands - only record actual jf commands
		if strings.HasPrefix(command, "jfcm ") {
			return nil
		}

		// Parse duration and exit code
		durationMs, err := strconv.ParseInt(durationStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}

		exitCode, err := strconv.Atoi(exitCodeStr)
		if err != nil {
			return fmt.Errorf("invalid exit code: %w", err)
		}

		// Record the history entry using the existing function
		AddHistoryEntry(version, command, time.Duration(durationMs)*time.Millisecond, exitCode, output, "")

		return nil
	},
}
