package main

import (
	"log"
	"os"

	"github.com/jfrog/jfrog-cli-vm/cmd"
	"github.com/urfave/cli/v2"
)

// Version information - these can be set during build time using ldflags
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	log.Println("Starting jfvm CLI...")
	app := &cli.App{
		Name:                 "jfvm",
		Usage:                "Manage multiple versions of JFrog CLI",
		Version:              Version,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmd.Install,
			cmd.Use,
			cmd.List,
			cmd.Remove,
			cmd.Clear,
			cmd.Alias,
			cmd.Link,
			cmd.Compare,
			cmd.Benchmark,
			cmd.History,
			cmd.VersionCmd,
			cmd.HealthCheck,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error running jfvm CLI: %v", err)
	}
}
