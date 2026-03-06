package lsp

import (
	"bytes"
	"strings"
	"testing"
)

func TestTraceLoggerFileEvent(t *testing.T) {
	var buf bytes.Buffer
	tl := newTraceLogger(&buf)

	before := memSnapshot{
		HeapAlloc: 100 * 1024 * 1024, // 100 MB
		HeapInuse: 120 * 1024 * 1024,
		HeapSys:   200 * 1024 * 1024,
	}
	after := memSnapshot{
		HeapAlloc: 105 * 1024 * 1024, // 105 MB
		HeapInuse: 125 * 1024 * 1024,
		HeapSys:   200 * 1024 * 1024,
	}

	tl.LogFileEvent("OPEN", "file:///test/foo.ts", before, after)

	output := buf.String()
	if !strings.Contains(output, "FILE OPEN: file:///test/foo.ts") {
		t.Errorf("expected file open event in output, got:\n%s", output)
	}
	if !strings.Contains(output, "HeapAlloc:") {
		t.Errorf("expected HeapAlloc in output, got:\n%s", output)
	}
	if !strings.Contains(output, "delta:") {
		t.Errorf("expected delta in output, got:\n%s", output)
	}
}

func TestTraceLoggerLargeIncrease(t *testing.T) {
	var buf bytes.Buffer
	tl := newTraceLogger(&buf)

	before := memSnapshot{
		HeapAlloc: 100 * 1024 * 1024, // 100 MB
		HeapInuse: 120 * 1024 * 1024,
		HeapSys:   200 * 1024 * 1024,
	}
	after := memSnapshot{
		HeapAlloc: 200 * 1024 * 1024, // 200 MB (+100MB)
		HeapInuse: 220 * 1024 * 1024,
		HeapSys:   300 * 1024 * 1024,
	}

	tl.LogFileEvent("OPEN", "file:///test/large.ts", before, after)

	output := buf.String()
	if !strings.Contains(output, "LARGE MEMORY INCREASE") {
		t.Errorf("expected LARGE MEMORY INCREASE warning in output, got:\n%s", output)
	}
}

func TestTraceLoggerRequest(t *testing.T) {
	var buf bytes.Buffer
	tl := newTraceLogger(&buf)

	before := memSnapshot{HeapAlloc: 50 * 1024 * 1024}
	after := memSnapshot{HeapAlloc: 51 * 1024 * 1024}

	tl.LogRequest("textDocument/completion", " (1)", 42000000, before, after) // 42ms

	output := buf.String()
	if !strings.Contains(output, "REQUEST textDocument/completion") {
		t.Errorf("expected request log in output, got:\n%s", output)
	}
}

func TestTraceLoggerProjectInfo(t *testing.T) {
	var buf bytes.Buffer
	tl := newTraceLogger(&buf)

	tl.LogProjectInfo("/home/user/project/tsconfig.json", 42)

	output := buf.String()
	if !strings.Contains(output, "PROJECT LOADED: /home/user/project/tsconfig.json (42 files)") {
		t.Errorf("expected project info in output, got:\n%s", output)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1536 * 1024, "1.50 MB"},
	}
	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatBytesDelta(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{1024 * 1024, "+1.00 MB"},
		{-1024 * 1024, "-1.00 MB"},
		{0, "+0 B"},
	}
	for _, tt := range tests {
		result := formatBytesDelta(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytesDelta(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
