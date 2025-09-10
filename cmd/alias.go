package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

type AliasData struct {
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

var Alias = &cli.Command{
	Name:  "alias",
	Usage: "Manage aliases for JFrog CLI versions",
	Subcommands: []*cli.Command{
		{
			Name:      "set",
			Usage:     "Set an alias (e.g., prod => 2.57.0) with optional description",
			ArgsUsage: "<alias> <version>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "description",
					Aliases: []string{"d"},
					Usage:   "Description to help identify the alias purpose",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 2 {
					return cli.Exit("Usage: jfvm alias set <alias> <version>", 1)
				}
				alias, version := c.Args().Get(0), c.Args().Get(1)
				description := c.String("description")

				// Prevent using "latest" as an alias since it's a reserved keyword
				if strings.ToLower(alias) == "latest" {
					return cli.Exit("'latest' is a reserved keyword and cannot be used as an alias", 1)
				}

				os.MkdirAll(utils.JfvmAliases, 0755)

				aliasData := AliasData{
					Version:     version,
					Description: description,
				}

				data, err := json.Marshal(aliasData)
				if err != nil {
					return fmt.Errorf("failed to encode alias data: %w", err)
				}

				return os.WriteFile(filepath.Join(utils.JfvmAliases, alias), data, 0644)
			},
		},
		{
			Name:      "get",
			Usage:     "Get the version and description mapped to an alias",
			ArgsUsage: "<alias>",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-color",
					Usage: "Disable colored output",
					Value: false,
				},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.Exit("Usage: jfvm alias get <alias>", 1)
				}

				aliasName := c.Args().Get(0)
				aliasData, err := getAliasData(aliasName)
				if err != nil {
					return fmt.Errorf("alias '%s' not found", aliasName)
				}

				if c.Bool("no-color") {
					fmt.Printf("Version: %s\n", aliasData.Version)
					if aliasData.Description != "" {
						fmt.Printf("Description: %s\n", aliasData.Description)
					}
				} else {
					aliasStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0052CC"))
					versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
					descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true)

					fmt.Printf("%s → %s\n",
						aliasStyle.Render(aliasName),
						versionStyle.Render(aliasData.Version))

					if aliasData.Description != "" {
						fmt.Printf("  %s\n", descStyle.Render(aliasData.Description))
					}
				}

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
	aliases := make(map[string]AliasData)
	for _, entry := range entries {
		if !entry.IsDir() {
			aliasName := entry.Name()
			// Read the version and description(if provided) from the alias file
			aliasData, err := getAliasData(aliasName)
			if err == nil {
				aliases[aliasName] = *aliasData
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

		descriptionStyle = lipgloss.NewStyle().
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
		aliasData := aliases[aliasName]

		// Build card content
		header := aliasStyle.Render(aliasName)

		var content string
		if aliasData.Description != "" {
			desc := aliasData.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			content = fmt.Sprintf("%s\n%s\n\n%s %s",
				header,
				descriptionStyle.Render(desc),
				arrowStyle.Render("🔗"),
				versionStyle.Render(aliasData.Version))
		} else {
			content = fmt.Sprintf("%s\n\n%s %s",
				header,
				arrowStyle.Render("🔗"),
				versionStyle.Render(aliasData.Version))
		}

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

func getAliasData(aliasName string) (*AliasData, error) {
	path := filepath.Join(utils.JfvmAliases, aliasName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var aliasData AliasData
	if err := json.Unmarshal(data, &aliasData); err == nil {
		return &aliasData, nil
	}

	version := strings.TrimSpace(string(data))
	return &AliasData{
		Version: version,
	}, nil
}
