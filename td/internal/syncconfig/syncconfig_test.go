package syncconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotThresholdDefault(t *testing.T) {
	// Clear any env var that might be set
	os.Unsetenv("TD_SYNC_SNAPSHOT_THRESHOLD")

	threshold := GetSnapshotThreshold()
	if threshold != 100 {
		t.Fatalf("default threshold: got %d, want 100", threshold)
	}
}

func TestSnapshotThresholdEnvVar(t *testing.T) {
	t.Setenv("TD_SYNC_SNAPSHOT_THRESHOLD", "500")

	threshold := GetSnapshotThreshold()
	if threshold != 500 {
		t.Fatalf("env threshold: got %d, want 500", threshold)
	}
}

func TestSnapshotThresholdEnvVarInvalid(t *testing.T) {
	t.Setenv("TD_SYNC_SNAPSHOT_THRESHOLD", "not-a-number")

	// Invalid env should fall through to default
	threshold := GetSnapshotThreshold()
	if threshold != 100 {
		t.Fatalf("invalid env threshold: got %d, want 100 (default)", threshold)
	}
}

func TestSnapshotThresholdEnvVarZero(t *testing.T) {
	t.Setenv("TD_SYNC_SNAPSHOT_THRESHOLD", "0")

	// Zero is valid: means snapshot bootstrap is disabled
	threshold := GetSnapshotThreshold()
	if threshold != 0 {
		t.Fatalf("zero env threshold: got %d, want 0 (disabled)", threshold)
	}
}

func TestSnapshotThresholdEnvVarNegative(t *testing.T) {
	t.Setenv("TD_SYNC_SNAPSHOT_THRESHOLD", "-5")

	// Negative should fall through to default
	threshold := GetSnapshotThreshold()
	if threshold != 100 {
		t.Fatalf("negative env threshold: got %d, want 100 (default)", threshold)
	}
}

func TestSnapshotThresholdEnvOverridesConfig(t *testing.T) {
	// Even if config has a value, env should take precedence
	t.Setenv("TD_SYNC_SNAPSHOT_THRESHOLD", "42")

	threshold := GetSnapshotThreshold()
	if threshold != 42 {
		t.Fatalf("env override: got %d, want 42", threshold)
	}
}

// setTestHome points the user home directory at a fresh temp dir.
// os.UserHomeDir reads HOME on Unix and USERPROFILE on Windows, so both are
// set to keep the tests hermetic on every platform.
func setTestHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	return tmpDir
}

// writeTestConfig creates a temp HOME with ~/.config/td/config.json and returns cleanup.
func writeTestConfig(t *testing.T, cfg *Config) {
	t.Helper()
	tmpDir := setTestHome(t)
	dir := filepath.Join(tmpDir, ".config", "td")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func boolPtr(b bool) *bool { return &b }

// TestSaveLoadAuthRoundTrip verifies SaveAuth/LoadAuth round-trip the API key
// and that on Windows the key is DPAPI-protected at rest (never plaintext on
// disk).
func TestSaveLoadAuthRoundTrip(t *testing.T) {
	tmpDir := setTestHome(t)

	const secret = "td_key_sk-secret-roundtrip-12345"
	in := &AuthCredentials{
		APIKey:    secret,
		UserID:    "user_1",
		Email:     "a@b.test",
		ServerURL: "http://localhost:8080",
		DeviceID:  "dev_1",
		ExpiresAt: "2027-01-01T00:00:00Z",
	}
	if err := SaveAuth(in); err != nil {
		t.Fatalf("SaveAuth: %v", err)
	}
	// SaveAuth must not mutate the caller's struct.
	if in.APIKey != secret {
		t.Errorf("SaveAuth mutated caller APIKey to %q", in.APIKey)
	}

	raw, err := os.ReadFile(filepath.Join(tmpDir, ".config", "td", "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if dpapiAvailable {
		if strings.Contains(string(raw), secret) {
			t.Error("auth.json contains the plaintext API key; expected DPAPI ciphertext")
		}
		if !strings.Contains(string(raw), "api_key_protected") {
			t.Error("auth.json missing api_key_protected field on Windows")
		}
	} else {
		if !strings.Contains(string(raw), secret) {
			t.Error("auth.json should contain the plaintext key on non-Windows platforms")
		}
	}

	out, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth: %v", err)
	}
	if out == nil {
		t.Fatal("LoadAuth returned nil credentials")
	}
	if out.APIKey != secret {
		t.Errorf("APIKey = %q, want %q", out.APIKey, secret)
	}
	if out.UserID != in.UserID || out.Email != in.Email || out.ServerURL != in.ServerURL ||
		out.DeviceID != in.DeviceID || out.ExpiresAt != in.ExpiresAt {
		t.Errorf("credentials did not round-trip: %+v", out)
	}
}

// TestLoadAuthPlaintextBackcompat verifies pre-DPAPI auth.json files with a
// plaintext api_key still load on every platform.
func TestLoadAuthPlaintextBackcompat(t *testing.T) {
	tmpDir := setTestHome(t)
	dir := filepath.Join(tmpDir, ".config", "td")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	legacy := `{"api_key":"legacy-plain-key","user_id":"u2","email":"c@d.test","server_url":"http://localhost:8080","device_id":"dev_2","expires_at":""}`
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(legacy), 0600); err != nil {
		t.Fatalf("write auth.json: %v", err)
	}

	out, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth: %v", err)
	}
	if out.APIKey != "legacy-plain-key" {
		t.Errorf("APIKey = %q, want legacy-plain-key", out.APIKey)
	}
}

