package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
)

// Constants for CLI comparison functionality
const (
	DefaultContextSize      = 3
	MaxDisplayLineLength    = 48
	TruncatedLineSuffix     = "..."
	PerformanceThresholdX   = 2 // 2x slower threshold
	TruncateIndicatorLength = 3 // Length of "..."
)

// ColorScheme manages all colors used in the CLI comparison functionality
type ColorScheme struct {
	Red     *color.Color
	Green   *color.Color
	Blue    *color.Color
	Yellow  *color.Color
	Cyan    *color.Color
	Magenta *color.Color
}

// NewColorScheme creates a new color scheme with consistent styling
func NewColorScheme(noColor bool) *ColorScheme {
	if noColor {
		color.NoColor = true
	}

	return &ColorScheme{
		Red:     color.New(color.FgRed),
		Green:   color.New(color.FgGreen, color.Bold),
		Blue:    color.New(color.FgBlue, color.Bold),
		Yellow:  color.New(color.FgYellow),
		Cyan:    color.New(color.FgCyan, color.Bold),
		Magenta: color.New(color.FgMagenta),
	}
}

// ExecutionResult holds the result of executing a JFrog CLI command
type ExecutionResult struct {
	Version   string
	Command   string
	Output    string
	ErrorMsg  string
	ExitCode  int
	Duration  time.Duration
	StartTime time.Time
}

// diffChange represents a single change in a diff
type diffChange struct {
	lineNum    int
	changeType string // "added", "removed", "context"
	text       string
}

// validateCLIArguments validates CLI-specific arguments and returns command parts
func validateCLIArguments(args []string) ([]string, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("insufficient arguments: need <version1> <version2> -- <command>")
	}

	// Find the separator "--"
	separatorIndex := findSeparator(args, "--")
	if separatorIndex == -1 {
		return nil, fmt.Errorf("missing '--' separator")
	}

	// Separator must be after at least 2 arguments (version1 and version2)
	if separatorIndex < 2 {
		return nil, fmt.Errorf("'--' separator must come after <version1> <version2>")
	}

	if len(args) <= separatorIndex+1 {
		return nil, fmt.Errorf("no command specified after '--'")
	}

	return args[separatorIndex+1:], nil
}

// findSeparator finds the index of a separator in arguments
func findSeparator(args []string, separator string) int {
	for i, arg := range args {
		if arg == separator {
			return i
		}
	}
	return -1
}

