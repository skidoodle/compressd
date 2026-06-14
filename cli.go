package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/skidoodle/compressd/logger"
)

// Config stores our command-line settings.
type Config struct {
	Jobs      int
	Format    string
	Quality   int
	CacheDir  string
	Verbose   bool
	TargetDir string
}

// parseFlags reads command-line arguments and validates them.
func parseFlags() (*Config, error) {
	var jFlag, jobsFlag int
	var fFlag, formatFlag string
	var qFlag, qualityFlag int
	var cFlag, cacheFlag string
	var vFlag, verboseFlag bool

	fs := flag.NewFlagSet("compressd", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: compressd [OPTIONS] <TARGET_DIRECTORY>\n\nOPTIONS:\n")
		fmt.Fprintf(os.Stderr, "  -j, --jobs <int>      Number of parallel processing tasks (default: %d)\n", runtime.NumCPU())
		fmt.Fprintf(os.Stderr, "  -f, --format <str>    Output format: webp or avif (default: \"webp\")\n")
		fmt.Fprintf(os.Stderr, "  -q, --quality <int>   Image quality [1-100] (default: 75)\n")
		fmt.Fprintf(os.Stderr, "  -c, --cache <path>    Where to store the persistent cache index\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose         Enable detailed logging\n")
	}

	fs.IntVar(&jFlag, "j", 0, "")
	fs.IntVar(&jobsFlag, "jobs", 0, "")
	fs.StringVar(&fFlag, "f", "webp", "")
	fs.StringVar(&formatFlag, "format", "webp", "")
	fs.IntVar(&qFlag, "q", 75, "")
	fs.IntVar(&qualityFlag, "quality", 75, "")
	fs.StringVar(&cFlag, "c", "", "")
	fs.StringVar(&cacheFlag, "cache", "", "")
	fs.BoolVar(&vFlag, "v", false, "")
	fs.BoolVar(&verboseFlag, "verbose", false, "")

	if len(os.Args) == 1 {
		fs.Usage()
		os.Exit(0)
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return nil, err
	}

	cfg := &Config{}

	// Determine concurrency level.
	if jFlag != 0 {
		cfg.Jobs = jFlag
	} else if jobsFlag != 0 {
		cfg.Jobs = jobsFlag
	} else {
		cfg.Jobs = runtime.NumCPU()
	}

	// Validate output format.
	if fFlag != "webp" {
		cfg.Format = fFlag
	} else {
		cfg.Format = formatFlag
	}
	if cfg.Format != "webp" && cfg.Format != "avif" {
		return nil, fmt.Errorf("invalid format: %s (must be webp or avif)", cfg.Format)
	}

	// Validate quality range.
	if qFlag != 75 {
		cfg.Quality = qFlag
	} else {
		cfg.Quality = qualityFlag
	}
	if cfg.Quality < 1 || cfg.Quality > 100 {
		return nil, fmt.Errorf("invalid quality: %d (must be [1-100])", cfg.Quality)
	}

	// Set up the cache directory. Defaults to ~/.cache/compressd if not provided.
	var rawCache string
	if cFlag != "" {
		rawCache = cFlag
	} else {
		rawCache = cacheFlag
	}
	if rawCache == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("unable to determine user home directory for default cache path: %w", err)
		}
		rawCache = filepath.Join(home, ".cache", "compressd")
	}
	absCache, err := filepath.Abs(rawCache)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for cache: %w", err)
	}
	cfg.CacheDir = absCache

	cfg.Verbose = vFlag || verboseFlag
	logger.Verbose = cfg.Verbose

	// The last argument must be the target directory.
	args := fs.Args()
	if len(args) < 1 {
		return nil, fmt.Errorf("missing target directory")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments")
	}
	targetDir := args[0]
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for target directory: %w", err)
	}
	cfg.TargetDir = absTarget

	return cfg, nil
}

// handleSignals listens for termination signals and kicks off a graceful shutdown.
func handleSignals(cancel context.CancelFunc, stop func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.LogWarn("", "Interrupted! Shutting down gracefully...")

		// Stop discovering new files.
		cancel()

		// Finish current work and flush remaining data.
		stop()

		os.Exit(0)
	}()
}
