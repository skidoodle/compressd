package engine

import "github.com/davidbyttow/govips/v2/vips"

// Init starts up libvips. We set concurrency to 1 so that we don't
// fight with the Go scheduler for thread control.
func Init(verbose bool) error {
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

// Shutdown cleans up the global libvips context.
func Shutdown() {
	vips.Shutdown()
}
