package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var HealthCheck = &cli.Command{
	Name:        "health-check",
	Usage:       "Perform comprehensive health check of jfvm installation",
	Description: "Verifies jfvm setup, priority status, system compatibility, and performance",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Show detailed health information",
		},
		&cli.BoolFlag{
			Name:    "fix",
			Aliases: []string{"f"},
			Usage:   "Attempt to fix detected issues automatically",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "Output results in JSON format",
		},
		&cli.BoolFlag{
			Name:    "performance",
			Aliases: []string{"p"},
			Usage:   "Include performance benchmarks",
		},
		&cli.BoolFlag{
			Name:    "security",
			Aliases: []string{"s"},
			Usage:   "Include security checks",
		},
	},
	Action: func(c *cli.Context) error {
		verbose := c.Bool("verbose")
		fix := c.Bool("fix")
		json := c.Bool("json")
		performance := c.Bool("performance")
		security := c.Bool("security")

		if json {
			return runHealthCheckJSON(verbose, fix, performance, security)
		}

		return runHealthCheck(verbose, fix, performance, security)
	},
}

type HealthStatus struct {
	Status    string `json:"status"`
	Component string `json:"component"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Fixable   bool   `json:"fixable,omitempty"`
}

type HealthReport struct {
	Timestamp    time.Time      `json:"timestamp"`
	Platform     string         `json:"platform"`
	Architecture string         `json:"architecture"`
	Overall      string         `json:"overall"`
	Checks       []HealthStatus `json:"checks"`
	Summary      map[string]int `json:"summary"`
}

func runHealthCheck(verbose, fix, performance, security bool) error {
	fmt.Println("🏥 jfvm Health Check")
	fmt.Println("===================")
	fmt.Println()

	report := &HealthReport{
		Timestamp:    time.Now(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Checks:       []HealthStatus{},
		Summary:      map[string]int{"pass": 0, "fail": 0, "warn": 0},
	}

	// 1. System Environment Check
	fmt.Println("1. 🔧 System Environment")
	startCount := len(report.Checks)
	checkSystemEnvironment(report, verbose)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 2. jfvm Installation Check
	fmt.Println("2. 📦 jfvm Installation")
	startCount = len(report.Checks)
	checkJfvmInstallation(report, verbose)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 3. Shim Setup Check
	fmt.Println("3. 🔗 Shim Setup")
	startCount = len(report.Checks)
	checkShimSetup(report, verbose, fix)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 4. PATH Priority Check
	fmt.Println("4. 🎯 PATH Priority")
	startCount = len(report.Checks)
	checkPathPriority(report, verbose, fix)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 5. Active Version Check
	fmt.Println("5. 📋 Active Version")
	startCount = len(report.Checks)
	checkActiveVersion(report, verbose)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 6. Binary Execution Check
	fmt.Println("6. ⚡ Binary Execution")
	startCount = len(report.Checks)
	checkBinaryExecution(report, verbose)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 7. Network Connectivity Check
	fmt.Println("7. 🌐 Network Connectivity")
	startCount = len(report.Checks)
	checkNetworkConnectivity(report, verbose)
	printHealthResults(report.Checks[startCount:], verbose)
	fmt.Println()

	// 8. Performance Check (optional)
	if performance {
		fmt.Println("8. 🚀 Performance")
		startCount = len(report.Checks)
		checkPerformance(report, verbose)
		printHealthResults(report.Checks[startCount:], verbose)
		fmt.Println()
	}

	// 9. Security Check (optional)
	if security {
		fmt.Println("9. 🔒 Security")
		startCount = len(report.Checks)
		checkSecurity(report, verbose)
		printHealthResults(report.Checks[startCount:], verbose)
		fmt.Println()
	}

	// Summary
	fmt.Println("📊 Health Check Summary")
	fmt.Println("=======================")
	fmt.Printf("✅ Passed: %d\n", report.Summary["pass"])
	fmt.Printf("❌ Failed: %d\n", report.Summary["fail"])
	fmt.Printf("⚠️  Warnings: %d\n", report.Summary["warn"])

	// Determine overall status
	if report.Summary["fail"] > 0 {
		report.Overall = "FAILED"
		fmt.Printf("\n❌ Overall Status: FAILED - %d critical issues found\n", report.Summary["fail"])
	} else if report.Summary["warn"] > 0 {
		report.Overall = "WARNING"
		fmt.Printf("\n⚠️  Overall Status: WARNING - %d non-critical issues found\n", report.Summary["warn"])
	} else {
		report.Overall = "HEALTHY"
		fmt.Printf("\n✅ Overall Status: HEALTHY - All checks passed\n")
	}

	if fix && report.Summary["fail"] > 0 {
		fmt.Println("\n🔧 Attempting to fix issues...")
		attemptFixes(report)
	}

	return nil
}

func checkSystemEnvironment(report *HealthReport, verbose bool) {
	// Check OS compatibility
	status := HealthStatus{Component: "OS Compatibility"}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		status.Status = "pass"
		status.Message = fmt.Sprintf("OS %s is supported", runtime.GOOS)
	} else {
		status.Status = "fail"
		status.Message = fmt.Sprintf("OS %s is not officially supported", runtime.GOOS)
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check architecture
	status = HealthStatus{Component: "Architecture"}
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		status.Status = "pass"
		status.Message = fmt.Sprintf("Architecture %s is supported", runtime.GOARCH)
	} else {
		status.Status = "warn"
		status.Message = fmt.Sprintf("Architecture %s may have limited support", runtime.GOARCH)
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check shell environment
	status = HealthStatus{Component: "Shell Environment"}
	shell := utils.GetCurrentShell()
	if shell != "" {
		status.Status = "pass"
		status.Message = fmt.Sprintf("Shell %s detected", shell)
		if verbose {
			status.Details = fmt.Sprintf("Profile file: %s", utils.GetShellProfile(shell))
		}
	} else {
		status.Status = "warn"
		status.Message = "Shell detection failed"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkJfvmInstallation(report *HealthReport, verbose bool) {
	// Check jfvm root directory
	status := HealthStatus{Component: "jfvm Root Directory"}
	if _, err := os.Stat(utils.JfvmRoot); err == nil {
		status.Status = "pass"
		status.Message = "jfvm root directory exists"
		if verbose {
			status.Details = utils.JfvmRoot
		}
	} else {
		status.Status = "fail"
		status.Message = "jfvm root directory missing"
		status.Fixable = true
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check versions directory
	status = HealthStatus{Component: "Versions Directory"}
	if _, err := os.Stat(utils.JfvmVersions); err == nil {
		status.Status = "pass"
		status.Message = "Versions directory exists"
	} else {
		status.Status = "warn"
		status.Message = "Versions directory missing (will be created on first install)"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check aliases directory
	status = HealthStatus{Component: "Aliases Directory"}
	if _, err := os.Stat(utils.JfvmAliases); err == nil {
		status.Status = "pass"
		status.Message = "Aliases directory exists"
	} else {
		status.Status = "warn"
		status.Message = "Aliases directory missing (will be created on first alias)"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkShimSetup(report *HealthReport, verbose bool, fix bool) {
	// Check shim directory
	status := HealthStatus{Component: "Shim Directory"}
	if _, err := os.Stat(utils.JfvmShim); err == nil {
		status.Status = "pass"
		status.Message = "Shim directory exists"
	} else {
		status.Status = "fail"
		status.Message = "Shim directory missing"
		status.Fixable = true
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check shim binary
	status = HealthStatus{Component: "Shim Binary"}
	shimPath := filepath.Join(utils.JfvmShim, utils.BinaryName)
	if _, err := os.Stat(shimPath); err == nil {
		status.Status = "pass"
		status.Message = "Shim binary exists"
		if verbose {
			status.Details = shimPath
		}
	} else {
		status.Status = "fail"
		status.Message = "Shim binary missing"
		status.Fixable = true
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check shim permissions
	if runtime.GOOS != "windows" {
		status = HealthStatus{Component: "Shim Permissions"}
		if info, err := os.Stat(shimPath); err == nil {
			if info.Mode()&0111 != 0 {
				status.Status = "pass"
				status.Message = "Shim has correct permissions"
			} else {
				status.Status = "fail"
				status.Message = "Shim is not executable"
				status.Fixable = true
			}
		} else {
			status.Status = "fail"
			status.Message = "Cannot check shim permissions"
		}
		report.Checks = append(report.Checks, status)
		report.Summary[status.Status]++
	}
}

func checkPathPriority(report *HealthReport, verbose bool, fix bool) {
	// Check PATH priority
	status := HealthStatus{Component: "PATH Priority"}
	if err := utils.VerifyPriority(); err == nil {
		status.Status = "pass"
		status.Message = "jfvm has highest priority in PATH"
	} else {
		status.Status = "fail"
		status.Message = "jfvm does not have highest priority in PATH"
		status.Details = err.Error()
		status.Fixable = true
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check which jf is active
	status = HealthStatus{Component: "Active jf Binary"}
	jfPath, err := exec.LookPath("jf")
	if err != nil {
		status.Status = "fail"
		status.Message = "jf binary not found in PATH"
	} else {
		shimDir := filepath.Clean(utils.JfvmShim)
		jfDir := filepath.Clean(filepath.Dir(jfPath))
		if verbose {
			fmt.Printf("[DEBUG] which jf: %s\n", jfPath)
			fmt.Printf("[DEBUG] utils.JfvmShim: %s\n", shimDir)
			fmt.Printf("[DEBUG] filepath.Dir(jfPath): %s\n", jfDir)
		}
		if jfDir == shimDir {
			status.Status = "pass"
			status.Message = "jfvm-managed jf is active"
			if verbose {
				status.Details = jfPath
			}
		} else {
			status.Status = "fail"
			status.Message = "System jf is active (not jfvm-managed)"
			status.Details = fmt.Sprintf("Expected: %s/jf, Found: %s", shimDir, jfPath)
			status.Fixable = true
		}
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkActiveVersion(report *HealthReport, verbose bool) {
	// Check active version
	status := HealthStatus{Component: "Active Version"}
	activeVersion, err := utils.GetActiveVersion()
	if err != nil {
		status.Status = "warn"
		status.Message = "No active version set"
	} else {
		status.Status = "pass"
		status.Message = fmt.Sprintf("Active version: %s", activeVersion)

		// Check if binary exists
		binaryPath := filepath.Join(utils.JfvmVersions, activeVersion, utils.BinaryName)
		if _, err := os.Stat(binaryPath); err == nil {
			status.Details = "Binary exists"
		} else {
			status.Status = "fail"
			status.Message = fmt.Sprintf("Active version %s binary missing", activeVersion)
			status.Details = binaryPath
		}
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check installed versions
	status = HealthStatus{Component: "Installed Versions"}
	if entries, err := os.ReadDir(utils.JfvmVersions); err == nil {
		count := len(entries)
		if count > 0 {
			status.Status = "pass"
			status.Message = fmt.Sprintf("%d version(s) installed", count)
			if verbose {
				var versions []string
				for _, entry := range entries {
					if entry.IsDir() {
						versions = append(versions, entry.Name())
					}
				}
				status.Details = strings.Join(versions, ", ")
			}
		} else {
			status.Status = "warn"
			status.Message = "No versions installed"
		}
	} else {
		status.Status = "warn"
		status.Message = "Cannot read versions directory"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkBinaryExecution(report *HealthReport, verbose bool) {
	// Test jf execution
	status := HealthStatus{Component: "jf Execution"}
	cmd := exec.Command("jf", "--version")
	output, err := cmd.Output()
	if err != nil {
		status.Status = "fail"
		status.Message = "jf execution failed"
		status.Details = err.Error()
	} else {
		status.Status = "pass"
		status.Message = "jf execution successful"
		if verbose {
			status.Details = strings.TrimSpace(string(output))
		}
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Test jfvm execution
	status = HealthStatus{Component: "jfvm Execution"}
	cmd = exec.Command("jfvm", "--help")
	if err := cmd.Run(); err != nil {
		status.Status = "fail"
		status.Message = "jfvm execution failed"
		status.Details = err.Error()
	} else {
		status.Status = "pass"
		status.Message = "jfvm execution successful"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkNetworkConnectivity(report *HealthReport, verbose bool) {
	// Test GitHub API connectivity
	status := HealthStatus{Component: "GitHub API"}
	cmd := exec.Command("curl", "-s", "--max-time", "5", "https://api.github.com")
	if err := cmd.Run(); err != nil {
		status.Status = "warn"
		status.Message = "GitHub API connectivity failed"
		status.Details = "Cannot fetch latest version information"
	} else {
		status.Status = "pass"
		status.Message = "GitHub API connectivity successful"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Test JFrog releases connectivity
	status = HealthStatus{Component: "JFrog Releases"}
	cmd = exec.Command("curl", "-s", "--max-time", "5", "https://releases.jfrog.io")
	if err := cmd.Run(); err != nil {
		status.Status = "warn"
		status.Message = "JFrog releases connectivity failed"
		status.Details = "Cannot download JFrog CLI binaries"
	} else {
		status.Status = "pass"
		status.Message = "JFrog releases connectivity successful"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkPerformance(report *HealthReport, verbose bool) {
	// Test jfvm command performance
	status := HealthStatus{Component: "jfvm Performance"}
	start := time.Now()
	cmd := exec.Command("jfvm", "list")
	if err := cmd.Run(); err != nil {
		status.Status = "fail"
		status.Message = "jfvm list command failed"
	} else {
		duration := time.Since(start)
		if duration < 100*time.Millisecond {
			status.Status = "pass"
			status.Message = fmt.Sprintf("jfvm list completed in %v", duration)
		} else {
			status.Status = "warn"
			status.Message = fmt.Sprintf("jfvm list took %v (slow)", duration)
		}
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Test jf command performance
	status = HealthStatus{Component: "jf Performance"}
	start = time.Now()
	cmd = exec.Command("jf", "--version")
	if err := cmd.Run(); err != nil {
		status.Status = "fail"
		status.Message = "jf version command failed"
	} else {
		duration := time.Since(start)
		if duration < 500*time.Millisecond {
			status.Status = "pass"
			status.Message = fmt.Sprintf("jf version completed in %v", duration)
		} else {
			status.Status = "warn"
			status.Message = fmt.Sprintf("jf version took %v (slow)", duration)
		}
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func checkSecurity(report *HealthReport, verbose bool) {
	// Check file permissions
	status := HealthStatus{Component: "File Permissions"}
	shimPath := filepath.Join(utils.JfvmShim, utils.BinaryName)
	if info, err := os.Stat(shimPath); err == nil {
		mode := info.Mode()
		if mode&0077 == 0 { // No world/group write permissions
			status.Status = "pass"
			status.Message = "Shim has secure permissions"
		} else {
			status.Status = "warn"
			status.Message = "Shim has loose permissions"
			status.Details = fmt.Sprintf("Mode: %v", mode)
		}
	} else {
		status.Status = "warn"
		status.Message = "Cannot check shim permissions"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++

	// Check for suspicious files
	status = HealthStatus{Component: "Suspicious Files"}
	suspiciousFound := false
	if entries, err := os.ReadDir(utils.JfvmRoot); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".exe") || strings.HasSuffix(entry.Name(), ".sh") {
				if !strings.Contains(entry.Name(), "jf") && !strings.Contains(entry.Name(), "jfvm") {
					suspiciousFound = true
					break
				}
			}
		}
	}

	if suspiciousFound {
		status.Status = "warn"
		status.Message = "Suspicious files found in jfvm directory"
	} else {
		status.Status = "pass"
		status.Message = "No suspicious files found"
	}
	report.Checks = append(report.Checks, status)
	report.Summary[status.Status]++
}

func printHealthResults(checks []HealthStatus, verbose bool) {
	for _, check := range checks {
		switch check.Status {
		case "pass":
			fmt.Printf("  ✅ %s: %s\n", check.Component, check.Message)
		case "fail":
			fmt.Printf("  ❌ %s: %s\n", check.Component, check.Message)
		case "warn":
			fmt.Printf("  ⚠️  %s: %s\n", check.Component, check.Message)
		}

		if verbose && check.Details != "" {
			fmt.Printf("      Details: %s\n", check.Details)
		}
	}
}

func attemptFixes(report *HealthReport) {
	fmt.Println("🔧 Attempting to fix issues...")

	for _, check := range report.Checks {
		if check.Status == "fail" && check.Fixable {
			fmt.Printf("  Fixing %s...\n", check.Component)

			switch check.Component {
			case "jfvm Root Directory":
				if err := os.MkdirAll(utils.JfvmRoot, 0755); err == nil {
					fmt.Printf("    ✅ Created jfvm root directory\n")
				}
			case "Shim Directory":
				if err := os.MkdirAll(utils.JfvmShim, 0755); err == nil {
					fmt.Printf("    ✅ Created shim directory\n")
				}
			case "Shim Binary":
				if err := utils.SetupShim(); err == nil {
					fmt.Printf("    ✅ Created shim binary\n")
				}
			case "Shim Permissions":
				shimPath := filepath.Join(utils.JfvmShim, utils.BinaryName)
				if err := os.Chmod(shimPath, 0755); err == nil {
					fmt.Printf("    ✅ Fixed shim permissions\n")
				}
			case "PATH Priority", "Active jf Binary":
				if err := utils.UpdatePATH(); err == nil {
					fmt.Printf("    ✅ Updated PATH configuration\n")
				}
			}
		}
	}
}

func runHealthCheckJSON(verbose, fix, performance, security bool) error {
	// Implementation for JSON output would go here
	// For now, just return an error indicating JSON is not implemented
	return fmt.Errorf("JSON output not yet implemented")
}
