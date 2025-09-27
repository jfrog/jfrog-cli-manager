package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
)

// validateRTArguments validates RT-specific arguments and returns server names and command parts
func validateRTArguments(args []string) (string, string, []string, error) {
	if len(args) < 3 {
		return "", "", nil, fmt.Errorf("insufficient arguments: need <server1> <server2> -- <command>")
	}

	// Find the separator "--"
	separatorIndex := findSeparator(args, "--")
	if separatorIndex == -1 {
		return "", "", nil, fmt.Errorf("missing '--' separator")
	}

	// Separator must be after at least 2 arguments (server1 and server2)
	if separatorIndex < 2 {
		return "", "", nil, fmt.Errorf("'--' separator must come after <server1> <server2>")
	}

	if len(args) <= separatorIndex+1 {
		return "", "", nil, fmt.Errorf("no command specified after '--'")
	}

	server1 := args[0]
	server2 := args[1]
	jfCommand := args[separatorIndex+1:]

	return server1, server2, jfCommand, nil
}

// executeJFCommandOnServer executes a JFrog CLI command on the specified server
func executeJFCommandOnServer(ctx context.Context, serverName string, jfCommand []string) (ExecutionResult, error) {
	result := ExecutionResult{
		Version:   serverName, // Use server name as "version" for display purposes
		Command:   strings.Join(jfCommand, " "),
		StartTime: time.Now(),
	}

	// Get the active jf binary path
	binaryPath, err := utils.GetActiveBinaryPath()
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("Failed to get active jf binary: %v", err)
		result.ExitCode = 1
		result.Duration = time.Since(result.StartTime)
		return result, err
	}

	// Add --server-id as a global flag before the subcommand for broad compatibility
	commandArgs := append([]string{"--server-id", serverName}, jfCommand...)

	// Execute the command with --server-id flag
	cmd := exec.CommandContext(ctx, binaryPath, commandArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result.Duration = time.Since(result.StartTime)

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		result.ErrorMsg = stderrStr

		// For failed commands, still capture stdout as output (help commands often write to stdout even on failure)
		if stdoutStr != "" {
			result.Output = stdoutStr
		} else if stderrStr != "" {
			result.Output = stderrStr
		}
	} else {
		// When command succeeds, combine stdout and stderr for output comparison
		// Many CLI tools write informational messages to stderr even on success
		if stdoutStr != "" && stderrStr != "" {
			result.Output = stdoutStr + "\n" + stderrStr
		} else if stdoutStr != "" {
			result.Output = stdoutStr
		} else {
			result.Output = stderrStr
		}
	}

	return result, nil
}
