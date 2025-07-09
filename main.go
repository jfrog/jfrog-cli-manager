package main

import (
	"fmt"
	"os"

	"github.com/jfrog/jfrog-cli-vm/cmd"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := &cli.App{
		Name:  "jfvm",
		Usage: "Manage multiple versions of JFrog CLI",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print the version",
			},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("version") {
				fmt.Printf("jfvm version %s\n", Version)
				fmt.Printf("  Build Date: %s\n", BuildDate)
				fmt.Printf("  Git Commit: %s\n", GitCommit)
				os.Exit(0)
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
			cmd.HealthCheck,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error running jfvm CLI: %v\n", err)
		os.Exit(1)
	}
}
