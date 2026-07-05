package cache

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScannerPool(t *testing.T) {
	// Get buffer
	buf := GetScannerBuffer()
	if len(buf) != DefaultScannerBufSize {
		t.Errorf("expected buffer size %d, got %d", DefaultScannerBufSize, len(buf))
	}

	// Put it back
	PutScannerBuffer(buf)

	// Get again (should reuse)
	buf2 := GetScannerBuffer()
	if len(buf2) != DefaultScannerBufSize {
		t.Errorf("expected buffer size %d, got %d", DefaultScannerBufSize, len(buf2))
	}
	PutScannerBuffer(buf2)
}

func TestNewScanner(t *testing.T) {
	content := "line1\nline2\nline3\n"
	reader := strings.NewReader(content)

	scanner, buf := NewScanner(reader)
	defer PutScannerBuffer(buf)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestIncrementalReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Read from start
	r, err := NewIncrementalReader(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line1" {
		t.Errorf("expected line1, got %s", line)
	}

	// Check offset tracking
	if r.Offset() != 6 { // "line1" + newline
		t.Errorf("expected offset 6, got %d", r.Offset())
	}

	line, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line2" {
		t.Errorf("expected line2, got %s", line)
	}
}

func TestIncrementalReader_FromOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Read from offset 6 (after "line1\n")
	r, err := NewIncrementalReader(path, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line2" {
		t.Errorf("expected line2, got %s", line)
	}
}

func TestIncrementalReader_EOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	if err := os.WriteFile(path, []byte("line1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewIncrementalReader(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	_, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestTailReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Read last 12 bytes (covers "line4\nline5\n")
	r, err := NewTailReader(path, 12)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	// First line should skip the partial line at seek point
	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	// Should get "line5" (skipped partial "line4")
	if string(line) != "line5" {
		t.Errorf("expected line5, got %s", line)
	}

	// EOF
	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestTailReader_SmallFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Request more than file size
	r, err := NewTailReader(path, 1000)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	// Should read from start, no skip
	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line1" {
		t.Errorf("expected line1, got %s", line)
	}

	line, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line2" {
		t.Errorf("expected line2, got %s", line)
	}
}

func TestHeadReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Read only first 3 lines
	r, err := NewHeadReader(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	var lines []string
	for {
		line, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(line))
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}

	if r.LinesRead() != 3 {
		t.Errorf("expected LinesRead() = 3, got %d", r.LinesRead())
	}
}

func TestHeadReader_Offset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewHeadReader(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	_, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if r.Offset() != 6 { // "line1" + newline
		t.Errorf("expected offset 6, got %d", r.Offset())
	}

	_, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if r.Offset() != 12 { // both lines
		t.Errorf("expected offset 12, got %d", r.Offset())
	}
}

func TestIncrementalReader_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// CRLF line endings: each terminator is 2 bytes. Offsets computed as
	// len(line)+1 would drift 1 byte per line and resume mid-line.
	content := "line1\r\nline2\r\nline3\r\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewIncrementalReader(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line1" {
		t.Errorf("expected line1 (\\r stripped), got %q", line)
	}
	if r.Offset() != 7 { // "line1" + CRLF = 7 bytes
		t.Errorf("expected offset 7 after CRLF line, got %d", r.Offset())
	}

	line, err = r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line2" {
		t.Errorf("expected line2, got %q", line)
	}
	resumeOffset := r.Offset()
	if resumeOffset != 14 {
		t.Errorf("expected offset 14 after two CRLF lines, got %d", resumeOffset)
	}

	// Resume from the recorded offset: must land exactly on line3.
	r2, err := NewIncrementalReader(path, resumeOffset)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r2.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	line, err = r2.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line3" {
		t.Errorf("resume at offset %d: expected line3, got %q", resumeOffset, line)
	}
	if r2.Offset() != int64(len(content)) {
		t.Errorf("expected final offset %d, got %d", len(content), r2.Offset())
	}
	if _, err := r2.Next(); err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestIncrementalReader_CRLFAppendResume(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	initial := "one\r\ntwo\r\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// First pass: consume everything, record the offset (as the msg caches do).
	r, err := NewIncrementalReader(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := r.Next(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
	}
	offset := r.Offset()
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if offset != int64(len(initial)) {
		t.Fatalf("expected offset %d after full read, got %d", len(initial), offset)
	}

	// Append new CRLF lines, then resume from the recorded offset.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("three\r\nfour\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	r2, err := NewIncrementalReader(path, offset)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r2.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	var lines []string
	for {
		line, err := r2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(line))
	}
	if len(lines) != 2 || lines[0] != "three" || lines[1] != "four" {
		t.Errorf("resume after append: expected [three four], got %v", lines)
	}
}

func TestIncrementalReader_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewIncrementalReader(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	if _, err := r.Next(); err != nil {
		t.Fatal(err)
	}
	line, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != "line2" {
		t.Errorf("expected line2, got %q", line)
	}
	// Offset must not overshoot EOF when the final line has no terminator.
	if r.Offset() != int64(len(content)) {
		t.Errorf("expected offset %d (EOF), got %d", len(content), r.Offset())
	}
}

func TestHeadReader_CRLFOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\r\nline2\r\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewHeadReader(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	if _, err := r.Next(); err != nil {
		t.Fatal(err)
	}
	if r.Offset() != 7 { // "line1" + CRLF
		t.Errorf("expected offset 7, got %d", r.Offset())
	}
	if _, err := r.Next(); err != nil {
		t.Fatal(err)
	}
	if r.Offset() != 14 {
		t.Errorf("expected offset 14, got %d", r.Offset())
	}
}

func TestHeadReader_FewerLinesThanMax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := "line1\nline2\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Request more lines than exist
	r, err := NewHeadReader(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("failed to close reader: %v", err)
		}
	}()

	var count int
	for {
		_, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 lines, got %d", count)
	}
}
