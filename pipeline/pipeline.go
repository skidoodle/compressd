package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/skidoodle/compressd/cache"
	"github.com/skidoodle/compressd/engine"
	"github.com/skidoodle/compressd/logger"
)

// Job holds the data for a single file waiting to be processed.
type Job struct {
	Path string
	Info os.FileInfo
}

// Update carries the file metadata we want to persist in the cache.
type Update struct {
	Path string
	Size int64
	Time int64
}

// Pipeline coordinates the entire flow: finding files, processing them, and updating the cache.
type Pipeline struct {
	workers         int
	queue           chan Job
	updateCh        chan Update
	wg              sync.WaitGroup
	batchWg         sync.WaitGroup
	format          string
	quality         int
	renameExtension bool
	store           *cache.Store
	closeOnce       sync.Once
	totalFiles      int64
	processed       int64
	activeMu        sync.Mutex
	activeFile      string
	errors          int64
}

func NewPipeline(workers int, format string, quality int, renameExtension bool, store *cache.Store, totalFiles int64) *Pipeline {
	return &Pipeline{
		workers:         workers,
		queue:           make(chan Job, workers*2),
		updateCh:        make(chan Update, workers*4),
		format:          format,
		quality:         quality,
		renameExtension: renameExtension,
		store:           store,
		totalFiles:      totalFiles,
	}
}

// Start kicks off the worker pool and the background database batcher.
func (p *Pipeline) Start(dber PebbleWriter) {
	p.batchWg.Add(1)
	go p.batcher(dber)

	p.drawProgress()

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Submit adds a new job to the processing queue.
func (p *Pipeline) Submit(job Job) {
	// We use a recovery block to catch any race conditions
	// where main tries to submit while the pipeline is closing.
	defer func() {
		_ = recover()
	}()
	p.queue <- job
}

// Close gracefully shuts down the pipeline, waiting for all work to finish.
func (p *Pipeline) Close() {
	p.closeOnce.Do(func() {
		close(p.queue)
		p.wg.Wait()
		close(p.updateCh)
		p.batchWg.Wait()

		// Clear the status line before exiting.
		p.activeMu.Lock()
		fmt.Fprintf(os.Stderr, "\r\x1b[2K\r")
		p.activeMu.Unlock()
	})
}

func (p *Pipeline) worker() {
	defer p.wg.Done()
	for job := range p.queue {
		p.executeWithRecovery(job)
	}
}

func (p *Pipeline) drawProgress() {
	p.activeMu.Lock()
	defer p.activeMu.Unlock()
	p.drawProgressLocked()
}

// drawProgressLocked renders the progress bar to stderr.
// Requires activeMu to be held.
func (p *Pipeline) drawProgressLocked() {
	processed := atomic.LoadInt64(&p.processed)
	total := p.totalFiles

	var pct float64
	if total > 0 {
		pct = float64(processed) / float64(total) * 100
	}

	const barWidth = 10
	completedWidth := 0
	if total > 0 {
		completedWidth = int(float64(processed) / float64(total) * barWidth)
	}
	if completedWidth > barWidth {
		completedWidth = barWidth
	}

	bar := make([]byte, barWidth)
	for i := 0; i < barWidth; i++ {
		if i < completedWidth {
			bar[i] = '='
		} else {
			bar[i] = '-'
		}
	}

	active := p.activeFile
	if len(active) > 12 {
		active = active[:9] + "..."
	}

	errs := atomic.LoadInt64(&p.errors)

	status := fmt.Sprintf("\rcompressd: [%s] %.1f%% (%d/%d) | J:%d | E:%d | F:%s\x1b[K",
		string(bar), pct, processed, total, p.workers, errs, active)

	fmt.Fprint(os.Stderr, status)
}

func (p *Pipeline) printCompleted(path string) {
	// Successful files are tracked via the progress bar.
}

// logError prints a formatted error and then redraws the progress bar.
func (p *Pipeline) logError(path string, err error) {
	atomic.AddInt64(&p.errors, 1)

	p.activeMu.Lock()
	defer p.activeMu.Unlock()

	// Clear the status line, print the error, and restore the bar.
	fmt.Fprint(os.Stderr, "\r\x1b[2K\r")
	logger.LogErr(path, stripErrorStack(err))
	p.drawProgressLocked()
}

func (p *Pipeline) logWarning(path string, msg string) {
	atomic.AddInt64(&p.errors, 1)

	p.activeMu.Lock()
	defer p.activeMu.Unlock()

	fmt.Fprint(os.Stderr, "\r\x1b[2K\r")
	logger.LogWarn(path, msg)
	p.drawProgressLocked()
}

// stripErrorStack removes the noisy libvips stack traces from error messages.
func stripErrorStack(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.Index(msg, "Stack:"); idx != -1 {
		msg = strings.TrimSpace(msg[:idx])
	}
	return msg
}

func (p *Pipeline) executeWithRecovery(job Job) {
	defer func() {
		// Catch any panics within a worker so we don't crash the whole app.
		if r := recover(); r != nil {
			p.logError(job.Path, fmt.Errorf("recovered from critical runtime panic: %v", r))
		}
		atomic.AddInt64(&p.processed, 1)
		p.drawProgress()
	}()

	p.activeMu.Lock()
	p.activeFile = filepath.Base(job.Path)
	p.activeMu.Unlock()
	p.drawProgress()

	fi, err := os.Stat(job.Path)
	if err != nil {
		p.logWarning(job.Path, fmt.Sprintf("failed to stat source file: %v", err))
		return
	}

	// Check the cache to see if we've already processed this exact file.
	cachedVal, err := p.store.Get([]byte(job.Path))
	if err == nil {
		cachedSize, cachedTime := cache.Unmarshal(cachedVal)
		if fi.Size() == cachedSize && fi.ModTime().Unix() == cachedTime.Unix() {
			return
		}
	}

	// Determine the target path.
	targetPath := job.Path
	if p.renameExtension {
		ext := filepath.Ext(job.Path)
		targetPath = job.Path[:len(job.Path)-len(ext)] + "." + p.format
	}

	// Process the image.
	if err := engine.ProcessImage(job.Path, targetPath, p.format, p.quality); err != nil {
		p.logError(job.Path, err)
		return
	}

	// If we changed the extension, we need to clean up the original file and cache.
	if targetPath != job.Path {
		if err := os.Remove(job.Path); err != nil {
			p.logWarning(job.Path, fmt.Sprintf("failed to remove original file after conversion: %v", err))
		}
		_ = p.store.Delete([]byte(job.Path))
	}

	// Get the new file stats so we can update the cache accurately.
	newFi, err := os.Stat(targetPath)
	if err != nil {
		p.logError(targetPath, fmt.Errorf("failed to stat processed file: %v", err))
		return
	}

	// Send the metadata update to the background batcher.
	p.updateCh <- Update{
		Path: targetPath,
		Size: newFi.Size(),
		Time: newFi.ModTime().Unix(),
	}

	p.printCompleted(targetPath)
}
