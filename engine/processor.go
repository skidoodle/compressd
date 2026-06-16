package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/skidoodle/compressd/engine/vips_bridge"
)

// ProcessImage loads, encodes, and saves the image to the target path.
// It uses a temporary file and renames it at the end to ensure the update is atomic.
func ProcessImage(srcPath string, targetPath string, format string, quality int) error {
	buf, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	img, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer img.Close()

	width := img.Width()
	height := img.Height()

	format = strings.ToLower(format)
	format = strings.TrimPrefix(format, ".")

	switch format {
	case "webp":
		// WebP has a hard limit of 16383px on either dimension.
		if width > 16383 || height > 16383 {
			return fmt.Errorf("dimensions (%dx%d) exceed maximum WebP limit of 16383px", width, height)
		}
	case "avif":
		// AVIF/AV1 bitstream maxes out at 65535px.
		if width > 65535 || height > 65535 {
			return fmt.Errorf("dimensions (%dx%d) exceed maximum AVIF limit of 65535px", width, height)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Including the format in the extension helps libvips pick the right saver automatically.
	tmpPath := fmt.Sprintf("%s.tmp.%s", targetPath, format)

	// Use our C bridge to save the file. This avoids bringing the heavy image data
	// into the Go heap and works around some AVIF buffer export bugs in libvips.
	if err := vips_bridge.SaveToVipsFile(img, tmpPath, quality); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace file: %w", err)
	}

	return nil
}
