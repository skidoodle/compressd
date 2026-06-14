package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessImage(t *testing.T) {
	if err := Init(false); err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}
	defer Shutdown()

	// 1x1 pixel transparent GIF
	gifData := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
		0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff,
		0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
		0x01, 0x00, 0x3b,
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.gif")
	if err := os.WriteFile(tempFile, gifData, 0644); err != nil {
		t.Fatalf("failed to write test GIF: %v", err)
	}

	// Test WebP conversion.
	if err := ProcessImage(tempFile, "webp", 75); err != nil {
		t.Fatalf("ProcessImage webp failed: %v", err)
	}

	fi, err := os.Stat(tempFile)
	if err != nil {
		t.Fatalf("failed to stat processed file: %v", err)
	}
	if fi.Size() == 0 {
		t.Error("processed file is empty")
	}

	// Test AVIF conversion.
	tempFile2 := filepath.Join(tempDir, "test2.gif")
	if err := os.WriteFile(tempFile2, gifData, 0644); err != nil {
		t.Fatalf("failed to write test GIF 2: %v", err)
	}

	if err := ProcessImage(tempFile2, "avif", 75); err != nil {
		t.Fatalf("ProcessImage avif failed: %v", err)
	}

	fi2, err := os.Stat(tempFile2)
	if err != nil {
		t.Fatalf("failed to stat processed file 2: %v", err)
	}
	if fi2.Size() == 0 {
		t.Error("processed file 2 is empty")
	}
}
