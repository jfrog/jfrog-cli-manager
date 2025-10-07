package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var List = &cli.Command{
	Name:  "list",
	Usage: "List all installed JFrog CLI versions",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "simple",
			Usage: "Show simple text list instead of enhanced display",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		if c.Bool("simple") {
			return displaySimpleList()
		}
		return displayEnhancedList(c.Bool("no-color"))
	},
}

func displaySimpleList() error {
	currentData, _ := os.ReadFile(utils.jfcmConfig)
	current := string(currentData)

	entries, err := os.ReadDir(utils.jfcmVersions)
	if err != nil {
		return err
	}

	fmt.Println("Installed versions:")
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			mark := ""
			if version == current {
				mark = " (current)"
			}
			fmt.Printf(" - %s%s\n", version, mark)
		}
	}
	return nil
}

func displayEnhancedList(noColor bool) error {
	currentData, _ := os.ReadFile(utils.jfcmConfig)
	current := string(currentData)

	entries, err := os.ReadDir(utils.jfcmVersions)
	if err != nil {
		return err
	}

	// JFrog brand colors
	var (
		jfrogGreen = lipgloss.Color("#43C74A")
		jfrogBlue  = lipgloss.Color("#0052CC")
		mutedGray  = lipgloss.Color("#6B7280")
	)

	// Define beautiful styles using JFrog colors
	var (
		titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogGreen).
				Padding(0, 1).
				MarginBottom(1)

		currentCardStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(jfrogGreen).
					Padding(1, 2).
					MarginBottom(1).
					MarginRight(2)

		regularCardStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(jfrogBlue).
					Padding(1, 2).
					MarginBottom(1).
					MarginRight(2)

		versionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#E5E7EB"))

		currentBadgeStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(jfrogGreen).
					Padding(0, 1).
					MarginLeft(1)

		metaStyle = lipgloss.NewStyle().
				Foreground(mutedGray).
				Italic(true)
	)

	// Handle no-color mode
	if noColor {
		titleStyle = titleStyle.Foreground(lipgloss.Color(""))
		currentCardStyle = currentCardStyle.BorderForeground(lipgloss.Color(""))
		regularCardStyle = regularCardStyle.BorderForeground(lipgloss.Color(""))
		versionStyle = versionStyle.Foreground(lipgloss.Color(""))
		currentBadgeStyle = currentBadgeStyle.Foreground(lipgloss.Color("")).Background(lipgloss.Color(""))
		metaStyle = metaStyle.Foreground(lipgloss.Color(""))
	}

	// Display title
	fmt.Println(titleStyle.Render("📦 INSTALLED JFROG CLI VERSIONS"))

	// Collect version info
	type VersionInfo struct {
		Name       string
		IsCurrent  bool
		Size       string
		ModTime    time.Time
		BinaryPath string
	}

	var versions []VersionInfo

	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			versionPath := filepath.Join(utils.jfcmVersions, version)

			info := VersionInfo{
				Name:       version,
				IsCurrent:  version == current,
				BinaryPath: filepath.Join(versionPath, utils.BinaryName),
			}

			// Get modification time and size
			if stat, err := entry.Info(); err == nil {
				info.ModTime = stat.ModTime()
			}

			// Get binary size if exists
			if binStat, err := os.Stat(info.BinaryPath); err == nil {
				info.Size = formatFileSize(binStat.Size())
			} else {
				info.Size = "N/A"
			}

			versions = append(versions, info)
		}
	}

	// Sort versions: current first, then by name
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].IsCurrent {
			return true
		}
		if versions[j].IsCurrent {
			return false
		}
		return versions[i].Name < versions[j].Name
	})

	// Create cards layout
	var cards []string
	cardsPerRow := 3

	for i, version := range versions {
		cardStyle := regularCardStyle
		if version.IsCurrent {
			cardStyle = currentCardStyle
		}

		// Build card content
		header := versionStyle.Render(version.Name)
		if version.IsCurrent {
			header += currentBadgeStyle.Render("CURRENT")
		}

		metadata := fmt.Sprintf("📅 %s\n📦 %s",
			metaStyle.Render(version.ModTime.Format("Jan 02, 2006")),
			metaStyle.Render(version.Size))

		cardContent := header + "\n\n" + metadata
		card := cardStyle.Width(25).Render(cardContent)
		cards = append(cards, card)

		// Display cards in rows
		if (i+1)%cardsPerRow == 0 || i == len(versions)-1 {
			startIdx := (i / cardsPerRow) * cardsPerRow
			endIdx := i + 1
			rowCards := cards[startIdx:endIdx]

			row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
			fmt.Println(row)
		}
	}

	// Summary with JFrog colors
	totalCount := len(versions)
	currentVersion := current
	if currentVersion == "" {
		currentVersion = "None"
	}

	summaryStyle := lipgloss.NewStyle().
		Foreground(mutedGray).
		MarginTop(1).
		Italic(true)

	if noColor {
		summaryStyle = summaryStyle.Foreground(lipgloss.Color(""))
	}

	summary := fmt.Sprintf("📊 Total: %s • 🎯 Current: %s",
		lipgloss.NewStyle().Foreground(jfrogBlue).Bold(true).Render(fmt.Sprintf("%d versions", totalCount)),
		lipgloss.NewStyle().Foreground(jfrogGreen).Bold(true).Render(currentVersion))
	fmt.Println(summaryStyle.Render(summary))

	return nil
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
