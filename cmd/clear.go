package cmd

import (
	"fmt"
	"os"

	"github.com/jfrog/jfrog-cli-manager/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Clear = &cli.Command{
	Name:  "clear",
	Usage: "Remove all installed JFrog CLI versions",
	Action: func(c *cli.Context) error {
		err := os.RemoveAll(utils.JFCMVersions)
		if err != nil {
			return fmt.Errorf("failed to clear versions: %w", err)
		}
		fmt.Println("All versions removed.")
		return nil
	},
}
