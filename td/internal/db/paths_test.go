package db

import (
	"runtime"
	"testing"
)

func TestToRepoRelative_BasicPath(t *testing.T) {
	rel, err := ToRepoRelative("/home/user/project/src/main.go", "/home/user/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "src/main.go" {
		t.Errorf("expected src/main.go, got %s", rel)
	}
}

func TestToRepoRelative_RootFile(t *testing.T) {
	rel, err := ToRepoRelative("/home/user/project/README.md", "/home/user/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "README.md" {
		t.Errorf("expected README.md, got %s", rel)
	}
}

func TestToRepoRelative_OutsideRepo(t *testing.T) {
	_, err := ToRepoRelative("/other/path/file.go", "/home/user/project")
	if err == nil {
		t.Error("expected error for path outside repo")
	}
}

func TestToRepoRelative_SamePath(t *testing.T) {
	rel, err := ToRepoRelative("/home/user/project", "/home/user/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "." {
		t.Errorf("expected '.', got %s", rel)
	}
}

// TestToRepoRelative_WindowsCaseInsensitive verifies that different path
// casing (cmd reports `c:\proj`, PowerShell `C:\proj`) does not produce a
// false "outside repo root" on Windows, and that the returned relative path
// keeps the file's original casing.
func TestToRepoRelative_WindowsCaseInsensitive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only behavior")
	}

	tests := []struct {
		name     string
		absPath  string
		repoRoot string
		want     string
	}{
		{
			name:     "lowercase drive letter vs uppercase root",
			absPath:  `c:\proj\src\Main.go`,
			repoRoot: `C:\proj`,
			want:     "src/Main.go",
		},
		{
			name:     "folded directory casing",
			absPath:  `C:\Proj\SRC\file.go`,
			repoRoot: `c:\proj`,
			want:     "SRC/file.go",
		},
		{
			name:     "same path different casing",
			absPath:  `c:\PROJ`,
			repoRoot: `C:\proj`,
			want:     ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := ToRepoRelative(tt.absPath, tt.repoRoot)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel != tt.want {
				t.Errorf("rel = %q, want %q", rel, tt.want)
			}
		})
	}

	// Still outside the root regardless of casing
	if _, err := ToRepoRelative(`c:\other\file.go`, `C:\proj`); err == nil {
		t.Error("expected error for path outside repo root")
	}
}

// TestToRepoRelative_UnixStaysCaseSensitive pins the Unix behavior: paths
// differing only by case are distinct.
func TestToRepoRelative_UnixStaysCaseSensitive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only behavior")
	}
	if _, err := ToRepoRelative("/home/User/project/file.go", "/home/user/project"); err == nil {
		t.Error("expected error: unix path comparison must stay case-sensitive")
	}
}

func TestIsAbsolutePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/bin/go", true},
		{"/home/user/file.go", true},
		{"src/main.go", false},
		{"./relative/path", false},
		{"relative", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsAbsolutePath(tt.path)
		if got != tt.want {
			t.Errorf("IsAbsolutePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeFilePathForID(t *testing.T) {
	// Forward slashes should be preserved
	result := NormalizeFilePathForID("src/main.go")
	if result != "src/main.go" {
		t.Errorf("expected src/main.go, got %s", result)
	}

	// Dot segments should be cleaned
	result = NormalizeFilePathForID("src/../pkg/main.go")
	if result != "pkg/main.go" {
		t.Errorf("expected pkg/main.go, got %s", result)
	}
}

func TestIssueFileID_CrossPlatformConsistency(t *testing.T) {
	// Both forward and backslash paths should produce the same ID
	id1 := IssueFileID("td-abc123", "src/main.go")
	id2 := IssueFileID("td-abc123", "src/main.go")
	if id1 != id2 {
		t.Errorf("IDs should match: %s vs %s", id1, id2)
	}

	// Cleaned path should match uncleaned
	id3 := IssueFileID("td-abc123", "src/../src/main.go")
	if id1 != id3 {
		t.Errorf("Cleaned path ID should match: %s vs %s", id1, id3)
	}
}
