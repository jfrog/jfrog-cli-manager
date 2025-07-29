package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

type BenchmarkResult struct {
	Version     string
	Iterations  int
	TotalTime   time.Duration
	AverageTime time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	SuccessRate float64
	Executions  []ExecutionResult
}

var Benchmark = &cli.Command{
	Name:        "benchmark",
	Usage:       descriptions.Benchmark.Usage,
	ArgsUsage:   "<version1,version2,...> -- <jf-command> [args...]",
	Description: descriptions.Benchmark.Format(),
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "iterations",
			Usage: "Number of iterations per version",
			Value: 5,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Command timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "detailed",
			Usage: "Show detailed execution logs",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "format",
			Usage: "Output format: table, json, csv",
			Value: "table",
		},
	},
	Action: func(c *cli.Context) error {
		// Parse and validate arguments
		versions, jfCommand, err := parseArguments(c.Args().Slice())
		if err != nil {
			return err
		}

		// Validate versions exist
		resolvedVersions, err := validateVersions(versions)
		if err != nil {
			return err
		}

		// Extract configuration
		config := extractBenchmarkConfig(c)

		// Run benchmarks
		results, err := runBenchmarks(resolvedVersions, jfCommand, config)
		if err != nil && config.Format == "table" {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: %v\n\n", err)
		}

		// Display results
		displayBenchmarkResults(results, config.Format, config.NoColor, config.Detailed)

		return nil
	},
}

type BenchmarkConfig struct {
	Iterations int
	Timeout    time.Duration
	Format     string
	NoColor    bool
	Detailed   bool
}

func parseArguments(args []string) (versions []string, jfCommand []string, err error) {
	if len(args) < 2 {
		return nil, nil, cli.Exit("Usage: jfvm benchmark [flags] <version1,version2,...> -- <jf-command> [args...]", 1)
	}

	// Find the separator "--"
	separatorIndex := -1
	for i, arg := range args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}

	if separatorIndex == -1 {
		return nil, nil, cli.Exit("Missing '--' separator. Usage: jfvm benchmark [flags] <versions> -- <jf-command> [args...]", 1)
	}

	if separatorIndex == 0 {
		return nil, nil, cli.Exit("No versions specified. Usage: jfvm benchmark [flags] <versions> -- <jf-command> [args...]", 1)
	}

	// Check for flags placed after the versions but before "--"
	for i := 1; i < separatorIndex; i++ {
		if strings.HasPrefix(args[i], "--") {
			return nil, nil, cli.Exit(fmt.Sprintf("❌ Flag '%s' detected after versions. Please place all flags before versions.\nCorrect usage: jfvm benchmark --iterations 3 %s -- %s",
				args[i], args[0], strings.Join(args[separatorIndex+1:], " ")), 1)
		}
	}

	versionsStr := args[0]
	jfCommand = args[separatorIndex+1:]
	versions = strings.Split(versionsStr, ",")

	if len(jfCommand) == 0 {
		return nil, nil, cli.Exit("No JFrog CLI command specified after '--'", 1)
	}

	return versions, jfCommand, nil
}

func validateVersions(versions []string) ([]string, error) {
	resolvedVersions := make([]string, len(versions))
	for i, version := range versions {
		version = strings.TrimSpace(version)
		resolved, err := utils.ResolveVersionOrAlias(version)
		if err != nil {
			resolved = version
		}
		if err := utils.CheckVersionExists(resolved); err != nil {
			return nil, fmt.Errorf("version %s (%s) not found: %w", version, resolved, err)
		}
		resolvedVersions[i] = resolved
	}
	return resolvedVersions, nil
}

func extractBenchmarkConfig(c *cli.Context) BenchmarkConfig {
	return BenchmarkConfig{
		Iterations: c.Int("iterations"),
		Timeout:    time.Duration(c.Int("timeout")) * time.Second,
		Format:     c.String("format"),
		NoColor:    c.Bool("no-color"),
		Detailed:   c.Bool("detailed"),
	}
}

func runBenchmarks(versions []string, jfCommand []string, config BenchmarkConfig) ([]BenchmarkResult, error) {
	// Only show headers for table format
	if config.Format == "table" {
		fmt.Printf("🏁 Benchmarking JFrog CLI versions: %s\n", strings.Join(versions, ", "))
		fmt.Printf("📝 Command: jf %s\n", strings.Join(jfCommand, " "))
		fmt.Printf("🔄 Iterations: %d per version\n\n", config.Iterations)
	}

	// Run benchmarks
	results := make([]BenchmarkResult, len(versions))
	g, ctx := errgroup.WithContext(context.Background())

	for i, version := range versions {
		i, version := i, version
		g.Go(func() error {
			result, err := runBenchmark(ctx, version, jfCommand, config.Iterations, config.Timeout)
			results[i] = result
			return err
		})
	}

	return results, g.Wait()
}

