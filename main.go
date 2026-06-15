package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/skidoodle/compressd/cache"
	"github.com/skidoodle/compressd/engine"
	"github.com/skidoodle/compressd/logger"
	"github.com/skidoodle/compressd/pipeline"
)

var (
	Version = "dev"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		logger.LogErr("", fmt.Sprintf("invalid configuration: %v", err))
		os.Exit(1)
	}

	if cfg.Verbose {
		logger.LogInfo("", fmt.Sprintf("compressd %s startup", Version))
	}

	// Use a cancellable context to allow for a clean shutdown on signals.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize our persistent metadata store.
	store, err := cache.Open(cfg.CacheDir, cfg.Verbose)
	if err != nil {
		logger.LogFatal("", fmt.Sprintf("failed to initialize cache store: %v", err))
	}
	defer store.Close()

	// Clean up any temporary files left over from previous interrupted runs.
	if err := sanitizeDir(cfg.TargetDir); err != nil {
		logger.LogFatal("", fmt.Sprintf("failed during Phase 0 sanitization: %v", err))
	}

	// Prepare the libvips image engine.
	if err := engine.Init(cfg.Verbose); err != nil {
		logger.LogFatal("", fmt.Sprintf("failed to initialize libvips engine: %v", err))
	}
	defer engine.Shutdown()

	// Pre-scan to count files so we can provide an accurate progress bar.
	var totalFiles int64
	_ = filepath.WalkDir(cfg.TargetDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() && isImage(path) {
			totalFiles++
		}
		return nil
	})

	// Set up the processing pipeline.
	p := pipeline.NewPipeline(cfg.Jobs, cfg.Format, cfg.Quality, cfg.Extension, store, totalFiles)
	p.Start(store)
	defer p.Close()

	// Set up signal handling for graceful exits.
	handleSignals(cancel, p.Close)

	// Crawl the directory and feed jobs into the pipeline.
	err = filepath.WalkDir(cfg.TargetDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() || !isImage(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			logger.LogWarn(path, fmt.Sprintf("failed to get file info: %v", err))
			return nil
		}

		p.Submit(pipeline.Job{Path: path, Info: info})
		return nil
	})

	if err != nil && err != context.Canceled {
		logger.LogFatal("", fmt.Sprintf("directory walk failed: %v", err))
	}
}

func isImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".tiff":
		return true
	}
	return false
}

// sanitizeDir removes orphaned .tmp files that might have been left behind.
func sanitizeDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			name := d.Name()
			if strings.HasSuffix(name, ".tmp.webp") || strings.HasSuffix(name, ".tmp.avif") {
				if err := os.Remove(path); err != nil {
					logger.LogWarn(path, fmt.Sprintf("failed to prune legacy temp file: %v", err))
				} else {
					logger.LogInfo(path, "pruned legacy temp file")
				}
			}
		}
		return nil
	})
}
