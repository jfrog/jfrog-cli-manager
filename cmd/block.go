package cmd

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Block = &cli.Command{
	Name:        "block",
	Usage:       "Block a specific version of jf cli",
	ArgsUsage:   "<version>",
	Description: `Block a specific version of jf cli from being used`,
	Action: func(c *cli.Context) error {
		if c.Args().Len() != 1 {
			return cli.Exit("Please provide a version to block", 1)
		}

		version := c.Args().Get(0)
		fmt.Printf("Blocking version %s...\n", version)

		if err := utils.BlockVersion(version); err != nil {
			return cli.Exit(fmt.Sprintf("Failed to block version: %v", err), 1)
		}

		fmt.Printf("✅ Successfully blocked version %s\n", version)
		return nil
	},
}
