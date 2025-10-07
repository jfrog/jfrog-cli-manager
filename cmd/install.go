package cmd

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-manager/internal"
	"github.com/urfave/cli/v2"
)

var Install = &cli.Command{
	Name:      "install",
	Usage:     "Install a specific version of JFrog CLI",
	ArgsUsage: "[version]",
	Action: func(c *cli.Context) error {
		if c.Args().Len() != 1 {
			return cli.Exit("Please provide a version (e.g., 2.57.0)", 1)
		}
		version := c.Args().Get(0)
		fmt.Printf("Installing JFrog CLI version: %s\n", version)
		return internal.DownloadAndInstall(version)
	},
}
