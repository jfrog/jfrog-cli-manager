package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

type HistoryEntry struct {
	ID        int       `json:"id"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command,omitempty"`
	Duration  int64     `json:"duration_ms,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
	Stdout    string    `json:"stdout,omitempty"`
	Stderr    string    `json:"stderr,omitempty"`
}

type VersionStats struct {
	Version   string
	Count     int
	FirstUsed time.Time
	LastUsed  time.Time
	TotalTime time.Duration
	Commands  map[string]int
}

type commandStat struct {
	command string
	count   int
}

var History = &cli.Command{
	Name:        "history",
	Usage:       descriptions.History.Usage,
	Description: descriptions.History.Format(),
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Limit number of entries to show",
			Value: 50,
		},
		&cli.BoolFlag{
			Name:  "stats",
			Usage: "Show aggregated statistics",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "version",
			Usage: "Filter by specific version",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "format",
			Usage: "Output format: table, json",
			Value: "table",
		},
		&cli.BoolFlag{
			Name:  "clear",
			Usage: "Clear history (cannot be undone)",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "show-output",
			Usage: "Show command output in history entries",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "command",
			Usage: "Filter by command pattern (case-insensitive)",
		},
		&cli.BoolFlag{
			Name:  "failures-only",
			Usage: "Show only failed commands (exit code != 0)",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "disable-recording",
			Usage: "Disable history recording (set jfcm_NO_HISTORY=1 for permanent disable)",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("clear") {
			return clearHistory()
		}

		// Handle execute by ID using !{id} syntax
		if c.Args().Len() > 0 {
			arg := c.Args().Get(0)
			if strings.HasPrefix(arg, "!") {
				idStr := strings.TrimPrefix(arg, "!")
				if id, err := strconv.Atoi(idStr); err == nil && id > 0 {
					return executeHistoryEntry(id)
				}
			}
		}

		historyFile := filepath.Join(utils.jfcmRoot, "history.json")

		entries, err := loadHistory(historyFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load history: %w", err)
		}

		// Filter by version if specified
		if version := c.String("version"); version != "" {
			filtered := []HistoryEntry{}
			for _, entry := range entries {
				if entry.Version == version {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		// Filter by command pattern if specified
		if cmdPattern := c.String("command"); cmdPattern != "" {
			filtered := []HistoryEntry{}
			for _, entry := range entries {
				if strings.Contains(strings.ToLower(entry.Command), strings.ToLower(cmdPattern)) {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		// Filter failures only if specified
		if c.Bool("failures-only") {
			filtered := []HistoryEntry{}
			for _, entry := range entries {
				if entry.ExitCode != 0 {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		if len(entries) == 0 {
			fmt.Println("📭 No history entries found.")
			return nil
		}

		if c.Bool("stats") {
			displayHistoryStats(entries, c.Bool("no-color"))
		} else {
			displayHistory(entries, c.Int("limit"), c.String("format"), c.Bool("no-color"), c.Bool("show-output"))
		}

		return nil
	},
}

func loadHistory(historyFile string) ([]HistoryEntry, error) {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, err
	}

	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func saveHistory(historyFile string, entries []HistoryEntry) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(historyFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyFile, data, 0644)
}

func AddHistoryEntry(version, command string, duration time.Duration, exitCode int, stdout, stderr string) {
	// Skip recording jfcm commands - only record actual jf commands
	if strings.HasPrefix(command, "jfcm ") {
		return
	}

	historyFile := filepath.Join(utils.jfcmRoot, "history.json")

	entries, err := loadHistory(historyFile)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	// Truncate output to prevent huge history files
	const maxOutputSize = 5000
	if len(stdout) > maxOutputSize {
		stdout = stdout[:maxOutputSize] + "\n... (truncated)"
	}
	if len(stderr) > maxOutputSize {
		stderr = stderr[:maxOutputSize] + "\n... (truncated)"
	}

	entry := HistoryEntry{
		Version:   version,
		Timestamp: time.Now(),
		Command:   command,
		Duration:  duration.Milliseconds(),
		ExitCode:  exitCode,
		Stdout:    stdout,
		Stderr:    stderr,
	}

	// Assign the next available ID
	nextID := 1
	if len(entries) > 0 {
		nextID = entries[len(entries)-1].ID + 1
	}
	entry.ID = nextID

	entries = append(entries, entry)

	// Keep only last 1000 entries to prevent unlimited growth
	if len(entries) > 1000 {
		entries = entries[len(entries)-1000:]
		// Reassign IDs after truncation to maintain sequential order
		for i := range entries {
			entries[i].ID = i + 1
		}
	}

	saveHistory(historyFile, entries)
}

func displayHistory(entries []HistoryEntry, limit int, format string, noColor, showOutput bool) {
	if noColor {
		color.NoColor = true
	}

	// Sort by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	switch format {
	case "json":
		displayHistoryJSON(entries)
	default:
		displayHistoryTable(entries, showOutput)
	}
}

// formatDuration returns a human-friendly duration string
func formatDurationMs(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// formatDurationHMS returns a human-friendly duration string in hours, minutes, seconds format
func formatDurationHMS(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		if hours > 24 {
			days := hours / 24
			hours = hours % 24
			return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
		}
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
}

func displayHistoryTable(entries []HistoryEntry, showOutput bool) {
	var (
		greenColor = color.New(color.FgGreen)
		redColor   = color.New(color.FgRed)
		boldColor  = color.New(color.Bold)
	)

	fmt.Printf("📊 jfcm USAGE HISTORY\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════════\n\n")

	// Create table with basic configuration
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "TIME", "VERSION", "DURATION", "EXIT", "COMMAND")

	for i, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		duration := formatDurationMs(entry.Duration)
		command := entry.Command
		if !showOutput && len(command) > 50 {
			command = command[:47] + "..."
		}

		exitCodeText := "0"
		if entry.ExitCode != 0 {
			exitCodeText = fmt.Sprintf("%d", entry.ExitCode)
		}

		// Apply colors
		coloredTimestamp := timestamp
		coloredVersion := entry.Version
		coloredExitCode := exitCodeText

		if i == 0 {
			coloredTimestamp = boldColor.Add(color.FgBlue).Sprint(timestamp)
			coloredVersion = boldColor.Add(color.FgBlue).Sprint(entry.Version)
		} else {
			coloredVersion = greenColor.Sprint(entry.Version)
		}

		if entry.ExitCode != 0 {
			coloredExitCode = redColor.Sprint(exitCodeText)
		}

		table.Append(fmt.Sprintf("%d", entry.ID), coloredTimestamp, coloredVersion, duration, coloredExitCode, command)

		// Print output if requested
		if showOutput && (entry.Stdout != "" || entry.Stderr != "") {
			table.Append("", "", "", "", "", "") // Empty row for spacing
			if entry.Stdout != "" {
				table.Append("", "", "", "", "", "📤 STDOUT: "+entry.Stdout)
			}
			if entry.Stderr != "" {
				table.Append("", "", "", "", "", "📥 STDERR: "+redColor.Sprint(entry.Stderr))
			}
		}
	}

	table.Render()
	fmt.Printf("\n📈 Total entries: %d\n", len(entries))
}

func displayHistoryJSON(entries []HistoryEntry) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func displayHistoryStats(entries []HistoryEntry, noColor bool) {
	if noColor {
		color.NoColor = true
	}

	// Aggregate statistics (same logic as before)
	stats := make(map[string]*VersionStats)
	totalCommands := make(map[string]int)

	for _, entry := range entries {
		if stats[entry.Version] == nil {
			stats[entry.Version] = &VersionStats{
				Version:   entry.Version,
				FirstUsed: entry.Timestamp,
				LastUsed:  entry.Timestamp,
				Commands:  make(map[string]int),
			}
		}

		s := stats[entry.Version]
		s.Count++
		s.TotalTime += time.Duration(entry.Duration) * time.Millisecond

		if entry.Timestamp.Before(s.FirstUsed) {
			s.FirstUsed = entry.Timestamp
		}
		if entry.Timestamp.After(s.LastUsed) {
			s.LastUsed = entry.Timestamp
		}

		if entry.Command != "" {
			s.Commands[entry.Command]++
			totalCommands[entry.Command]++
		}
	}

	// Display enhanced stats using Charm libraries
	displayEnhancedStats(stats, totalCommands, entries, noColor)
}

func displayEnhancedStats(stats map[string]*VersionStats, totalCommands map[string]int, entries []HistoryEntry, noColor bool) {
	// JFrog brand colors
	var (
		jfrogGreen  = lipgloss.Color("#43C74A")
		jfrogOrange = lipgloss.Color("#FF6B35")
		jfrogBlue   = lipgloss.Color("#0052CC")
		mutedGray   = lipgloss.Color("#6B7280")
	)

	// Define beautiful styles using JFrog colors
	var (
		// Header styles
		titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogGreen).
				Padding(0, 1).
				MarginBottom(1)

		headerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogGreen).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogGreen).
				Padding(1, 2).
				MarginBottom(1)

		// Box styles
		boxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogGreen).
				Padding(1, 2).
				MarginBottom(1).
				MarginRight(2)

		// Color palette for content
		primaryColor   = jfrogGreen
		secondaryColor = jfrogBlue
		accentColor    = jfrogOrange
		mutedColor     = mutedGray
	)

	// Disable colors if requested
	if noColor {
		titleStyle = titleStyle.Foreground(lipgloss.Color(""))
		headerStyle = headerStyle.Foreground(lipgloss.Color("")).BorderForeground(lipgloss.Color(""))
		boxStyle = boxStyle.BorderForeground(lipgloss.Color(""))
		primaryColor = lipgloss.Color("")
		secondaryColor = lipgloss.Color("")
		accentColor = lipgloss.Color("")
		mutedColor = lipgloss.Color("")
	}

	// Main title
	fmt.Println(titleStyle.Render("📊 jfcm USAGE STATISTICS"))

	// Create layout sections
	sections := []string{}

	// 1. Version Usage Section (text-based)
	versionSection := createVersionUsageSection(stats, boxStyle, primaryColor, secondaryColor, accentColor)
	sections = append(sections, versionSection)

	// 2. Enhanced Version Chart Section (separate beautiful chart)
	if len(stats) > 0 {
		chartSection := createVersionChartSection(stats, boxStyle, primaryColor, secondaryColor, accentColor)
		sections = append(sections, chartSection)
	}

	// 3. Command Frequency Section
	commandSection := createCommandFrequencySection(totalCommands, boxStyle, primaryColor, secondaryColor, mutedColor)
	sections = append(sections, commandSection)

	// 4. Timeline Section
	timelineSection := createTimelineSection(entries, boxStyle, primaryColor, accentColor, mutedColor)
	sections = append(sections, timelineSection)

	// Display sections in a beautiful layout
	if len(sections) >= 3 {
		// Top row: Version Usage + Enhanced Chart
		topLayout := lipgloss.JoinHorizontal(lipgloss.Top, sections[0], sections[1])
		fmt.Println(topLayout)

		// Second row: Command Frequency (full width)
		fmt.Println(sections[2])

		// Third row: Timeline (full width)
		if len(sections) > 3 {
			fmt.Println(sections[3])
		}
	} else {
		// Fall back to vertical layout
		for _, section := range sections {
			fmt.Println(section)
		}
	}
}

func createVersionUsageSection(stats map[string]*VersionStats, boxStyle lipgloss.Style, primaryColor, secondaryColor, accentColor lipgloss.Color) string {
	// Sort versions by usage count
	versions := make([]*VersionStats, 0, len(stats))
	for _, stat := range stats {
		versions = append(versions, stat)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Count > versions[j].Count
	})

	// Build clean version usage display without embedded charts
	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("🔢 VERSION USAGE") + "\n\n"

	maxCount := 0
	if len(versions) > 0 {
		maxCount = versions[0].Count
	}

	for i, stat := range versions {
		// Color based on ranking
		var color lipgloss.Color = secondaryColor
		if i == 0 {
			color = primaryColor
		} else if i == len(versions)-1 && len(versions) > 1 {
			color = accentColor
		}

		// Create progress bar
		barWidth := 20
		progress := float64(stat.Count) / float64(maxCount)
		filled := int(progress * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		versionLine := fmt.Sprintf("%-12s %s %s (%d uses)\n",
			lipgloss.NewStyle().Foreground(color).Bold(true).Render(stat.Version),
			lipgloss.NewStyle().Foreground(color).Render(bar),
			lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%3.0f%%", progress*100)),
			stat.Count)

		content += versionLine

		// Add timing info
		timingInfo := fmt.Sprintf("             ⏱️  %s total • 📅 %s to %s\n",
			formatDurationHMS(stat.TotalTime),
			stat.FirstUsed.Format("Jan 02"),
			stat.LastUsed.Format("Jan 02"))
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(timingInfo)
	}

	return boxStyle.Width(50).Render(content)
}

func createVersionChartSection(stats map[string]*VersionStats, boxStyle lipgloss.Style, primaryColor, secondaryColor, accentColor lipgloss.Color) string {
	if len(stats) == 0 {
		return boxStyle.Width(50).Render(lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📊 VERSION CHART") + "\n\nNo data available for chart")
	}

	// Prepare data for bar chart - limit to top 5 for readability
	versions := make([]*VersionStats, 0, len(stats))
	for _, stat := range stats {
		versions = append(versions, stat)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Count > versions[j].Count
	})

	maxShow := 5
	if len(versions) < maxShow {
		maxShow = len(versions)
	}

	// Create beautiful bar chart with ntcharts
	bc := barchart.New(42, 10)

	for i := 0; i < maxShow; i++ {
		version := versions[i]

		// Choose color based on ranking using JFrog theme
		var chartColor lipgloss.Style
		if i == 0 {
			chartColor = lipgloss.NewStyle().Foreground(primaryColor) // Top version in JFrog green
		} else if i == maxShow-1 {
			chartColor = lipgloss.NewStyle().Foreground(accentColor) // Last in JFrog orange
		} else {
			chartColor = lipgloss.NewStyle().Foreground(secondaryColor) // Others in JFrog blue
		}

		// Create bar data with proper labeling
		barData := barchart.BarData{
			Label: fmt.Sprintf("v%s", version.Version),
			Values: []barchart.BarValue{
				{
					Name:  fmt.Sprintf("%d uses", version.Count),
					Value: float64(version.Count),
					Style: chartColor,
				},
			},
		}
		bc.Push(barData)
	}

	bc.Draw()
	chartView := bc.View()

	// Create beautiful content with title and chart
	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📊 VERSION POPULARITY CHART") + "\n\n"
	content += chartView + "\n\n"

	// Add a legend
	legend := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render("Top " + fmt.Sprintf("%d", maxShow) + " most used versions")
	content += legend

	return boxStyle.Width(50).Render(content)
}

func createCommandFrequencySection(totalCommands map[string]int, boxStyle lipgloss.Style, primaryColor, secondaryColor, mutedColor lipgloss.Color) string {
	if len(totalCommands) == 0 {
		return boxStyle.Width(60).Render(lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("🚀 MOST COMMON COMMANDS") + "\n\nNo commands recorded")
	}

	// Sort commands by frequency
	commands := make([]commandStat, 0, len(totalCommands))
	for cmd, count := range totalCommands {
		commands = append(commands, commandStat{cmd, count})
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].count > commands[j].count
	})

	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("🚀 MOST COMMON COMMANDS") + "\n\n"

	// Add clean sparkline visualization for top commands (single line per command)
	if len(commands) > 0 {
		sparklineSection := createCleanCommandSparklines(commands, primaryColor, secondaryColor)
		content += sparklineSection + "\n"
	}

	// Add detailed frequency list
	content += lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📋 DETAILED USAGE") + "\n\n"

	maxShow := 8
	if len(commands) < maxShow {
		maxShow = len(commands)
	}

	maxCount := commands[0].count

	for i, cmd := range commands[:maxShow] {
		// Color based on ranking
		var color lipgloss.Color = secondaryColor
		if i == 0 {
			color = primaryColor
		}

		// Truncate long commands
		displayCmd := cmd.command
		if len(displayCmd) > 35 {
			displayCmd = displayCmd[:32] + "..."
		}

		// Create mini progress bar
		barWidth := 10
		progress := float64(cmd.count) / float64(maxCount)
		filled := int(progress * float64(barWidth))
		bar := strings.Repeat("▓", filled) + strings.Repeat("░", barWidth-filled)

		line := fmt.Sprintf("%-38s %s %s\n",
			displayCmd,
			lipgloss.NewStyle().Foreground(color).Render(bar),
			lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("×%d", cmd.count)))

		content += line
	}

	return boxStyle.Width(60).Render(content)
}

// createCleanCommandSparklines creates clean, single-line sparkline visualizations
func createCleanCommandSparklines(commands []commandStat, primaryColor, secondaryColor lipgloss.Color) string {
	if len(commands) == 0 {
		return ""
	}

	result := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📊 COMMAND TRENDS") + "\n\n"

	// Show clean sparklines for top 5 commands
	maxShow := 5
	if len(commands) < maxShow {
		maxShow = len(commands)
	}

	for i := 0; i < maxShow; i++ {
		cmd := commands[i]

		// Create simple trend data
		trendData := generateTrendData(cmd.count)

		// Create a simple sparkline string manually (more control over appearance)
		sparkline := createSimpleSparkline(trendData)

		// Choose color
		color := secondaryColor
		if i == 0 {
			color = primaryColor
		}

		// Truncate command name for display
		displayCmd := cmd.command
		if len(displayCmd) > 20 {
			displayCmd = displayCmd[:17] + "..."
		}

		// Create clean single-line output
		line := fmt.Sprintf("%-22s %s (%d uses)\n",
			lipgloss.NewStyle().Bold(true).Foreground(color).Render(displayCmd),
			lipgloss.NewStyle().Foreground(color).Render(sparkline),
			cmd.count)

		result += line
	}

	return result
}

// createSimpleSparkline creates a clean, single-line sparkline
func createSimpleSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	// Find min and max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Create sparkline using block characters
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	sparkline := ""

	for _, v := range data {
		var level int
		if max == min {
			level = 3 // Middle level if all values are the same
		} else {
			level = int((v - min) / (max - min) * 7) // 0-7 range
		}
		sparkline += string(sparkChars[level])
	}

	return sparkline
}

// generateTrendData creates synthetic trend data for demonstration
// In a real implementation, this would analyze historical command usage patterns
func generateTrendData(count int) []float64 {
	// Create a synthetic trend based on command count
	data := make([]float64, 10)
	base := float64(count) / 10.0

	for i := range data {
		// Create some variation around the base
		variation := float64(i%3) * 0.3
		if i > 5 {
			variation += 0.5 // Simulate recent increase
		}
		data[i] = base + variation
	}

	return data
}

func createTimelineSection(entries []HistoryEntry, boxStyle lipgloss.Style, primaryColor, accentColor, mutedColor lipgloss.Color) string {
	if len(entries) == 0 {
		return boxStyle.Width(110).Render("📅 USAGE TIMELINE\n\nNo entries available")
	}

	// Calculate timeline stats
	oldest := entries[0].Timestamp
	newest := entries[0].Timestamp
	for _, entry := range entries {
		if entry.Timestamp.Before(oldest) {
			oldest = entry.Timestamp
		}
		if entry.Timestamp.After(newest) {
			newest = entry.Timestamp
		}
	}

	duration := newest.Sub(oldest)
	avgPerDay := float64(len(entries)) / (duration.Hours() / 24)

	// Build content
	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("📅 USAGE TIMELINE") + "\n\n"

	// Timeline stats in a nice format
	stats := []struct {
		label string
		value string
		color lipgloss.Color
	}{
		{"📅 First Usage", oldest.Format("2006-01-02 15:04:05"), primaryColor},
		{"🕒 Latest Usage", newest.Format("2006-01-02 15:04:05"), primaryColor},
		{"⏳ Total Period", formatDurationHMS(duration), accentColor},
		{"📊 Total Entries", fmt.Sprintf("%d", len(entries)), primaryColor},
	}

	if duration.Hours() > 24 {
		stats = append(stats, struct {
			label string
			value string
			color lipgloss.Color
		}{"📈 Avg Per Day", fmt.Sprintf("%.1f", avgPerDay), accentColor})
	}

	// Create two columns of stats
	leftColumn := ""
	rightColumn := ""

	for i, stat := range stats {
		line := fmt.Sprintf("%-15s %s\n",
			stat.label+":",
			lipgloss.NewStyle().Foreground(stat.color).Bold(true).Render(stat.value))

		if i%2 == 0 {
			leftColumn += line
		} else {
			rightColumn += line
		}
	}

	// Join columns side by side
	timelineContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
	content += timelineContent

	return boxStyle.Width(110).Render(content)
}

func executeHistoryEntry(id int) error {
	historyFile := filepath.Join(utils.jfcmRoot, "history.json")
	entries, err := loadHistory(historyFile)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Find the entry with the specified ID
	var targetEntry *HistoryEntry
	for _, entry := range entries {
		if entry.ID == id {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		return fmt.Errorf("history entry with ID %d not found", id)
	}

	fmt.Printf("🔄 Executing history entry #%d: %s\n", id, targetEntry.Command)
	fmt.Printf("📋 Version: %s\n", targetEntry.Version)

	// First, switch to the required version
	if err := utils.SwitchToVersion(targetEntry.Version); err != nil {
		return fmt.Errorf("failed to switch to version %s: %w", targetEntry.Version, err)
	}

	// Parse the command to extract the actual jf command (remove "jf " prefix)
	command := targetEntry.Command
	if strings.HasPrefix(command, "jf ") {
		command = strings.TrimPrefix(command, "jf ")
	}

	// Execute the command
	cmd := exec.Command("jf", strings.Fields(command)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func clearHistory() error {
	historyFile := filepath.Join(utils.jfcmRoot, "history.json")

	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		fmt.Println("📭 No history file found.")
		return nil
	}

	if err := os.Remove(historyFile); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Println("🗑️  History cleared successfully.")
	return nil
}
