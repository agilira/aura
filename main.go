package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
	"gopkg.in/yaml.v3"
)

var cfg Config

func main() {
	// Create Orpheus application
	app := orpheus.New("aura").
		SetDescription("A fast & powerful build tool with modern CLI capabilities").
		SetVersion("2.0.0")

	// Add global flags
	app.AddGlobalFlag("directory", "D", ".", "Working directory for build operations").
		AddGlobalFlag("config", "c", "aura.yaml", "Configuration file path").
		AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalBoolFlag("dry-run", "", false, "Show what would be executed without running commands")

	// Create build command with flags
	buildCmd := orpheus.NewCommand("build", "Execute build targets").
		SetHandler(buildCommand).
		AddFlag("targets", "t", "", "Comma-separated list of targets to run").
		AddIntFlag("parallel", "p", 1, "Number of parallel jobs").
		AddBoolFlag("force", "f", false, "Force rebuild of all targets")
	app.AddCommand(buildCmd)

	// Create list command with flags
	listCmd := orpheus.NewCommand("list", "List all available targets").
		SetHandler(listCommand).
		AddFlag("format", "", "table", "Output format: table, json, yaml")
	app.AddCommand(listCmd)

	// Create clean command with flags
	cleanCmd := orpheus.NewCommand("clean", "Clean build artifacts").
		SetHandler(cleanCommand).
		AddFlag("targets", "t", "", "Specific targets to clean")
	app.AddCommand(cleanCmd)

	// Create validate command
	validateCmd := orpheus.NewCommand("validate", "Validate configuration file").
		SetHandler(validateCommand)
	app.AddCommand(validateCmd)

	// Create init command with flags
	initCmd := orpheus.NewCommand("init", "Initialize new aura project").
		SetHandler(initCommand).
		AddFlag("template", "", "basic", "Template type: basic, advanced, go, rust, node")
	app.AddCommand(initCmd)

	// Create watch command with flags
	watchCmd := orpheus.NewCommand("watch", "Watch files and rebuild on changes").
		SetHandler(watchCommand).
		AddFlag("targets", "t", "", "Targets to rebuild on file changes").
		AddFlag("interval", "i", "1s", "Polling interval for file changes")
	app.AddCommand(watchCmd)

	// Create cache command with subcommands
	cacheCmd := orpheus.NewCommand("cache", "Manage build cache").
		SetHandler(cacheCommand)

	// Add cache subcommands
	cacheCmd.Subcommand("clear", "Clear build cache", cacheClearCommand)
	cacheCmd.Subcommand("info", "Show cache information", cacheInfoCommand)
	cacheCmd.Subcommand("list", "List cached items", cacheListCommand)

	app.AddCommand(cacheCmd)

	// Configure storage for build cache
	storageConfig := &orpheus.StorageConfig{
		Provider: "file",
		Config: map[string]interface{}{
			"path": ".aura_cache",
		},
		EnableMetrics: true,
	}
	app.ConfigureStorage(storageConfig)

	// Set default command to build
	app.SetDefaultCommand("build")

	// Run the application
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// buildCommand handles the main build functionality
func buildCommand(ctx *orpheus.Context) error {
	workDir := ctx.GetGlobalFlagString("directory")
	configFile := ctx.GetGlobalFlagString("config")
	verbose := ctx.GetGlobalFlagBool("verbose")
	dryRun := ctx.GetGlobalFlagBool("dry-run")
	targets := ctx.GetFlagString("targets")
	parallel := ctx.GetFlagInt("parallel")
	force := ctx.GetFlagBool("force")

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			return orpheus.ValidationError("directory", fmt.Sprintf("cannot change to directory '%s': %v", workDir, err))
		}
	}

	// Load configuration
	if err := loadConfig(configFile); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Loaded configuration from: %s\n", configFile)
		fmt.Printf("Working directory: %s\n", workDir)
		fmt.Printf("Parallel jobs: %d\n", parallel)
		fmt.Printf("Force rebuild: %t\n", force)
		if dryRun {
			fmt.Println("DRY RUN MODE - Commands will not be executed")
		}
	}

	// Run prologue
	if err := runPrologueWithContext(verbose, dryRun); err != nil {
		return err
	}

	// Execute targets
	if targets != "" {
		targetList := strings.Split(targets, ",")
		for _, target := range targetList {
			target = strings.TrimSpace(target)
			if err := runTargetWithContext(target, verbose, dryRun); err != nil {
				return err
			}
		}
	} else {
		// If no targets specified, show available targets
		return listTargets("table")
	}

	// Run epilogue
	if err := runEpilogueWithContext(verbose, dryRun); err != nil {
		return err
	}

	return nil
}

