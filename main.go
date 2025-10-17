package main

import (
	"fmt"
	"os"

	"strings"

	"github.com/jfrog/jfrog-cli-manager/cmd"
	"github.com/jfrog/jfrog-cli-manager/cmd/utils"
	"github.com/jfrog/jfrog-cli-manager/internal"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := &cli.App{
		Name:  "jfcm",
		Usage: "Manage multiple versions of JFrog CLI",
		After: func(c *cli.Context) error {
			// Capture feature_id only if it matches a registered top-level command (or its alias)
			// 1) Build a set of valid command names and aliases
			valid := make(map[string]struct{})
			if c != nil && c.App != nil {
				for _, cmd := range c.App.Commands {
					if cmd.Name != "" {
						valid[cmd.Name] = struct{}{}
					}
					for _, al := range cmd.Aliases {
						if al != "" {
							valid[al] = struct{}{}
						}
					}
				}
			}

			// 2) Find the first non-flag arg after the binary
			candidate := ""
			for _, a := range os.Args[1:] {
				if a == "" || strings.HasPrefix(a, "-") {
					continue
				}
				candidate = a
				break
			}

			// 3) Record only if it's a known command/alias
			if candidate != "" {
				if _, ok := valid[candidate]; ok {
					internal.AppendLocalJFcmMetric(candidate)
				}
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print the version",
			},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("version") {
				fmt.Printf("jfcm version %s\n", Version)
				fmt.Printf("  Build Date: %s\n", BuildDate)
				fmt.Printf("  Git Commit: %s\n", GitCommit)
				os.Exit(0)
			}

			// Initialize jfcm directories
			if err := utils.InitializejfcmDirectories(); err != nil {
				return fmt.Errorf("failed to initialize jfcm directories: %w", err)
			}

			return nil
		},
		Commands: []*cli.Command{
			cmd.Use,
			cmd.Install,
			cmd.List,
			cmd.Remove,
			cmd.Clear,
			cmd.Alias,
			cmd.Link,
			cmd.Compare,
			cmd.Benchmark,
			cmd.History,
			cmd.AddHistoryEntryCmd,
			cmd.HealthCheck,
			cmd.Block,
			cmd.Unblock,
			cmd.ListBlocked,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error running jfcm CLI: %v\n", err)
		os.Exit(1)
	}
}
