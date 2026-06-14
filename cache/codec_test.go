package cache

import (
	"testing"
	"time"
)

func TestMarshalUnmarshal(t *testing.T) {
	testTime := time.Unix(1718274567, 0)
	var testSize int64 = 1048576

	buf := Marshal(testSize, testTime)
	if len(buf) != 16 {
		t.Fatalf("expected 16 bytes payload, got %d", len(buf))
	}

	size, modTime := Unmarshal(buf)
	if size != testSize {
		t.Errorf("expected size %d, got %d", testSize, size)
	}

	if modTime.Unix() != testTime.Unix() {
		t.Errorf("expected modTime %v, got %v", testTime, modTime)
	}
}

func TestUnmarshalShortBuffer(t *testing.T) {
	size, modTime := Unmarshal([]byte{0, 1, 2})
	if size != 0 || !modTime.IsZero() {
		t.Errorf("expected zero values for short buffer, got size=%d, modTime=%v", size, modTime)
	}
}