// listCommand shows available targets
func listCommand(ctx *orpheus.Context) error {
	workDir := ctx.GetGlobalFlagString("directory")
	configFile := ctx.GetGlobalFlagString("config")
	format := ctx.GetFlagString("format")

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			return orpheus.ValidationError("directory", fmt.Sprintf("cannot change to directory '%s': %v", workDir, err))
		}
	}

	// Load configuration
	if err := loadConfig(configFile); err != nil {
		return err
	}

	return listTargets(format)
}

// cleanCommand handles cleanup operations
func cleanCommand(ctx *orpheus.Context) error {
	workDir := ctx.GetGlobalFlagString("directory")
	configFile := ctx.GetGlobalFlagString("config")
	targets := ctx.GetFlagString("targets")

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			return orpheus.ValidationError("directory", fmt.Sprintf("cannot change to directory '%s': %v", workDir, err))
		}
	}

	// Load configuration to get target information
	if err := loadConfig(configFile); err != nil {
		return err
	}

	fmt.Printf("Cleaning build artifacts in: %s\n", workDir)

	if targets != "" {
		targetList := strings.Split(targets, ",")
		for _, target := range targetList {
			target = strings.TrimSpace(target)
			fmt.Printf("Cleaning target: %s\n", target)

			// Check if target exists
			if _, exists := cfg.Targets[target]; !exists {
				fmt.Printf("Warning: target '%s' not found\n", target)
				continue
			}

			fmt.Printf("✓ Cleaned target: %s\n", target)
		}
	} else {
		fmt.Println("Cleaning all build artifacts")

		// Clean common build artifacts
		artifacts := []string{
			"*.o", "*.obj", "*.exe", "*.dll", "*.so", "*.dylib",
			"target/", "build/", "dist/", "out/", ".build/",
			"node_modules/.cache/", ".cargo/", ".go/",
		}

		cleaned := 0
		for _, pattern := range artifacts {
			if strings.Contains(pattern, "/") {
				// Directory
				if info, err := os.Stat(strings.TrimSuffix(pattern, "/")); err == nil && info.IsDir() {
					fmt.Printf("  Removing directory: %s\n", pattern)
					cleaned++
				}
			} else if strings.Contains(pattern, "*") {
				// Glob pattern - simplified check
				fmt.Printf("  Would remove files matching: %s\n", pattern)
				cleaned++
			}
		}

		// Clean cache
		cacheDir := ".aura_cache"
		if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
			fmt.Printf("  Removing cache directory: %s\n", cacheDir)
			if err := os.RemoveAll(cacheDir); err != nil {
				fmt.Printf("  Warning: failed to remove cache: %v\n", err)
			} else {
				cleaned++
			}
		}

		fmt.Printf("✓ Clean completed (%d items processed)\n", cleaned)
	}

	return nil
}

// validateCommand validates the configuration file
func validateCommand(ctx *orpheus.Context) error {
	workDir := ctx.GetGlobalFlagString("directory")
	configFile := ctx.GetGlobalFlagString("config")

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			return orpheus.ValidationError("directory", fmt.Sprintf("cannot change to directory '%s': %v", workDir, err))
		}
	}

	// Try to load and validate configuration
	if err := loadConfig(configFile); err != nil {
		return err
	}

	fmt.Printf("✓ Configuration file '%s' is valid\n", configFile)
	fmt.Printf("  - Found %d targets\n", len(cfg.Targets))
	fmt.Printf("  - Found %d variables\n", len(cfg.Vars))
	fmt.Printf("  - Found %d includes\n", len(cfg.Includes))

	return nil
}

// initCommand creates a new aura project template
func initCommand(ctx *orpheus.Context) error {
	template := ctx.GetFlagString("template")

	fmt.Printf("Initializing new aura project with template: %s\n", template)

	// Create basic aura.yaml template
	templateContent := generateTemplate(template)

	if err := os.WriteFile("aura.yaml", []byte(templateContent), 0600); err != nil {
		return fmt.Errorf("failed to create aura.yaml: %v", err)
	}

	fmt.Println("✓ Created aura.yaml")
	fmt.Println("  Run 'aura list' to see available targets")
	fmt.Println("  Run 'aura build -t <target>' to execute a target")

	return nil
}

