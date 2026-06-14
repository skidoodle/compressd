package cache

import (
	"encoding/binary"
	"time"
)

type Metadata struct {
	Size    int64
	ModTime time.Time
}

// Marshal converts file stats into a 16-byte binary format for storage.
func Marshal(size int64, modTime time.Time) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(size))
	binary.BigEndian.PutUint64(buf[8:16], uint64(modTime.Unix()))
	return buf
}

// Unmarshal decodes our 16-byte metadata back into a usable format.
func Unmarshal(buf []byte) (int64, time.Time) {
	if len(buf) < 16 {
		return 0, time.Time{}
	}
	size := binary.BigEndian.Uint64(buf[0:8])
	sec := binary.BigEndian.Uint64(buf[8:16])
	return int64(size), time.Unix(int64(sec), 0)
}
