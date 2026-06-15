package engine

import (
	"os"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/skidoodle/compressd/engine/vips_bridge"
)

// Init starts up libvips. We set concurrency to 1 so that we don't
// fight with the Go scheduler for thread control.
func Init(verbose bool) error {
	// Look for bundled modules in ../lib/vips-modules relative to the executable.
	exePath, err := os.Executable()
	if err == nil {
		libDir := filepath.Join(filepath.Dir(exePath), "lib", "vips-modules")
		if _, err := os.Stat(libDir); err == nil {
			// Prepend to module path so our bundled ones are preferred.
			oldPath := os.Getenv("VIPS_MODULE_PATH")
			newPath := libDir
			if oldPath != "" {
				newPath = libDir + string(os.PathListSeparator) + oldPath
			}
			os.Setenv("VIPS_MODULE_PATH", newPath)
		}
	}

	if verbose {
		vips.LoggingSettings(nil, vips.LogLevelInfo)
	} else {
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			// Mute logs in non-verbose mode.
		}, vips.LogLevelError)
	}

	return vips.Startup(&vips.Config{
		ConcurrencyLevel: 1,
		MaxCacheMem:      0,
	})
}

// Support holds information about which formats are supported by the current libvips build.
type Support struct {
	WebpLoad bool
	WebpSave bool
	AvifLoad bool
	AvifSave bool
}

// GetSupport queries libvips for runtime format support.
func GetSupport() Support {
	return Support{
		WebpLoad: vips_bridge.HasLoader("webp"),
		WebpSave: vips_bridge.HasSaver("webp"),
		AvifLoad: vips_bridge.HasLoader("avif"),
		AvifSave: vips_bridge.HasSaver("avif"),
	}
}

// Shutdown cleans up the global libvips context.
func Shutdown() {
	vips.Shutdown()
}