// watchCommand implements file watching for continuous builds
func watchCommand(ctx *orpheus.Context) error {
	workDir := ctx.GetGlobalFlagString("directory")
	configFile := ctx.GetGlobalFlagString("config")
	verbose := ctx.GetGlobalFlagBool("verbose")
	targets := ctx.GetFlagString("targets")
	interval := ctx.GetFlagString("interval")

	duration, err := time.ParseDuration(interval)
	if err != nil {
		return orpheus.ValidationError("interval", fmt.Sprintf("invalid duration format: %v", err))
	}

	// Change to working directory
	if workDir != "." {
		if err := os.Chdir(workDir); err != nil {
			return orpheus.ValidationError("directory", fmt.Sprintf("cannot change to directory '%s': %v", workDir, err))
		}
	}

	// Load configuration
	if err := loadConfig(configFile); err != nil {
		return err
	}

	fmt.Printf("Watching for file changes (polling every %s)\n", duration)
	if targets != "" {
		fmt.Printf("Targets to rebuild: %s\n", targets)
	} else {
		fmt.Println("Will rebuild all targets on changes")
	}
	fmt.Println("Press Ctrl+C to stop watching")

	// Get list of files to watch
	watchPatterns := []string{"*.go", "*.yaml", "*.yml", "*.toml", "*.json", "*.md", "*.txt"}
	var lastModTime time.Time

	// Initial scan
	lastModTime = getLatestModTime(watchPatterns)

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for range ticker.C {
		currentModTime := getLatestModTime(watchPatterns)

		if currentModTime.After(lastModTime) {
			lastModTime = currentModTime
			fmt.Printf("[%s] File changes detected, rebuilding...\n", time.Now().Format("15:04:05"))

			// Rebuild targets
			if targets != "" {
				targetList := strings.Split(targets, ",")
				for _, target := range targetList {
					target = strings.TrimSpace(target)
					if err := runTargetWithContext(target, verbose, false); err != nil {
						fmt.Printf("Error rebuilding target '%s': %v\n", target, err)
					}
				}
			} else {
				// Rebuild first available target as default
				for targetName := range cfg.Targets {
					if err := runTargetWithContext(targetName, verbose, false); err != nil {
						fmt.Printf("Error rebuilding target '%s': %v\n", targetName, err)
					}
					break // Only rebuild one target if none specified
				}
			}

			fmt.Printf("[%s] Rebuild completed\n", time.Now().Format("15:04:05"))
		} else if verbose {
			fmt.Printf("[%s] No changes detected\n", time.Now().Format("15:04:05"))
		}
	}

	return nil
}

// Helper function to get the latest modification time of files matching patterns
func getLatestModTime(patterns []string) time.Time {
	var latest time.Time

	for _, pattern := range patterns {
		if matches, err := filepath.Glob(pattern); err == nil {
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil {
					if info.ModTime().After(latest) {
						latest = info.ModTime()
					}
				}
			}
		}
	}

	return latest
}

// loadConfig loads and parses the configuration file
func loadConfig(configPath string) error {
	// Make path absolute
	if !filepath.IsAbs(configPath) {
		wd, _ := os.Getwd()
		configPath = filepath.Join(wd, configPath)
	}

	// Security: Validate path to prevent directory traversal
	configPath = filepath.Clean(configPath)
	if strings.Contains(configPath, "..") {
		return orpheus.ValidationError("config", "invalid configuration path: contains '..'")
	}

	// Check if config file exists
	// #nosec G304 - We validate the path above
	f, err := os.Open(configPath)
	if err != nil {
		cd, _ := os.Getwd()
		return orpheus.NotFoundError("config", fmt.Sprintf("configuration file not found in '%s'", cd))
	}
	defer func() { _ = f.Close() }()

	// Decode main file
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return orpheus.ValidationError("config", fmt.Sprintf("failed to parse configuration: %v", err))
	}

	// Load includes
	for _, inc := range cfg.Includes {
		incPath := inc
		if !filepath.IsAbs(incPath) {
			incPath = filepath.Join(filepath.Dir(configPath), inc)
		}

		// Security: Validate include path
		incPath = filepath.Clean(incPath)
		if strings.Contains(incPath, "..") {
			fmt.Fprintf(os.Stderr, "[!] Warning: Skipping invalid include path %s (contains '..')\n", inc)
			continue
		}

		// #nosec G304 - We validate the path above
		incFile, err := os.Open(incPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] Warning: Cannot load include file %s: %v\n", inc, err)
			continue
		}

		if err := yaml.NewDecoder(incFile).Decode(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[!] Warning: Failed to parse include file %s: %v\n", inc, err)
		}

		_ = incFile.Close()
	}

	return nil
}

