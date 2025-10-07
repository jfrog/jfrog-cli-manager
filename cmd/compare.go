package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

// Default constants for changelog comparison source
const (
	DefaultChangelogOwner = "jfrog"
	DefaultChangelogRepo  = "jfrog-cli"
)

// VersionConfig holds version information after validation and resolution
type VersionConfig struct {
	Version1  string
	Version2  string
	Resolved1 string
	Resolved2 string
}

// validateAndResolveVersions validates and resolves version arguments
func validateAndResolveVersions(args []string, minArgs int) (*VersionConfig, error) {
	if len(args) < minArgs {
		return nil, fmt.Errorf("requires at least %d arguments", minArgs)
	}

	config := &VersionConfig{
		Version1: args[0],
		Version2: args[1],
	}

	// Resolve aliases if needed
	resolved1, err := utils.ResolveVersionOrAlias(config.Version1)
	if err != nil {
		resolved1 = config.Version1
	}
	config.Resolved1 = resolved1

	resolved2, err := utils.ResolveVersionOrAlias(config.Version2)
	if err != nil {
		resolved2 = config.Version2
	}
	config.Resolved2 = resolved2

	return config, nil
}

var Compare = &cli.Command{
	Name:        "compare",
	Usage:       descriptions.Compare.Usage,
	Description: descriptions.Compare.Format(),
	Subcommands: []*cli.Command{
		CompareChangelog,
		CompareCli,
		CompareRt,
	},
}

var CompareChangelog = &cli.Command{
	Name:      "changelog",
	Usage:     "Compare release notes between two versions",
	ArgsUsage: "<version1> <version2>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Command timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:  "timing",
			Usage: "Show execution timing information",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		args := c.Args().Slice()

		// Validate and resolve versions
		config, err := validateAndResolveVersions(args, 2)
		if err != nil {
			return cli.Exit("Usage: jfcm compare changelog <version1> <version2>", 1)
		}

		// Handle changelog comparison
		return handleChangelogComparison(c, config.Version1, config.Version2, config.Resolved1, config.Resolved2)
	},
}

var CompareCli = &cli.Command{
	Name:      "cli",
	Usage:     "Compare JFrog CLI command execution between two versions",
	ArgsUsage: "<version1> <version2> -- <jf-command> [args...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "unified",
			Usage: "Show unified diff format instead of side-by-side",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Command timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:  "timing",
			Usage: "Show execution timing information",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		args := c.Args().Slice()

		// Validate and resolve versions
		config, err := validateAndResolveVersions(args, 3)
		if err != nil {
			return cli.Exit("Usage: jfcm compare cli <version1> <version2> -- <jf-command> [args...]", 1)
		}

		// Validate CLI-specific arguments
		jfCommand, err := validateCLIArguments(args)
		if err != nil {
			return cli.Exit("Missing '--' separator. Usage: jfcm compare cli <version1> <version2> -- <jf-command> [args...]", 1)
		}

		// Check if versions exist
		if err := utils.CheckVersionExists(config.Resolved1); err != nil {
			return fmt.Errorf("version %s (%s) not found: %w", config.Version1, config.Resolved1, err)
		}
		if err := utils.CheckVersionExists(config.Resolved2); err != nil {
			return fmt.Errorf("version %s (%s) not found: %w", config.Version2, config.Resolved2, err)
		}

		fmt.Printf("🔄 Comparing JFrog CLI versions: %s vs %s\n", config.Version1, config.Version2)
		fmt.Printf("📝 Command: jf %s\n\n", strings.Join(jfCommand, " "))

		// Execute commands in parallel
		results := make([]ExecutionResult, 2)
		g, ctx := errgroup.WithContext(context.Background())

		timeout := time.Duration(c.Int("timeout")) * time.Second
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		g.Go(func() error {
			result, err := executeJFCommand(timeoutCtx, config.Resolved1, jfCommand)
			results[0] = result
			return err
		})

		g.Go(func() error {
			result, err := executeJFCommand(timeoutCtx, config.Resolved2, jfCommand)
			results[1] = result
			return err
		})

		if err := g.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: %v\n\n", err)
		}

		// Display results
		displayComparison(results[0], results[1], c.Bool("unified"), c.Bool("no-color"), c.Bool("timing"))

		return nil
	},
}

var CompareRt = &cli.Command{
	Name:      "rt",
	Usage:     "Compare JFrog CLI command execution between two servers",
	ArgsUsage: "<server1> <server2> -- <jf-command> [args...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "unified",
			Usage: "Show unified diff format instead of side-by-side",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Command timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:  "timing",
			Usage: "Show execution timing information",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		args := c.Args().Slice()

		// Validate RT-specific arguments
		server1, server2, jfCommand, err := validateRTArguments(args)
		if err != nil {
			return cli.Exit("Usage: jfcm compare rt <server1> <server2> -- <jf-command> [args...]", 1)
		}

		fmt.Printf("🔄 Comparing JFrog CLI command across servers: %s vs %s\n", server1, server2)
		fmt.Printf("📝 Command: jf %s\n\n", strings.Join(jfCommand, " "))

		// Execute commands against both servers in parallel
		results := make([]ExecutionResult, 2)
		g, ctx := errgroup.WithContext(context.Background())

		timeout := time.Duration(c.Int("timeout")) * time.Second
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		g.Go(func() error {
			result, err := executeJFCommandOnServer(timeoutCtx, server1, jfCommand)
			results[0] = result
			return err
		})

		g.Go(func() error {
			result, err := executeJFCommandOnServer(timeoutCtx, server2, jfCommand)
			results[1] = result
			return err
		})

		if err := g.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: %v\n\n", err)
		}

		// Display results
		displayComparison(results[0], results[1], c.Bool("unified"), c.Bool("no-color"), c.Bool("timing"))

		return nil
	},
}

func handleChangelogComparison(c *cli.Context, version1, version2, resolved1, resolved2 string) error {
	fmt.Printf("📖 Comparing Release Notes: %s vs %s\n", version1, version2)
	fmt.Printf("🔍 Fetching changelog between versions...\n\n")

	// Create context with timeout
	timeout := time.Duration(c.Int("timeout")) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()

	// Call FetchTopReleasesNotes() to get changelog data
	owner := DefaultChangelogOwner
	repo := DefaultChangelogRepo

	// Ensure tags have "v" prefix for GitHub API
	fromTag := resolved1
	if !strings.HasPrefix(fromTag, "v") {
		fromTag = "v" + fromTag
	}
	toTag := resolved2
	if !strings.HasPrefix(toTag, "v") {
		toTag = "v" + toTag
	}

	releaseNotes, err := FetchTopReleasesNotes(ctx, owner, repo, fromTag, toTag)
	fetchDuration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("failed to fetch release notes: %w", err)
	}

	// Filter release notes to remove unwanted sections
	for i := range releaseNotes {
		releaseNotes[i].Body = FilterReleaseNotes(releaseNotes[i].Body)
	}

	// Display the changelog results using the moved display function
	DisplayChangelogResults(releaseNotes, version1, version2, fetchDuration, c.Bool("no-color"), c.Bool("timing"))

	return nil
}