func runBenchmark(ctx context.Context, version string, jfCommand []string, iterations int, timeout time.Duration) (BenchmarkResult, error) {
	result := BenchmarkResult{
		Version:    version,
		Iterations: iterations,
		MinTime:    time.Hour,
		Executions: make([]ExecutionResult, iterations),
	}

	successCount := 0

	for i := 0; i < iterations; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		exec, err := executeJFCommand(timeoutCtx, version, jfCommand)
		cancel()

		result.Executions[i] = exec
		result.TotalTime += exec.Duration

		if exec.ExitCode == 0 {
			successCount++
		}

		if exec.Duration < result.MinTime {
			result.MinTime = exec.Duration
		}
		if exec.Duration > result.MaxTime {
			result.MaxTime = exec.Duration
		}

		if err != nil {
			fmt.Printf("⚠️  Iteration %d for %s failed: %v\n", i+1, version, err)
		}
	}

	result.AverageTime = result.TotalTime / time.Duration(iterations)
	result.SuccessRate = float64(successCount) / float64(iterations) * 100

	return result, nil
}

func displayBenchmarkResults(results []BenchmarkResult, format string, noColor, detailed bool) {
	switch format {
	case "json":
		displayBenchmarkJSON(results)
	case "csv":
		displayBenchmarkCSV(results)
	default:
		displayEnhancedBenchmarkResults(results, noColor, detailed)
	}
}

func displayEnhancedBenchmarkResults(results []BenchmarkResult, noColor, detailed bool) {
	// JFrog brand colors
	var (
		jfrogGreen  = lipgloss.Color("#43C74A")
		jfrogOrange = lipgloss.Color("#FF6B35")
		jfrogBlue   = lipgloss.Color("#0052CC")
		mutedGray   = lipgloss.Color("#6B7280")
	)

	// Define beautiful styles using JFrog colors
	var (
		titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogGreen).
				Padding(0, 2).
				MarginBottom(1)

		cardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogBlue).
				Padding(1, 2).
				MarginBottom(1).
				MarginRight(2)

		winnerCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogGreen).
				Padding(1, 2).
				MarginBottom(1).
				MarginRight(2)

		summaryStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogBlue).
				Padding(1, 2).
				MarginTop(1)
	)

	// Handle no-color mode
	if noColor {
		titleStyle = titleStyle.Foreground(lipgloss.Color(""))
		cardStyle = cardStyle.BorderForeground(lipgloss.Color(""))
		winnerCardStyle = winnerCardStyle.BorderForeground(lipgloss.Color(""))
		summaryStyle = summaryStyle.BorderForeground(lipgloss.Color(""))
		jfrogGreen = lipgloss.Color("")
		jfrogOrange = lipgloss.Color("")
		jfrogBlue = lipgloss.Color("")
		mutedGray = lipgloss.Color("")
	}

	// Sort by average time (fastest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].AverageTime < results[j].AverageTime
	})

	// Display title
	fmt.Println(titleStyle.Render("🏁 JFROG CLI BENCHMARK RESULTS"))

	// Create performance cards
	var cards []string
	fastest := results[0].AverageTime

	for i, result := range results {
		style := cardStyle
		if i == 0 {
			style = winnerCardStyle
		}

		// Version header with performance badge
		versionHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB")).Render(result.Version)
		if i == 0 {
			badge := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(jfrogGreen).
				Padding(0, 1).
				MarginLeft(1).
				Render("🏆 FASTEST")
			versionHeader += badge
		} else {
			speedup := float64(result.AverageTime) / float64(fastest)
			badge := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(jfrogBlue).
				Padding(0, 1).
				MarginLeft(1).
				Render(fmt.Sprintf("%.1fx slower", speedup))
			versionHeader += badge
		}

		// Performance metrics with better contrast
		metrics := fmt.Sprintf(
			"⚡ Avg: %s\n"+
				"🏃 Min: %s\n"+
				"🐌 Max: %s\n"+
				"⏱️  Total: %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Bold(true).Render(formatDuration(result.AverageTime)),
			lipgloss.NewStyle().Foreground(jfrogGreen).Bold(true).Render(formatDuration(result.MinTime)),
			lipgloss.NewStyle().Foreground(jfrogOrange).Bold(true).Render(formatDuration(result.MaxTime)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6")).Render(formatDuration(result.TotalTime)),
		)

		// Success rate with visual indicator
		successColor := jfrogGreen
		successIcon := "✅"
		if result.SuccessRate < 100 {
			successColor = jfrogOrange
			successIcon = "⚠️"
		}
		if result.SuccessRate < 80 {
			successColor = lipgloss.Color("#EF4444")
			successIcon = "❌"
		}

		successRate := fmt.Sprintf("%s Success: %s",
			successIcon,
			lipgloss.NewStyle().Foreground(successColor).Bold(true).Render(fmt.Sprintf("%.1f%%", result.SuccessRate)))

		// Iterations info
		iterationsInfo := lipgloss.NewStyle().Foreground(mutedGray).Italic(true).
			Render(fmt.Sprintf("📊 %d iterations", result.Iterations))

		cardContent := versionHeader + "\n\n" + metrics + "\n" + successRate + "\n" + iterationsInfo
		card := style.Width(28).Render(cardContent)
		cards = append(cards, card)
	}

	// Display cards in rows (max 3 per row)
	cardsPerRow := 3
	for i := 0; i < len(cards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...)
		fmt.Println(row)
	}

	// Performance summary
	summaryContent := createBenchmarkSummary(results, jfrogGreen, jfrogOrange, jfrogBlue, mutedGray, fastest)
	fmt.Println(summaryStyle.Width(90).Render(summaryContent))

	// Detailed execution logs if requested
	if detailed {
		fmt.Println()
		displayDetailedBenchmarkLogs(results, jfrogGreen, jfrogOrange, jfrogBlue)
	}
}