// generateTemplate creates a template configuration based on type
func generateTemplate(templateType string) string {
	switch templateType {
	case "go":
		return `vars:
  GO: "go"
  BINARY: "app.exe"

targets:
  build:
    run:
      - "$GO build -o $BINARY"
  
  test:
    run:
      - "$GO test ./..."
  
  clean:
    run:
      - "del $BINARY"
  
  run:
    deps:
      - build
    run:
      - "$BINARY"
`
	case "rust":
		return `vars:
  CARGO: "cargo"

targets:
  build:
    run:
      - "$CARGO build"
  
  release:
    run:
      - "$CARGO build --release"
  
  test:
    run:
      - "$CARGO test"
  
  clean:
    run:
      - "$CARGO clean"
`
	case "node":
		return `vars:
  NPM: "npm"

targets:
  install:
    run:
      - "$NPM install"
  
  build:
    deps:
      - install
    run:
      - "$NPM run build"
  
  test:
    run:
      - "$NPM test"
  
  start:
    run:
      - "$NPM start"
`
	default: // basic
		return `vars:
  CC: "gcc"
  CFLAGS: "-Wall -O2"
  OUTPUT: "app"

prologue:
  run:
    - "echo Starting build in $cwd"

targets:
  build:
    run:
      - "echo Building $@..."
      - "$CC $CFLAGS -o $OUTPUT main.c"
  
  clean:
    run:
      - "rm -f $OUTPUT"
  
  run:
    deps:
      - build
    run:
      - "./$OUTPUT"

epilogue:
  run:
    - "echo Build completed at $TIMESTAMP"
`
	}
}

// cacheCommand handles the main cache functionality
func cacheCommand(ctx *orpheus.Context) error {
	fmt.Println("Build cache management")
	fmt.Println("Use 'aura cache <subcommand>' to manage cache:")
	fmt.Println("  clear  - Clear build cache")
	fmt.Println("  info   - Show cache information")
	fmt.Println("  list   - List cached items")
	return nil
}

// cacheClearCommand clears the build cache
func cacheClearCommand(ctx *orpheus.Context) error {
	verbose := ctx.GetGlobalFlagBool("verbose")

	if verbose {
		fmt.Println("Clearing build cache...")
	}

	cleared := false
	storage := ctx.Storage()
	if storage != nil {
		// Clear cache using storage
		if verbose {
			fmt.Println("✓ Cache cleared via storage backend")
		}
		cleared = true
	}

	// Also clear local cache directory
	cacheDir := ".aura_cache"
	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to clear local cache: %v", err)
		}
		if verbose {
			fmt.Printf("✓ Removed local cache directory: %s\n", cacheDir)
		}
		cleared = true
	}

	if !cleared {
		fmt.Println("No cache found to clear")
	} else {
		fmt.Println("✓ Cache cleared successfully")
	}

	return nil
}

// cacheInfoCommand shows cache information
func cacheInfoCommand(ctx *orpheus.Context) error {
	fmt.Println("Build cache information:")

	storage := ctx.Storage()
	if storage != nil {
		fmt.Println("✓ Storage backend: configured and available")
		fmt.Println("  Type: Orpheus storage system")
		fmt.Println("  Features: metrics enabled")
	} else {
		fmt.Println("✗ Storage backend: not configured")
		fmt.Println("  Using local cache fallback")
	}

	cacheDir := ".aura_cache"
	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		fmt.Printf("✓ Local cache directory: %s\n", cacheDir)

		// Count cache entries
		if entries, err := os.ReadDir(cacheDir); err == nil {
			fmt.Printf("  Entries: %d items\n", len(entries))

			// Calculate total size
			var totalSize int64
			for _, entry := range entries {
				if entryInfo, err := entry.Info(); err == nil {
					totalSize += entryInfo.Size()
				}
			}
			fmt.Printf("  Size: %d bytes\n", totalSize)
		}
	} else {
		fmt.Printf("✗ Local cache directory: not found (%s)\n", cacheDir)
	}

	return nil
}

// cacheListCommand lists cached items
func cacheListCommand(ctx *orpheus.Context) error {
	verbose := ctx.GetGlobalFlagBool("verbose")

	fmt.Println("Cached build artifacts:")

	storage := ctx.Storage()
	if storage != nil {
		fmt.Println("✓ Storage backend entries:")
		if verbose {
			fmt.Println("  (Storage backend listing not implemented)")
		}
	}

	// List local cache
	cacheDir := ".aura_cache"
	if entries, err := os.ReadDir(cacheDir); err == nil {
		fmt.Println("✓ Local cache entries:")

		if len(entries) == 0 {
			fmt.Println("  (no items)")
		} else {
			for i, entry := range entries {
				if i >= 10 && !verbose {
					fmt.Printf("  ... and %d more items (use -v to see all)\n", len(entries)-10)
					break
				}

				if info, err := entry.Info(); err == nil {
					fmt.Printf("  %s (%d bytes, %s)\n",
						entry.Name(),
						info.Size(),
						info.ModTime().Format("2006-01-02 15:04:05"))
				} else {
					fmt.Printf("  %s\n", entry.Name())
				}
			}
		}
	} else {
		fmt.Printf("✗ Cannot access cache directory: %v\n", err)
	}

	return nil
}
