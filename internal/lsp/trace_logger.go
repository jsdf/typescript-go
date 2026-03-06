package lsp

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

// traceLogger writes timestamped trace entries with memory usage information
// to a log file, enabling debugging of which files increase memory usage and why.
type traceLogger struct {
	mu       sync.Mutex
	w        io.Writer
	lastHeap uint64
}

func newTraceLogger(w io.Writer) *traceLogger {
	tl := &traceLogger{w: w}
	// Capture initial memory baseline
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	tl.lastHeap = m.HeapAlloc
	tl.writeHeader()
	return tl
}

func (tl *traceLogger) writeHeader() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(tl.w, "=== tsgo LSP Trace Log ===\n")
	fmt.Fprintf(tl.w, "Started: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(tl.w, "Initial HeapAlloc: %s\n\n", formatBytes(m.HeapAlloc))
}

// memSnapshot captures a point-in-time memory reading.
type memSnapshot struct {
	HeapAlloc  uint64
	HeapInuse  uint64
	HeapSys    uint64
	TotalAlloc uint64
	NumGC      uint32
}

func captureMemSnapshot() memSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memSnapshot{
		HeapAlloc:  m.HeapAlloc,
		HeapInuse:  m.HeapInuse,
		HeapSys:    m.HeapSys,
		TotalAlloc: m.TotalAlloc,
		NumGC:      m.NumGC,
	}
}

// LogFileEvent logs a file-related LSP event (open, change, close, save) with
// memory usage before and after, highlighting significant increases.
func (tl *traceLogger) LogFileEvent(event string, uri string, before, after memSnapshot) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	delta := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	totalDelta := int64(after.HeapAlloc) - int64(tl.lastHeap)

	ts := time.Now().Format("15:04:05.000")

	fmt.Fprintf(tl.w, "[%s] FILE %s: %s\n", ts, event, uri)
	fmt.Fprintf(tl.w, "  HeapAlloc: %s -> %s (delta: %s)\n",
		formatBytes(before.HeapAlloc), formatBytes(after.HeapAlloc), formatBytesDelta(delta))
	fmt.Fprintf(tl.w, "  HeapInuse: %s -> %s\n",
		formatBytes(before.HeapInuse), formatBytes(after.HeapInuse))
	fmt.Fprintf(tl.w, "  HeapSys: %s, TotalAlloc: %s, GC cycles: %d\n",
		formatBytes(after.HeapSys), formatBytes(after.TotalAlloc), after.NumGC)

	if totalDelta > 10*1024*1024 { // >10MB increase from last event
		fmt.Fprintf(tl.w, "  *** LARGE MEMORY INCREASE: %s since last event ***\n", formatBytesDelta(totalDelta))
	}

	fmt.Fprintln(tl.w)
	tl.lastHeap = after.HeapAlloc
}

// LogRequest logs an LSP request method with timing and memory usage.
func (tl *traceLogger) LogRequest(method string, id string, duration time.Duration, before, after memSnapshot) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	delta := int64(after.HeapAlloc) - int64(before.HeapAlloc)

	ts := time.Now().Format("15:04:05.000")

	fmt.Fprintf(tl.w, "[%s] REQUEST %s%s (%v)\n", ts, method, id, duration)
	fmt.Fprintf(tl.w, "  HeapAlloc: %s -> %s (delta: %s)\n",
		formatBytes(before.HeapAlloc), formatBytes(after.HeapAlloc), formatBytesDelta(delta))

	if delta > 10*1024*1024 { // >10MB increase
		fmt.Fprintf(tl.w, "  *** LARGE MEMORY INCREASE ***\n")
	}

	fmt.Fprintln(tl.w)
	tl.lastHeap = after.HeapAlloc
}

// LogProjectInfo logs information about projects loaded during a snapshot update.
func (tl *traceLogger) LogProjectInfo(projectName string, fileCount int) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(tl.w, "[%s] PROJECT LOADED: %s (%d files)\n", ts, projectName, fileCount)
}

// LogMessage logs a general trace message.
func (tl *traceLogger) LogMessage(format string, args ...any) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(tl.w, "[%s] %s\n", ts, fmt.Sprintf(format, args...))
}

func formatBytes(b uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatBytesDelta(d int64) string {
	prefix := "+"
	if d < 0 {
		prefix = "-"
		d = -d
	}
	return prefix + formatBytes(uint64(d))
}
