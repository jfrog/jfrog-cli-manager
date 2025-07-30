package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

type ExecutionResult struct {
	Version   string
	Command   string
	Output    string
	ErrorMsg  string
	ExitCode  int
	Duration  time.Duration
	StartTime time.Time
}

var Compare = &cli.Command{
	Name:        "compare",
	Usage:       descriptions.Compare.Usage,
	ArgsUsage:   "<version1> <version2> -- <jf-command> [args...]",
	Description: descriptions.Compare.Format(),
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "unified",
			Usage: "Show unified diff format instead of side-by-side",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "table",
			Usage: "Show clean table format (default)",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Command timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:  "timing",
			Usage: "Show execution timing information",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		args := c.Args().Slice()
		if len(args) < 3 {
			return cli.Exit("Usage: jfrog-cli-vm compare <version1> <version2> -- <jf-command> [args...]", 1)
		}

		// Find the separator "--"
		separatorIndex := -1
		for i, arg := range args {
			if arg == "--" {
				separatorIndex = i
				break
			}
		}

		if separatorIndex == -1 || separatorIndex != 2 {
			return cli.Exit("Missing '--' separator. Usage: jfrog-cli-vm compare <version1> <version2> -- <jf-command> [args...]", 1)
		}

		version1 := args[0]
		version2 := args[1]
		jfCommand := args[3:]

		if len(jfCommand) == 0 {
			return cli.Exit("No JFrog CLI command specified after '--'", 1)
		}

		// Resolve aliases if needed
		resolved1, err := utils.ResolveVersionOrAlias(version1)
		if err != nil {
			resolved1 = version1
		}
		resolved2, err := utils.ResolveVersionOrAlias(version2)
		if err != nil {
			resolved2 = version2
		}

		// Check if versions exist
		if err := utils.CheckVersionExists(resolved1); err != nil {
			return fmt.Errorf("version %s (%s) not found: %w", version1, resolved1, err)
		}
		if err := utils.CheckVersionExists(resolved2); err != nil {
			return fmt.Errorf("version %s (%s) not found: %w", version2, resolved2, err)
		}

		fmt.Printf("🔄 Comparing JFrog CLI versions: %s vs %s\n", version1, version2)
		fmt.Printf("📝 Command: jf %s\n\n", strings.Join(jfCommand, " "))

		// Execute commands in parallel
		results := make([]ExecutionResult, 2)
		g, ctx := errgroup.WithContext(context.Background())

		timeout := time.Duration(c.Int("timeout")) * time.Second
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		g.Go(func() error {
			result, err := executeJFCommand(timeoutCtx, resolved1, jfCommand)
			results[0] = result
			return err
		})

		g.Go(func() error {
			result, err := executeJFCommand(timeoutCtx, resolved2, jfCommand)
			results[1] = result
			return err
		})

		if err := g.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: %v\n\n", err)
		}

		// Display results
		displayComparison(results[0], results[1], c.Bool("unified"), c.Bool("no-color"), c.Bool("timing"))

		return nil
	},
}

