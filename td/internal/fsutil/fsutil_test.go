package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRenameBasic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Rename(src, dst); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "payload" {
		t.Errorf("dst content = %q, want payload", data)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("src should be gone, stat err = %v", err)
	}
}

func TestRenameMissingSourceFails(t *testing.T) {
	dir := t.TempDir()
	err := Rename(filepath.Join(dir, "nope.txt"), filepath.Join(dir, "dst.txt"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

// TestRenameRetriesWhileTargetHeldOpen exercises the Windows retry path: a
// rename over a target held open by another handle fails with a sharing
// violation until the handle closes. On Unix the first attempt succeeds.
func TestRenameRetriesWhileTargetHeldOpen(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		f.Close()
		close(release)
	}()

	err = Rename(src, dst)
	<-release
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Fatalf("Rename should have succeeded after target handle closed: %v", err)
		}
		t.Fatalf("Rename: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("dst content = %q, want new", data)
	}
}