func TestAutoSyncEnabledFromConfig(t *testing.T) {
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{Enabled: boolPtr(false)}}})
	t.Setenv("TD_SYNC_AUTO", "")
	if GetAutoSyncEnabled() {
		t.Error("expected auto-sync disabled from config")
	}
}

func TestAutoSyncOnStartFromConfig(t *testing.T) {
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{OnStart: boolPtr(false)}}})
	t.Setenv("TD_SYNC_AUTO_START", "")
	if GetAutoSyncOnStart() {
		t.Error("expected on_start disabled from config")
	}
}

func TestAutoSyncDebounceFromConfig(t *testing.T) {
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{Debounce: "10s"}}})
	t.Setenv("TD_SYNC_AUTO_DEBOUNCE", "")
	if d := GetAutoSyncDebounce(); d != 10*time.Second {
		t.Errorf("expected 10s from config, got %v", d)
	}
}

func TestAutoSyncIntervalFromConfig(t *testing.T) {
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{Interval: "15m"}}})
	t.Setenv("TD_SYNC_AUTO_INTERVAL", "")
	if d := GetAutoSyncInterval(); d != 15*time.Minute {
		t.Errorf("expected 15m from config, got %v", d)
	}
}

func TestAutoSyncPullFromConfig(t *testing.T) {
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{Pull: boolPtr(false)}}})
	t.Setenv("TD_SYNC_AUTO_PULL", "")
	if GetAutoSyncPull() {
		t.Error("expected pull disabled from config")
	}
}

func TestAutoSyncEnvOverridesConfig(t *testing.T) {
	// Config says disabled, env says enabled — env should win
	writeTestConfig(t, &Config{Sync: SyncConfig{Auto: AutoSyncConfig{
		Enabled:  boolPtr(false),
		OnStart:  boolPtr(false),
		Debounce: "10s",
		Interval: "15m",
		Pull:     boolPtr(false),
	}}})

	t.Setenv("TD_SYNC_AUTO", "true")
	if !GetAutoSyncEnabled() {
		t.Error("env should override config for enabled")
	}

	t.Setenv("TD_SYNC_AUTO_START", "1")
	if !GetAutoSyncOnStart() {
		t.Error("env should override config for on_start")
	}

	t.Setenv("TD_SYNC_AUTO_DEBOUNCE", "500ms")
	if d := GetAutoSyncDebounce(); d != 500*time.Millisecond {
		t.Errorf("env should override config for debounce, got %v", d)
	}

	t.Setenv("TD_SYNC_AUTO_INTERVAL", "30s")
	if d := GetAutoSyncInterval(); d != 30*time.Second {
		t.Errorf("env should override config for interval, got %v", d)
	}

	t.Setenv("TD_SYNC_AUTO_PULL", "true")
	if !GetAutoSyncPull() {
		t.Error("env should override config for pull")
	}
}
