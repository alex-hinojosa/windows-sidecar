package db

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// ToRepoRelative converts an absolute file path to a repo-relative path
// using forward slashes for cross-platform stability.
// Returns an error if the path is outside the repo root.
//
// On Windows the containment check is case-insensitive: the filesystem is
// case-insensitive and different shells report different casing (`c:\proj`
// from cmd vs `C:\proj` from PowerShell), while filepath.Rel compares path
// elements case-sensitively and would report a false "outside repo root".
func ToRepoRelative(absPath, repoRoot string) (string, error) {
	// Clean both paths
	absPath = filepath.Clean(absPath)
	repoRoot = filepath.Clean(repoRoot)

	relFrom, relTo := repoRoot, absPath
	if runtime.GOOS == "windows" {
		// Fold ASCII case for the computation only. The fold is
		// length-preserving, so offsets into the folded paths are valid for
		// the originals.
		relFrom, relTo = foldPathASCII(repoRoot), foldPathASCII(absPath)
	}

	rel, err := filepath.Rel(relFrom, relTo)
	if err != nil {
		return "", fmt.Errorf("cannot compute relative path: %w", err)
	}

	// Check if the path escapes the repo root
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("file %s is outside repo root %s", absPath, repoRoot)
	}

	if runtime.GOOS == "windows" && rel != "." {
		// Re-derive the relative path from the original absPath so the
		// returned value keeps its original casing instead of the folded one.
		rel = strings.TrimLeft(absPath[len(repoRoot):], `\/`)
	}

	// Normalize to forward slashes for cross-platform consistency
	return filepath.ToSlash(rel), nil
}

// foldPathASCII lowercases ASCII letters only. Unlike strings.ToLower it is
// guaranteed length-preserving (non-ASCII runes pass through untouched),
// which ToRepoRelative relies on when mapping folded offsets back onto the
// original path.
func foldPathASCII(p string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, p)
}

// IsAbsolutePath returns true if the path looks like an absolute path
// (starts with / on Unix or a drive letter on Windows).
func IsAbsolutePath(p string) bool {
	if strings.HasPrefix(p, "/") {
		return true
	}
	// Windows drive letter: e.g. C:\, D:/
	if runtime.GOOS == "windows" && len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') {
		return true
	}
	return false
}

// NormalizeFilePathForID normalizes a file path to forward slashes
// for use in deterministic ID generation. This ensures the same ID
// is generated regardless of OS path separators.
func NormalizeFilePathForID(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}
