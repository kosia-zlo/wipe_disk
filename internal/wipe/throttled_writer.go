package wipe

import (
	"io"
	"os"
	"sync"
	"time"
)

// ThrottledWriter ограничивает скорость записи (thread-safe)
type ThrottledWriter struct {
	file         *os.File
	maxSpeedMBps float64
	lastWrite    time.Time
	mu           sync.RWMutex
	closed       bool
}

// NewThrottledWriter создает новый throttled writer
func NewThrottledWriter(file *os.File, maxSpeedMBps float64) *ThrottledWriter {
	return &ThrottledWriter{
		file:         file,
		maxSpeedMBps: maxSpeedMBps,
		lastWrite:    time.Now(),
		closed:       false,
	}
}

// Write записывает данные с ограничением скорости (thread-safe)
func (tw *ThrottledWriter) Write(data []byte) (int, error) {
	if tw.closed {
		return 0, io.ErrClosedPipe
	}

	if len(data) == 0 {
		return 0, nil
	}

	tw.mu.Lock()
	defer tw.mu.Unlock()

	now := time.Now()
	if tw.maxSpeedMBps > 0 {
		bytesPerSec := tw.maxSpeedMBps * 1024 * 1024
		if bytesPerSec > 0 {
			expected := time.Duration(float64(len(data)) / bytesPerSec * float64(time.Second))
			actual := now.Sub(tw.lastWrite)
			if actual < expected {
				time.Sleep(expected - actual)
			}
		}
	}

	n, err := tw.file.Write(data)
	tw.lastWrite = time.Now()
	return n, err
}

// Sync синхронизирует данные на диск
func (tw *ThrottledWriter) Sync() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.closed {
		return io.ErrClosedPipe
	}

	return tw.file.Sync()
}

// Close закрывает файл
func (tw *ThrottledWriter) Close() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.closed {
		return nil
	}

	tw.closed = true
	return tw.file.Close()
}
