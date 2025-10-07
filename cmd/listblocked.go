package cmd

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-manager/cmd/utils"
	"github.com/urfave/cli/v2"
)

var ListBlocked = &cli.Command{
	Name:    "list-blocked",
	Usage:   "Lists all blocked versions of jf-cli",
	Aliases: []string{"lb"},
	Description: `All versions of jf-cli that are blocked.
                  These versions cannot be used until they are unblocked using 'jfcm unblock <versions>' command.`,
	Action: func(c *cli.Context) error {
		blockedVersions, err := utils.GetBlockedVersions()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to get blocked versions: %v", err), 1)
		}

		if len(blockedVersions) == 0 {
			fmt.Println("\nNo versions are currently blocked.")
			return nil
		}

		fmt.Println("blocked versions:")
		for _, version := range blockedVersions {
			fmt.Printf("  • %s\n", version)
		}

		fmt.Println("\nuse 'jfcm unblock <version>' to unblock a specific version")
		return nil
	},
}
