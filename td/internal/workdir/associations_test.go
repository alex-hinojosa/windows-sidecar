package workdir

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setTestHome points the user home at a fresh temp dir on every platform.
// os.UserHomeDir reads HOME on Unix and USERPROFILE on Windows.
func setTestHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	return tmp
}

func TestLoadAssociations_MissingFile(t *testing.T) {
	// Point the home dir at a temp dir so ConfigDir returns an empty config
	setTestHome(t)

	assoc, err := LoadAssociations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assoc) != 0 {
		t.Fatalf("expected empty map, got %v", assoc)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	tmp := setTestHome(t)

	input := map[string]string{
		filepath.Join(tmp, "code", "repo-one"): filepath.Join(tmp, "notes", "vault-one"),
		filepath.Join(tmp, "code", "repo-two"): filepath.Join(tmp, "notes", "vault-two"),
	}

	if err := SaveAssociations(input); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadAssociations()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded) != len(input) {
		t.Fatalf("expected %d entries, got %d", len(input), len(loaded))
	}
	for k, v := range input {
		if loaded[k] != v {
			t.Errorf("key %s: expected %s, got %s", k, v, loaded[k])
		}
	}
}

// writeAssociations writes an associations.json under the test home dir.
func writeAssociations(t *testing.T, home string, assoc map[string]string) {
	t.Helper()
	configDir := filepath.Join(home, ".config", "td")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(assoc)
	if err := os.WriteFile(filepath.Join(configDir, associationsFile), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLookupAssociation(t *testing.T) {
	tmp := setTestHome(t)

	repoDir := filepath.Join(tmp, "code", "myrepo")
	targetDir := filepath.Join(tmp, "projects", "myproject")
	writeAssociations(t, tmp, map[string]string{repoDir: targetDir})

	// Found
	target, ok := LookupAssociation(repoDir)
	if !ok {
		t.Fatal("expected association to be found")
	}
	if target != targetDir {
		t.Errorf("expected %s, got %s", targetDir, target)
	}

	// Not found
	_, ok = LookupAssociation(filepath.Join(tmp, "code", "other"))
	if ok {
		t.Fatal("expected no association")
	}
}

// TestLookupAssociation_CaseInsensitiveOnWindows verifies that a lookup with
// different path casing (cmd reports `c:\proj`, PowerShell `C:\proj`) still
// resolves on Windows and still misses on case-sensitive Unix.
func TestLookupAssociation_CaseInsensitiveOnWindows(t *testing.T) {
	tmp := setTestHome(t)

	repoDir := filepath.Join(tmp, "Code", "MyRepo")
	targetDir := filepath.Join(tmp, "projects", "myproject")
	writeAssociations(t, tmp, map[string]string{repoDir: targetDir})

	swapped := strings.ToLower(repoDir)
	if swapped == repoDir {
		t.Skip("temp dir path has no upper-case characters to fold")
	}

	target, ok := LookupAssociation(swapped)
	if runtime.GOOS == "windows" {
		if !ok {
			t.Fatalf("expected case-insensitive association hit for %s", swapped)
		}
		if target != targetDir {
			t.Errorf("expected %s, got %s", targetDir, target)
		}
	} else {
		if ok {
			t.Fatalf("expected case-sensitive miss on %s for %s", runtime.GOOS, swapped)
		}
	}
}

func TestResolveBaseDir_TdRootPriorityOverAssociation(t *testing.T) {
	tmp := setTestHome(t)

	// Set up a directory with both .td-root and an association
	projectDir := filepath.Join(tmp, "project")
	tdRootTarget := filepath.Join(tmp, "tdroot-target")
	assocTarget := filepath.Join(tmp, "assoc-target")

	for _, d := range []string{projectDir, tdRootTarget, assocTarget} {
		if err := os.MkdirAll(filepath.Join(d, ".todos"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Write .td-root pointing to tdRootTarget
	if err := os.WriteFile(filepath.Join(projectDir, ".td-root"), []byte(tdRootTarget), 0644); err != nil {
		t.Fatal(err)
	}

	// Write association pointing to assocTarget
	configDir := filepath.Join(tmp, ".config", "td")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	assoc := map[string]string{projectDir: assocTarget}
	data, _ := json.Marshal(assoc)
	if err := os.WriteFile(filepath.Join(configDir, associationsFile), data, 0644); err != nil {
		t.Fatal(err)
	}

	// .td-root should win
	result := ResolveBaseDir(projectDir)
	if result != tdRootTarget {
		t.Errorf("expected .td-root target %s, got %s", tdRootTarget, result)
	}
}

func TestResolveBaseDir_AssociationUsed(t *testing.T) {
	tmp := setTestHome(t)

	// Set up a directory with only an association (no .td-root, no .todos)
	projectDir := filepath.Join(tmp, "project")
	assocTarget := filepath.Join(tmp, "assoc-target")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(assocTarget, ".todos"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write association
	configDir := filepath.Join(tmp, ".config", "td")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	assoc := map[string]string{projectDir: assocTarget}
	data, _ := json.Marshal(assoc)
	if err := os.WriteFile(filepath.Join(configDir, associationsFile), data, 0644); err != nil {
		t.Fatal(err)
	}

	result := ResolveBaseDir(projectDir)
	if result != assocTarget {
		t.Errorf("expected association target %s, got %s", assocTarget, result)
	}
}
