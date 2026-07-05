// Package fsutil provides small filesystem helpers shared across td.
package fsutil

import (
	"os"
	"runtime"
	"time"
)

// Rename renames oldpath to newpath like os.Rename. On Windows the rename is
// retried with a short bounded backoff: replacing a file that another process
// (antivirus scanner, search indexer, concurrent td) briefly holds open fails
// with a transient sharing violation, which would otherwise orphan the temp
// file during atomic-write (temp file + rename) sequences.
func Rename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil || runtime.GOOS != "windows" {
		return err
	}

	// Bounded retry: 5+10+20+40+80ms ≈ 155ms worst case.
	delay := 5 * time.Millisecond
	for i := 0; i < 5; i++ {
		time.Sleep(delay)
		if err = os.Rename(oldpath, newpath); err == nil {
			return nil
		}
		delay *= 2
	}
	return err
}
