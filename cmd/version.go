package cmd

import (
	"fmt"
	"runtime"

	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/urfave/cli/v2"
)

// Version information - these can be set during build time using ldflags
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

var VersionCmd = &cli.Command{
	Name:        "version",
	Usage:       descriptions.Version.Usage,
	Description: descriptions.Version.Format(),
	Action: func(c *cli.Context) error {
		fmt.Printf("jfvm version %s\n", Version)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}
