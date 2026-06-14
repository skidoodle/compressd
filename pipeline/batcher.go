package pipeline

import (
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/skidoodle/compressd/cache"
)

// PebbleWriter defines what we need from a database to batch writes.
type PebbleWriter interface {
	NewBatch() *pebble.Batch
}

// batcher collects metadata updates and flushes them to the database in groups.
// This prevents us from hammering the disk with many small writes.
func (p *Pipeline) batcher(dber PebbleWriter) {
	defer p.batchWg.Done()

	const batchSize = 100
	updates := make([]Update, 0, batchSize)

	flush := func() {
		if len(updates) == 0 {
			return
		}
		batch := dber.NewBatch()
		for _, up := range updates {
			key := []byte(up.Path)
			val := cache.Marshal(up.Size, time.Unix(up.Time, 0))
			if err := batch.Set(key, val, nil); err != nil {
				p.logError(up.Path, fmt.Errorf("failed to add to batch: %v", err))
			}
		}
		if err := batch.Commit(pebble.Sync); err != nil {
			p.logError("", fmt.Errorf("database batch write error: %v", err))
		}
		batch.Close()
		updates = updates[:0]
	}

	for up := range p.updateCh {
		updates = append(updates, up)
		if len(updates) >= batchSize {
			flush()
		}
	}
	flush() // Final flush to catch anything remaining.
}
