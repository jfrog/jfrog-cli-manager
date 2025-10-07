package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfrog/jfrog-cli-vm/cmd/descriptions"
	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/jfrog/jfrog-cli-vm/internal"
	"github.com/urfave/cli/v2"
)

var Use = &cli.Command{
	Name:        "use",
	Usage:       descriptions.Use.Usage,
	ArgsUsage:   "[version or alias] (optional if .jfrog-version exists)",
	Description: descriptions.Use.Format(),
	Action: func(c *cli.Context) error {
		fmt.Println("Executing 'jfcm use' command...")
		var version string
		versionExplicitlyProvided := false

		if c.Args().Len() == 1 {
			v := c.Args().Get(0)
			fmt.Printf("Received argument: %s\n", v)
			versionExplicitlyProvided = true

			// Handle "latest" parameter
			if strings.ToLower(v) == "latest" {
				fmt.Println("Fetching latest version...")
				latestVersion, err := utils.GetLatestVersionWithFallback()
				if err != nil {
					return fmt.Errorf("failed to get latest version: %w", err)
				}
				version = latestVersion
				fmt.Printf("Latest version: %s\n", version)

				// Check if latest version is already installed
				binPath := filepath.Join(utils.JFCMVersions, version, utils.BinaryName)
				if _, err := os.Stat(binPath); os.IsNotExist(err) {
					fmt.Printf("Latest version %s not found locally. Downloading...\n", version)
					if err := internal.DownloadAndInstall(version); err != nil {
						return fmt.Errorf("failed to download latest version: %w", err)
					}
				} else {
					fmt.Printf("Latest version %s is already installed.\n", version)
				}
			} else {
				// Try to resolve alias (silently fallback if not found)
				resolved, err := utils.ResolveAlias(v)
				if err == nil {
					version = strings.TrimSpace(resolved)
					fmt.Printf("Using alias '%s' resolved to version: %s\n", v, version)
				} else {
					// don't log anything — just fallback silently
					version = v
				}
			}
		} else {
			v, err := utils.GetVersionFromProjectFile()
			if err != nil {
				return cli.Exit("No version provided and no .jfrog-version file found", 1)
			}

			if utils.IsVersionConstraint(v) {
				installedVersions, err := utils.GetInstalledVersions()
				if err != nil {
					return fmt.Errorf("failed to get installed cli versions: %v", err)
				}

				matchingVersion, err := utils.FindMatchingVersion(v, installedVersions)
				if err != nil {
					return fmt.Errorf("please install a cli version %s which is compatible with the project", v)
				}

				version = matchingVersion
			} else {
				version = v
				fmt.Printf("Using version from .jfrog-version: %s\n", version)
			}
		}

		isBlocked, err := utils.IsVersionBlocked(version)
		if err != nil {
			return fmt.Errorf("failed to check if version is blocked: %w", err)
		}
		if isBlocked {
			return cli.Exit("This version is blocked for this project. Please use another version.", 1)
		}

		// For non-latest versions, check if binary exists and install if needed
		if c.Args().Len() == 0 || strings.ToLower(c.Args().Get(0)) != "latest" {
			binPath := filepath.Join(utils.JFCMVersions, version, utils.BinaryName)
			fmt.Printf("Checking if binary exists at: %s\n", binPath)

			if _, err := os.Stat(binPath); os.IsNotExist(err) {
				fmt.Printf("Version %s not found locally. Installing...\n", version)
				if err := internal.DownloadAndInstall(version); err != nil {
					return fmt.Errorf("auto-install failed: %w", err)
				}
			}
		}

		if err := utils.ValidateVersionAgainstProject(version, versionExplicitlyProvided); err != nil {
			return cli.Exit(fmt.Sprintf("%v", err), 1)
		}

		fmt.Printf("Writing selected version '%s' to config file: %s\n", version, utils.JFCMConfig)
		if err := os.WriteFile(utils.JFCMConfig, []byte(version), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		// Set up shim to redirect jf commands to the active version
		fmt.Println("Setting up jf shim...")
		if err := utils.SetupShim(); err != nil {
			return fmt.Errorf("failed to setup shim: %w", err)
		}

		// Update PATH to prioritize jfcm-managed jf over system jf
		fmt.Println("Updating PATH to prioritize jfcm-managed jf...")
		if err := utils.UpdatePATH(); err != nil {
			fmt.Printf("Warning: Failed to update PATH: %v\n", err)
			fmt.Println("You may need to manually add jfcm shim to your PATH")
		}

		// Verify priority is working correctly
		fmt.Println("Verifying jfcm priority...")
		if err := utils.VerifyPriority(); err != nil {
			fmt.Printf("⚠️  Priority verification failed: %v\n", err)
			fmt.Println("This may be due to current shell session not being updated yet.")
			fmt.Println("Please restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc)")
		} else {
			fmt.Println("✅ Priority verification successful")
		}

		fmt.Printf("✅ Successfully activated jf version %s\n", version)
		fmt.Printf("🔧 jfcm-managed jf binary now takes highest priority over system installations\n")
		fmt.Printf("📝 Restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc) to apply changes\n")
		fmt.Printf("🔍 Run 'which jf' to verify jfcm-managed version is being used\n")

		return nil
	},
}
