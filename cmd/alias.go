package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Alias = &cli.Command{
	Name:  "alias",
	Usage: "Manage aliases for JFrog CLI versions",
	Subcommands: []*cli.Command{
		{
			Name:      "set",
			Usage:     "Set an alias (e.g., prod => 2.57.0)",
			ArgsUsage: "<alias> <version>",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 2 {
					return cli.Exit("Usage: jfvm alias set <alias> <version>", 1)
				}
				alias, version := c.Args().Get(0), c.Args().Get(1)

				// Prevent using "latest" as an alias since it's a reserved keyword
				if strings.ToLower(alias) == "latest" {
					return cli.Exit("'latest' is a reserved keyword and cannot be used as an alias", 1)
				}

				os.MkdirAll(utils.JfvmAliases, 0755)
				return os.WriteFile(filepath.Join(utils.JfvmAliases, alias), []byte(version), 0644)
			},
		},
		{
			Name:      "get",
			Usage:     "Get the version mapped to an alias",
			ArgsUsage: "<alias>",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.Exit("Usage: jfvm alias get <alias>", 1)
				}
				version, err := utils.ResolveAlias(c.Args().Get(0))
				if err != nil {
					return err
				}
				fmt.Println(version)
				return nil
			},
		},
		{
			Name:      "remove",
			Usage:     "Remove an alias",
			ArgsUsage: "<alias>",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.Exit("Usage: jfvm alias remove <alias>", 1)
				}
				return os.Remove(filepath.Join(utils.JfvmAliases, c.Args().Get(0)))
			},
		},
		{
			Name:  "list",
			Usage: "List all configured aliases",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-color",
					Usage: "Disable colored output",
					Value: false,
				},
			},
			Action: func(c *cli.Context) error {
				return listAliases(c.Bool("no-color"))
			},
		},
	},
}

func listAliases(noColor bool) error {
	// Check if aliases directory exists
	if _, err := os.Stat(utils.JfvmAliases); os.IsNotExist(err) {
		fmt.Println("No aliases configured yet.")
		return nil
	}

	// Read all files from aliases directory
	entries, err := os.ReadDir(utils.JfvmAliases)
	if err != nil {
		return fmt.Errorf("failed to read aliases directory: %w", err)
	}

	// Filter out directories and collect aliases
	aliases := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			aliasName := entry.Name()
			// Read the version from the alias file
			version, err := utils.ResolveAlias(aliasName)
			if err == nil {
				aliases[aliasName] = version
			}
		}
	}

	if len(aliases) == 0 {
		fmt.Println("No aliases configured yet.")
		return nil
	}

	// JFrog brand colors
	var (
		jfrogGreen = lipgloss.Color("#43C74A")
		jfrogBlue  = lipgloss.Color("#0052CC")
		mutedGray  = lipgloss.Color("#6B7280")
	)

	// Define styles
	var (
		titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogGreen).
				Padding(0, 1).
				MarginBottom(1)

		aliasStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(jfrogBlue)

		versionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB"))

		arrowStyle = lipgloss.NewStyle().
				Foreground(mutedGray)

		cardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(jfrogBlue).
				Padding(1, 2).
				MarginBottom(1).
				MarginRight(2)

		metaStyle = lipgloss.NewStyle().
				Foreground(mutedGray).
				Italic(true)
	)

	// Handle no-color mode
	if noColor {
		titleStyle = titleStyle.Foreground(lipgloss.Color(""))
		aliasStyle = aliasStyle.Foreground(lipgloss.Color(""))
		versionStyle = versionStyle.Foreground(lipgloss.Color(""))
		arrowStyle = arrowStyle.Foreground(lipgloss.Color(""))
		cardStyle = cardStyle.BorderForeground(lipgloss.Color(""))
		metaStyle = metaStyle.Foreground(lipgloss.Color(""))
	}

	// Display title
	fmt.Println(titleStyle.Render("🔗 CONFIGURED ALIASES"))

	// Sort aliases alphabetically
	sortedAliases := make([]string, 0, len(aliases))
	for alias := range aliases {
		sortedAliases = append(sortedAliases, alias)
	}
	sort.Strings(sortedAliases)

	// Create cards layout
	var cards []string
	cardsPerRow := 3

	for i, aliasName := range sortedAliases {
		version := aliases[aliasName]

		// Build card content
		header := aliasStyle.Render(aliasName)

		content := fmt.Sprintf("%s\n\n%s %s",
			header,
			arrowStyle.Render("🔗"),
			versionStyle.Render(version))

		card := cardStyle.Width(25).Render(content)
		cards = append(cards, card)

		// Display cards in rows
		if (i+1)%cardsPerRow == 0 || i == len(sortedAliases)-1 {
			startIdx := (i / cardsPerRow) * cardsPerRow
			endIdx := i + 1
			rowCards := cards[startIdx:endIdx]

			row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
			fmt.Println(row)
		}
	}

	// Summary
	summaryStyle := lipgloss.NewStyle().
		Foreground(mutedGray).
		MarginTop(1).
		Italic(true)

	if noColor {
		summaryStyle = summaryStyle.Foreground(lipgloss.Color(""))
	}

	summary := fmt.Sprintf("📊 Total: %s",
		lipgloss.NewStyle().Foreground(jfrogBlue).Bold(true).Render(fmt.Sprintf("%d aliases", len(aliases))))
	fmt.Println(summaryStyle.Render(summary))

	return nil
}
