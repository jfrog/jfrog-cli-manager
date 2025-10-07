package cmd

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-manager/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Unblock = &cli.Command{
	Name:        "unblock",
	Usage:       "Unblock a previously blocked version of jf cli",
	ArgsUsage:   "<version>",
	Description: `Unblock a specific version of jf-cli that was previously blocked.`,
	Action: func(c *cli.Context) error {
		if c.Args().Len() != 1 {
			return cli.Exit("Please provide a specific version to unblock", 1)
		}

		version := c.Args().Get(0)

		if _, err := utils.ParseVersion(version); err != nil {
			return cli.Exit(fmt.Sprintf("Invalid version format: %v", err), 1)
		}

		fmt.Printf("Unblocking version %s...\n", version)

		if err := utils.UnblockVersion(version); err != nil {
			return cli.Exit(fmt.Sprintf("Failed to unblock version: %v", err), 1)
		}

		fmt.Printf("✅ Successfully unblocked version %s\n", version)
		return nil
	},
}
