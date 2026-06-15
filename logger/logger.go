package logger

import (
	"fmt"
	"os"
	"sync"
)

var (
	Verbose bool
	mu      sync.Mutex
)

// LogInfo prints an informational message if Verbose is enabled.
func LogInfo(path, msg string) {
	if Verbose {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(os.Stderr, "compressd: info: %s%s\n", formatPath(path), msg)
	}
}

// LogInfof prints a formatted informational message if Verbose is enabled.
func LogInfof(format string, args ...any) {
	if Verbose {
		msg := fmt.Sprintf(format, args...)
		LogInfo("", msg)
	}
}

// LogWarn prints a warning message.
func LogWarn(path, msg string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(os.Stderr, "compressd: warning: %s%s\n", formatPath(path), msg)
}

// LogErr prints an error message.
func LogErr(path, msg string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(os.Stderr, "compressd: error: %s%s\n", formatPath(path), msg)
}

// LogFatal prints an error and exits the program.
func LogFatal(path, msg string) {
	LogErr(path, msg)
	os.Exit(2)
}

// LogDBFatal is specifically for database-related fatal errors.
func LogDBFatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	LogErr("", "DB fatal error: "+msg)
}

func formatPath(p string) string {
	if p == "" {
		return ""
	}
	return p + ": "
}
