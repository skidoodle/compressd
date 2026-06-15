//go:build linux

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func init() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exePath)
	libDir := filepath.Join(exeDir, "lib")

	// no local lib folder? nothing to do.
	if _, err := os.Stat(libDir); err != nil {
		return
	}

	// find vips modules subdir
	vipsModPath := libDir
	if entries, err := os.ReadDir(libDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "vips-modules-") {
				vipsModPath = filepath.Join(libDir, entry.Name())
				break
			}
		}
	}

	heifDir := filepath.Join(libDir, "libheif")
	hasHeifDir := false
	if _, err := os.Stat(heifDir); err == nil {
		hasHeifDir = true
	}

	// check if env is already set up to avoid infinite loop
	envSet := os.Getenv("COMPRESSD_ENV_SET") == "1"
	correctModPath := os.Getenv("VIPS_MODULE_PATH") == vipsModPath
	correctHeifPath := !hasHeifDir || os.Getenv("LIBHEIF_PLUGIN_PATH") == heifDir
	correctVipsHome := os.Getenv("VIPSHOME") == libDir

	if envSet && correctModPath && correctHeifPath && correctVipsHome {
		return
	}

	// prep environment
	os.Setenv("VIPS_MODULE_PATH", vipsModPath)
	os.Setenv("LD_LIBRARY_PATH", libDir)
	os.Setenv("VIPS_PREFIX", exeDir)
	os.Setenv("VIPSHOME", libDir)
	if hasHeifDir {
		os.Setenv("LIBHEIF_PLUGIN_PATH", heifDir)
	}
	os.Setenv("COMPRESSD_ENV_SET", "1")

	// re-exec with updated env
	err = syscall.Exec(exePath, os.Args, os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "compressd/env: failed to re-execute binary: %v\n", err)
		os.Exit(1)
	}
}