func createBenchmarkSummary(results []BenchmarkResult, jfrogGreen, jfrogOrange, jfrogBlue, mutedGray lipgloss.Color, fastest time.Duration) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(jfrogBlue).Render("📈 PERFORMANCE SUMMARY")

	content := header + "\n\n"

	// Winner info
	winner := results[0]
	content += fmt.Sprintf("🏆 Fastest Version: %s (%s average)\n",
		lipgloss.NewStyle().Foreground(jfrogGreen).Bold(true).Render(winner.Version),
		lipgloss.NewStyle().Foreground(jfrogGreen).Render(formatDuration(winner.AverageTime)))

	if len(results) > 1 {
		slowest := results[len(results)-1]
		speedDiff := float64(slowest.AverageTime) / float64(fastest)
		content += fmt.Sprintf("🐌 Slowest Version: %s (%s average, %.1fx slower)\n",
			lipgloss.NewStyle().Foreground(jfrogBlue).Bold(true).Render(slowest.Version),
			lipgloss.NewStyle().Foreground(jfrogBlue).Render(formatDuration(slowest.AverageTime)),
			speedDiff)
	}

	// Overall stats
	totalTime := time.Duration(0)
	totalIterations := 0
	for _, result := range results {
		totalTime += result.TotalTime
		totalIterations += result.Iterations
	}

	content += fmt.Sprintf("\n%s\n",
		lipgloss.NewStyle().Foreground(mutedGray).Render(
			fmt.Sprintf("📊 Total: %d versions tested • %d total iterations • %s combined time",
				len(results), totalIterations, formatDuration(totalTime))))

	return content
}

func displayDetailedBenchmarkLogs(results []BenchmarkResult, jfrogGreen, jfrogOrange, jfrogBlue lipgloss.Color) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(jfrogBlue).
		Padding(0, 1).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("📝 DETAILED EXECUTION LOGS"))

	for _, result := range results {
		versionHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(jfrogBlue).
			Render(fmt.Sprintf("\n🔸 %s:", result.Version))

		fmt.Println(versionHeader)

		for i, exec := range result.Executions {
			status := "✅"
			statusColor := jfrogGreen
			if exec.ExitCode != 0 {
				status = "❌"
				statusColor = jfrogOrange
			}

			line := fmt.Sprintf("  #%d: %s %s",
				i+1,
				lipgloss.NewStyle().Foreground(statusColor).Render(status),
				formatDuration(exec.Duration))

			if exec.ExitCode != 0 {
				line += lipgloss.NewStyle().Foreground(jfrogOrange).Render(fmt.Sprintf(" (exit %d)", exec.ExitCode))
			}

			fmt.Println(line)
		}
	}
}

func displayBenchmarkJSON(results []BenchmarkResult) {
	fmt.Printf("{\n")
	fmt.Printf("  \"benchmark_results\": [\n")
	for i, result := range results {
		fmt.Printf("    {\n")
		fmt.Printf("      \"version\": \"%s\",\n", result.Version)
		fmt.Printf("      \"iterations\": %d,\n", result.Iterations)
		fmt.Printf("      \"total_time_ms\": %.2f,\n", float64(result.TotalTime.Nanoseconds())/1e6)
		fmt.Printf("      \"average_time_ms\": %.2f,\n", float64(result.AverageTime.Nanoseconds())/1e6)
		fmt.Printf("      \"min_time_ms\": %.2f,\n", float64(result.MinTime.Nanoseconds())/1e6)
		fmt.Printf("      \"max_time_ms\": %.2f,\n", float64(result.MaxTime.Nanoseconds())/1e6)
		fmt.Printf("      \"success_rate\": %.2f\n", result.SuccessRate)
		if i < len(results)-1 {
			fmt.Printf("    },\n")
		} else {
			fmt.Printf("    }\n")
		}
	}
	fmt.Printf("  ]\n")
	fmt.Printf("}\n")
}

func displayBenchmarkCSV(results []BenchmarkResult) {
	fmt.Printf("version,iterations,total_time_ms,average_time_ms,min_time_ms,max_time_ms,success_rate\n")
	for _, result := range results {
		fmt.Printf("%s,%d,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			result.Version,
			result.Iterations,
			float64(result.TotalTime.Nanoseconds())/1e6,
			float64(result.AverageTime.Nanoseconds())/1e6,
			float64(result.MinTime.Nanoseconds())/1e6,
			float64(result.MaxTime.Nanoseconds())/1e6,
			result.SuccessRate)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