// executeJFCommand executes a JFrog CLI command with the specified version
func executeJFCommand(ctx context.Context, version string, jfCommand []string) (ExecutionResult, error) {
	result := ExecutionResult{
		Version:   version,
		Command:   strings.Join(jfCommand, " "),
		StartTime: time.Now(),
	}

	binPath := filepath.Join(utils.jfcmVersions, version, utils.BinaryName)

	cmd := exec.CommandContext(ctx, binPath, jfCommand...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

// displayComparison displays the comparison results between two CLI executions
func displayComparison(result1, result2 ExecutionResult, unified, noColor, showTiming bool) {
	colors := NewColorScheme(noColor)

	displayComparisonHeader()

	if showTiming {
		displayTimingInfo(result1, result2, colors)
	}

	if result1.ExitCode != result2.ExitCode {
		displayExitCodeDiff(result1, result2, colors)
	}

	if result1.ErrorMsg != "" || result2.ErrorMsg != "" {
		displayErrorOutput(result1, result2, colors)
	}

	displayOutputDiff(result1, result2, unified, colors)
}

// displayComparisonHeader shows the comparison results header
func displayComparisonHeader() {
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════════\n")
	fmt.Printf("🔍 COMPARISON RESULTS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════════\n\n")
}

// displayTimingInfo shows execution timing information and performance comparison
func displayTimingInfo(result1, result2 ExecutionResult, colors *ColorScheme) {
	fmt.Printf("⏱️  EXECUTION TIMING:\n")
	fmt.Printf("   Version %s: %v\n", colors.Blue.Sprint(result1.Version), result1.Duration)
	fmt.Printf("   Version %s: %v\n", colors.Blue.Sprint(result2.Version), result2.Duration)

	// Add performance comparison
	if result1.Duration > result2.Duration*PerformanceThresholdX {
		fmt.Printf("   Performance: %s is significantly slower\n", colors.Yellow.Sprint(result1.Version))
	} else if result2.Duration > result1.Duration*PerformanceThresholdX {
		fmt.Printf("   Performance: %s is significantly slower\n", colors.Yellow.Sprint(result2.Version))
	} else if result1.Duration != result2.Duration {
		fmt.Printf("   Performance: Similar execution times\n")
	}
	fmt.Printf("\n")
}

// displayExitCodeDiff shows exit code differences between results
func displayExitCodeDiff(result1, result2 ExecutionResult, colors *ColorScheme) {
	fmt.Printf("🚨 EXIT CODE DIFFERENCE:\n")
	displaySingleExitCode(result1.Version, result1.ExitCode, colors)
	displaySingleExitCode(result2.Version, result2.ExitCode, colors)
	fmt.Printf("\n")
}

// displaySingleExitCode formats and displays a single exit code
func displaySingleExitCode(version string, exitCode int, colors *ColorScheme) {
	if exitCode == 0 {
		fmt.Printf("   %s: %s\n", version, colors.Green.Sprint("✓ 0"))
	} else {
		fmt.Printf("   %s: %s\n", version, colors.Red.Sprintf("✗ %d", exitCode))
	}
}

// displayErrorOutput shows error messages if any exist
func displayErrorOutput(result1, result2 ExecutionResult, colors *ColorScheme) {
	fmt.Printf("🚨 ERROR OUTPUT:\n")
	if result1.ErrorMsg != "" {
		fmt.Printf("   %s ERROR:\n%s\n", colors.Red.Sprint(result1.Version), result1.ErrorMsg)
	}
	if result2.ErrorMsg != "" {
		fmt.Printf("   %s ERROR:\n%s\n", colors.Red.Sprint(result2.Version), result2.ErrorMsg)
	}
	fmt.Printf("\n")
}

// displayOutputDiff compares and displays output differences
func displayOutputDiff(result1, result2 ExecutionResult, unified bool, colors *ColorScheme) {
	output1, output2 := prepareOutputsForComparison(result1, result2)

	// Check if outputs are identical
	if areOutputsIdentical(output1, output2, result1, result2) {
		displayIdenticalOutputs(output1)
		return
	}

	fmt.Printf("📊 OUTPUT DIFFERENCES:\n")

	if unified {
		displayUnifiedDiff(output1, output2, result1.Version, result2.Version, colors)
	} else {
		displayTableComparison(output1, output2, result1.Version, result2.Version, colors)
	}
}

// prepareOutputsForComparison prepares outputs for comparison, handling error fallback
func prepareOutputsForComparison(result1, result2 ExecutionResult) (string, string) {
	output1 := strings.TrimSpace(result1.Output)
	if output1 == "" && result1.ErrorMsg != "" {
		output1 = strings.TrimSpace(result1.ErrorMsg)
	}

	output2 := strings.TrimSpace(result2.Output)
	if output2 == "" && result2.ErrorMsg != "" {
		output2 = strings.TrimSpace(result2.ErrorMsg)
	}

	return output1, output2
}

// areOutputsIdentical checks if outputs are considered identical
func areOutputsIdentical(output1, output2 string, result1, result2 ExecutionResult) bool {
	// Commands with different exit codes should never be considered identical
	// Even if their stdout happens to be the same, they represent different execution results
	return output1 == output2 && result1.ExitCode == result2.ExitCode && result1.ErrorMsg == result2.ErrorMsg
}

// displayIdenticalOutputs shows when outputs are identical
func displayIdenticalOutputs(output string) {
	fmt.Printf("✅ OUTPUTS ARE IDENTICAL\n")
	if output != "" {
		lineCount := len(strings.Split(output, "\n"))
		fmt.Printf("📄 Output (%d lines):\n", lineCount)
		fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")
		fmt.Printf("%s\n", output)
	}
}

// displayUnifiedDiff displays output differences in unified diff format
func displayUnifiedDiff(output1, output2, version1, version2 string, colors *ColorScheme) {
	lines1 := strings.Split(output1, "\n")
	lines2 := strings.Split(output2, "\n")

	// Header
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")
	fmt.Printf("%s %s\n", colors.Red.Sprint("---"), colors.Cyan.Sprint(version1))
	fmt.Printf("%s %s\n", colors.Green.Sprint("+++"), colors.Cyan.Sprint(version2))
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")

	// Create a simple line-based diff
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	// Track context for cleaner output
	contextSize := DefaultContextSize
	changes := []diffChange{}

	// Identify all changes first
	for i := 0; i < maxLines; i++ {
		line1 := ""
		line2 := ""

		if i < len(lines1) {
			line1 = strings.TrimSpace(lines1[i])
		}
		if i < len(lines2) {
			line2 = strings.TrimSpace(lines2[i])
		}

		if line1 != line2 {
			if line1 != "" && line2 == "" {
				changes = append(changes, diffChange{lineNum: i + 1, changeType: "removed", text: line1})
			} else if line1 == "" && line2 != "" {
				changes = append(changes, diffChange{lineNum: i + 1, changeType: "added", text: line2})
			} else if line1 != "" && line2 != "" {
				changes = append(changes, diffChange{lineNum: i + 1, changeType: "removed", text: line1})
				changes = append(changes, diffChange{lineNum: i + 1, changeType: "added", text: line2})
			}
		} else if line1 != "" {
			changes = append(changes, diffChange{lineNum: i + 1, changeType: "context", text: line1})
		}
	}

	// Display changes with context
	for i, change := range changes {
		switch change.changeType {
		case "removed":
			fmt.Printf("%s\n", colors.Red.Sprintf("- %s", change.text))
		case "added":
			fmt.Printf("%s\n", colors.Green.Sprintf("+ %s", change.text))
		case "context":
			// Only show context lines near changes
			showContext := false
			for j := max(0, i-contextSize); j <= min(len(changes)-1, i+contextSize); j++ {
				if changes[j].changeType != "context" {
					showContext = true
					break
				}
			}
			if showContext {
				fmt.Printf("  %s\n", change.text)
			}
		}
	}
}

// displayTableComparison displays output differences in a side-by-side table format
func displayTableComparison(output1, output2, version1, version2 string, colors *ColorScheme) {
	lines1 := strings.Split(output1, "\n")
	lines2 := strings.Split(output2, "\n")

	// Create clean table header - removed Status column, optimized width
	fmt.Printf("┌─────┬──────────────────────────────────────────────────┬──────────────────────────────────────────────────┐\n")
	headerLine := fmt.Sprintf("│ %s │ %s │ %s │",
		colors.Cyan.Sprintf("%-3s", "Line"),
		colors.Cyan.Sprintf("%-*s", MaxDisplayLineLength, version1),
		colors.Cyan.Sprintf("%-*s", MaxDisplayLineLength, version2))
	fmt.Println(headerLine)
	fmt.Printf("├─────┼──────────────────────────────────────────────────┼──────────────────────────────────────────────────┤\n")

	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	for i := 0; i < maxLines; i++ {
		line1 := ""
		line2 := ""

		if i < len(lines1) {
			line1 = strings.TrimSpace(lines1[i])
		}
		if i < len(lines2) {
			line2 = strings.TrimSpace(lines2[i])
		}

		// Skip empty lines for both versions to reduce noise
		if line1 == "" && line2 == "" {
			continue
		}

		// Increased line length for better readability - show more text
		if len(line1) > MaxDisplayLineLength {
			line1 = line1[:MaxDisplayLineLength-TruncateIndicatorLength] + TruncatedLineSuffix
		}
		if len(line2) > MaxDisplayLineLength {
			line2 = line2[:MaxDisplayLineLength-TruncateIndicatorLength] + TruncatedLineSuffix
		}

		lineNum := fmt.Sprintf("%d", i+1)

		// Create table row - removed status column
		if line1 == line2 {
			// Same lines - no special coloring needed
			fmt.Printf("│ %-3s │ %-*s │ %-*s │\n", lineNum, MaxDisplayLineLength, line1, MaxDisplayLineLength, line2)
		} else if line1 != "" && line2 == "" {
			// Removed line - red
			fmt.Printf("│ %-3s │ %s │ %-*s │\n",
				lineNum,
				colors.Red.Sprintf("%-*s", MaxDisplayLineLength, line1),
				MaxDisplayLineLength, "")
		} else if line1 == "" && line2 != "" {
			// Added line - green
			fmt.Printf("│ %-3s │ %-*s │ %s │\n",
				lineNum,
				MaxDisplayLineLength, "",
				colors.Green.Sprintf("%-*s", MaxDisplayLineLength, line2))
		} else {
			// Modified line - yellow
			fmt.Printf("│ %-3s │ %s │ %s │\n",
				lineNum,
				colors.Yellow.Sprintf("%-*s", MaxDisplayLineLength, line1),
				colors.Yellow.Sprintf("%-*s", MaxDisplayLineLength, line2))
		}
	}

	// Table footer - adjusted for 3 columns with 48-char width
	fmt.Printf("└─────┴──────────────────────────────────────────────────┴──────────────────────────────────────────────────┘\n")

	// Simplified legend - colors speak for themselves
	fmt.Printf("\n📋 Legend: %s Added │ %s Removed │ %s Modified │ Normal = Same\n",
		colors.Green.Sprint("Green"),
		colors.Red.Sprint("Red"),
		colors.Yellow.Sprint("Yellow"))
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