func executeJFCommand(ctx context.Context, version string, jfCommand []string) (ExecutionResult, error) {
	result := ExecutionResult{
		Version:   version,
		Command:   strings.Join(jfCommand, " "),
		StartTime: time.Now(),
	}

	binPath := filepath.Join(utils.JfvmVersions, version, utils.BinaryName)

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

func displayComparison(result1, result2 ExecutionResult, unified, noColor, showTiming bool) {
	// Setup colors
	var (
		redColor    = color.New(color.FgRed)
		greenColor  = color.New(color.FgGreen)
		blueColor   = color.New(color.FgBlue)
		yellowColor = color.New(color.FgYellow)
	)

	if noColor {
		color.NoColor = true
	}

	// Display headers
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════════\n")
	fmt.Printf("🔍 COMPARISON RESULTS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════════\n\n")

	// Display timing information
	if showTiming {
		fmt.Printf("⏱️  EXECUTION TIMING:\n")
		fmt.Printf("   Version %s: %v\n", blueColor.Sprint(result1.Version), result1.Duration)
		fmt.Printf("   Version %s: %v\n", blueColor.Sprint(result2.Version), result2.Duration)

		// Add performance comparison
		if result1.Duration > result2.Duration*2 {
			fmt.Printf("   Performance: %s is significantly slower\n", yellowColor.Sprint(result1.Version))
		} else if result2.Duration > result1.Duration*2 {
			fmt.Printf("   Performance: %s is significantly slower\n", yellowColor.Sprint(result2.Version))
		} else if result1.Duration != result2.Duration {
			fmt.Printf("   Performance: Similar execution times\n")
		}
		fmt.Printf("\n")
	}

	// Display exit codes if different
	if result1.ExitCode != result2.ExitCode {
		fmt.Printf("🚨 EXIT CODE DIFFERENCE:\n")
		if result1.ExitCode == 0 {
			fmt.Printf("   %s: %s\n", result1.Version, greenColor.Sprint("✓ 0"))
		} else {
			fmt.Printf("   %s: %s\n", result1.Version, redColor.Sprintf("✗ %d", result1.ExitCode))
		}
		if result2.ExitCode == 0 {
			fmt.Printf("   %s: %s\n", result2.Version, greenColor.Sprint("✓ 0"))
		} else {
			fmt.Printf("   %s: %s\n", result2.Version, redColor.Sprintf("✗ %d", result2.ExitCode))
		}
		fmt.Printf("\n")
	}

	// Display errors if any
	if result1.ErrorMsg != "" || result2.ErrorMsg != "" {
		fmt.Printf("🚨 ERROR OUTPUT:\n")
		if result1.ErrorMsg != "" {
			fmt.Printf("   %s ERROR:\n%s\n", redColor.Sprint(result1.Version), result1.ErrorMsg)
		}
		if result2.ErrorMsg != "" {
			fmt.Printf("   %s ERROR:\n%s\n", redColor.Sprint(result2.Version), result2.ErrorMsg)
		}
		fmt.Printf("\n")
	}

	// Compare outputs
	// For failed commands, use ErrorMsg if Output is empty (help commands often write to stdout even on failure)
	output1 := strings.TrimSpace(result1.Output)
	if output1 == "" && result1.ErrorMsg != "" {
		output1 = strings.TrimSpace(result1.ErrorMsg)
	}

	output2 := strings.TrimSpace(result2.Output)
	if output2 == "" && result2.ErrorMsg != "" {
		output2 = strings.TrimSpace(result2.ErrorMsg)
	}

	// Commands with different exit codes should never be considered identical
	// Even if their stdout happens to be the same, they represent different execution results
	if output1 == output2 && result1.ExitCode == result2.ExitCode && result1.ErrorMsg == result2.ErrorMsg {
		fmt.Printf("✅ OUTPUTS ARE IDENTICAL\n")
		if output1 != "" {
			lineCount := len(strings.Split(output1, "\n"))
			fmt.Printf("📄 Output (%d lines):\n", lineCount)
			fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")
			fmt.Printf("%s\n", output1)
		}
		return
	}

	fmt.Printf("📊 OUTPUT DIFFERENCES:\n")

	if unified {
		displayUnifiedDiff(output1, output2, result1.Version, result2.Version, noColor)
	} else {
		displayTableComparison(output1, output2, result1.Version, result2.Version, noColor)
	}
}

func displayUnifiedDiff(output1, output2, version1, version2 string, noColor bool) {
	lines1 := strings.Split(output1, "\n")
	lines2 := strings.Split(output2, "\n")

	var (
		redColor   = color.New(color.FgRed)
		greenColor = color.New(color.FgGreen)
		cyanColor  = color.New(color.FgCyan, color.Bold)
	)

	// Header
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")
	if !noColor {
		fmt.Printf("%s %s\n", redColor.Sprint("---"), cyanColor.Sprint(version1))
		fmt.Printf("%s %s\n", greenColor.Sprint("+++"), cyanColor.Sprint(version2))
	} else {
		fmt.Printf("--- %s\n", version1)
		fmt.Printf("+++ %s\n", version2)
	}
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────\n")

	// Create a simple line-based diff
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	// Track context for cleaner output
	contextSize := 3
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
			if !noColor {
				fmt.Printf("%s\n", redColor.Sprintf("- %s", change.text))
			} else {
				fmt.Printf("- %s\n", change.text)
			}
		case "added":
			if !noColor {
				fmt.Printf("%s\n", greenColor.Sprintf("+ %s", change.text))
			} else {
				fmt.Printf("+ %s\n", change.text)
			}
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

type diffChange struct {
	lineNum    int
	changeType string // "added", "removed", "context"
	text       string
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func displayTableComparison(output1, output2, version1, version2 string, noColor bool) {
	lines1 := strings.Split(output1, "\n")
	lines2 := strings.Split(output2, "\n")

	var (
		greenColor  = color.New(color.FgGreen)
		redColor    = color.New(color.FgRed)
		yellowColor = color.New(color.FgYellow)
		cyanColor   = color.New(color.FgCyan, color.Bold)
	)

	// Create clean table header - removed Status column, optimized width
	fmt.Printf("┌─────┬──────────────────────────────────────────────────┬──────────────────────────────────────────────────┐\n")
	headerLine := fmt.Sprintf("│ %-3s │ %-48s │ %-48s │", "Line", version1, version2)
	if !noColor {
		headerLine = fmt.Sprintf("│ %s │ %s │ %s │",
			cyanColor.Sprintf("%-3s", "Line"),
			cyanColor.Sprintf("%-48s", version1),
			cyanColor.Sprintf("%-48s", version2))
	}
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
		if len(line1) > 48 {
			line1 = line1[:45] + "..."
		}
		if len(line2) > 48 {
			line2 = line2[:45] + "..."
		}

		lineNum := fmt.Sprintf("%d", i+1)

		// Create table row - removed status column
		if line1 == line2 {
			// Same lines - no special coloring needed
			fmt.Printf("│ %-3s │ %-48s │ %-48s │\n", lineNum, line1, line2)
		} else if line1 != "" && line2 == "" {
			// Removed line - red
			if !noColor {
				fmt.Printf("│ %-3s │ %s │ %-48s │\n",
					lineNum,
					redColor.Sprintf("%-48s", line1),
					"")
			} else {
				fmt.Printf("│ %-3s │ %-48s │ %-48s │\n", lineNum, line1, "")
			}
		} else if line1 == "" && line2 != "" {
			// Added line - green
			if !noColor {
				fmt.Printf("│ %-3s │ %-48s │ %s │\n",
					lineNum,
					"",
					greenColor.Sprintf("%-48s", line2))
			} else {
				fmt.Printf("│ %-3s │ %-48s │ %-48s │\n", lineNum, "", line2)
			}
		} else {
			// Modified line - yellow
			if !noColor {
				fmt.Printf("│ %-3s │ %s │ %s │\n",
					lineNum,
					yellowColor.Sprintf("%-48s", line1),
					yellowColor.Sprintf("%-48s", line2))
			} else {
				fmt.Printf("│ %-3s │ %-48s │ %-48s │\n", lineNum, line1, line2)
			}
		}
	}

	// Table footer - adjusted for 3 columns with 48-char width
	fmt.Printf("└─────┴──────────────────────────────────────────────────┴──────────────────────────────────────────────────┘\n")

	// Simplified legend - colors speak for themselves
	fmt.Printf("\n📋 Legend: %s Added │ %s Removed │ %s Modified │ Normal = Same\n",
		greenColor.Sprint("Green"),
		redColor.Sprint("Red"),
		yellowColor.Sprint("Yellow"))
}
