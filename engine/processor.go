package engine

import (
	"fmt"
	"os"

	"github.com/davidbyttow/govips/v2/vips"
)

// ProcessImage handles the heavy lifting: loading, encoding, and saving the image.
// It writes to a temporary file first and then renames it for an atomic replacement.
func ProcessImage(path string, format string, quality int) error {
	img, err := vips.NewImageFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer img.Close()

	var buf []byte
	switch format {
	case "webp":
		width := img.Width()
		height := img.Height()
		// WebP has a hard limit of 16383px.
		if width > 16383 || height > 16383 {
			return fmt.Errorf("dimensions (%dx%d) exceed maximum WebP limit of 16383px", width, height)
		}

		params := vips.NewWebpExportParams()
		params.Quality = quality
		b, _, err := img.ExportWebp(params)
		if err != nil {
			return fmt.Errorf("failed to export webp: %w", err)
		}
		buf = b
	case "avif":
		width := img.Width()
		height := img.Height()
		// AVIF/AV1 bitstream maxes out at 65535px.
		if width > 65535 || height > 65535 {
			return fmt.Errorf("dimensions (%dx%d) exceed maximum AVIF limit of 65535px", width, height)
		}

		params := vips.NewAvifExportParams()
		params.Quality = quality
		b, _, err := img.ExportAvif(params)
		if err != nil {
			return fmt.Errorf("failed to export avif: %w", err)
		}
		buf = b
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	tmpPath := path + "." + format + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Swap the new file into place.
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) // Cleanup if the rename failed.
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	return nil
}
