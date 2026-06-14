package logger

import (
	"fmt"
	"os"
)

var Verbose bool

func LogInfo(path, msg string) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "compressd: info: %s%s\n", formatPath(path), msg)
	}
}

func LogInfof(format string, args ...any) {
	if Verbose {
		msg := fmt.Sprintf(format, args...)
		LogInfo("", msg)
	}
}

func LogWarn(path, msg string) {
	fmt.Fprintf(os.Stderr, "compressd: warning: %s%s\n", formatPath(path), msg)
}

func LogErr(path, msg string) {
	fmt.Fprintf(os.Stderr, "compressd: error: %s%s\n", formatPath(path), msg)
}

func LogFatal(path, msg string) {
	LogErr(path, msg)
	os.Exit(2)
}

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
