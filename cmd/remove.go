package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Remove = &cli.Command{
	Name:      "remove",
	Usage:     "Remove an installed JFrog CLI version",
	ArgsUsage: "[version]",
	Action: func(c *cli.Context) error {
		if c.Args().Len() != 1 {
			return cli.Exit("Please provide a version to remove", 1)
		}
		version := c.Args().Get(0)
		dir := filepath.Join(utils.jfcmVersions, version)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("version %s is not installed", version)
		}

		return os.RemoveAll(dir)
	},
}
