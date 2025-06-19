package main

import (
	"log"
	"os"

	"github.com/jfrog/jfrog-cli-vm/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Println("Starting jfvm CLI...")
	app := &cli.App{
		Name:                 "jfvm",
		Usage:                "Manage multiple versions of JFrog CLI",
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
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error running jfvm CLI: %v", err)
	}
}
