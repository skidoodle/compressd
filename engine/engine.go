package engine

import (
	"fmt"
	"os"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/skidoodle/compressd/engine/vips_bridge"
)

// Init starts up libvips. We set concurrency to 1 so that we don't
// fight with the Go scheduler for thread control.
func Init(verbose bool) error {
	// Set debug flags.
	if verbose {
		os.Setenv("G_MESSAGES_DEBUG", "all")
		os.Setenv("VIPS_DEBUG", "1")
		vips_bridge.SetEnv("G_MESSAGES_DEBUG", "all")
		vips_bridge.SetEnv("VIPS_DEBUG", "1")
		vips.LoggingSettings(nil, vips.LogLevelInfo)
	} else {
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			// Mute logs in non-verbose mode.
		}, vips.LogLevelError)
	}

	if verbose {
		fmt.Printf("compressd: info: VIPS_MODULE_PATH is %q\n", os.Getenv("VIPS_MODULE_PATH"))
		fmt.Printf("compressd: info: LIBHEIF_PLUGIN_PATH is %q\n", os.Getenv("LIBHEIF_PLUGIN_PATH"))
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
