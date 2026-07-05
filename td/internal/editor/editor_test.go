package editor

import (
	"reflect"
	"runtime"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantName string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "bare editor",
			raw:      "vi",
			wantName: "vi",
			wantArgs: []string{},
		},
		{
			name:     "editor with flag",
			raw:      "code --wait",
			wantName: "code",
			wantArgs: []string{"--wait"},
		},
		{
			name:     "quoted windows path with spaces plus flag",
			raw:      `"C:\Program Files\Microsoft VS Code\bin\code.cmd" --wait`,
			wantName: `C:\Program Files\Microsoft VS Code\bin\code.cmd`,
			wantArgs: []string{"--wait"},
		},
		{
			name:     "quoted path only",
			raw:      `"C:\Program Files\Notepad++\notepad++.exe"`,
			wantName: `C:\Program Files\Notepad++\notepad++.exe`,
			wantArgs: []string{},
		},
		{
			name:     "multiple args",
			raw:      "emacsclient -c -a vim",
			wantName: "emacsclient",
			wantArgs: []string{"-c", "-a", "vim"},
		},
		{
			name:     "quoted argument",
			raw:      `myeditor --title "my file"`,
			wantName: "myeditor",
			wantArgs: []string{"--title", "my file"},
		},
		{
			name:     "extra whitespace",
			raw:      "  code   --wait  ",
			wantName: "code",
			wantArgs: []string{"--wait"},
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			raw:     "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, err := Parse(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error, got name=%q args=%v", tt.raw, name, args)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.raw, err)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			gotArgs := args
			if gotArgs == nil {
				gotArgs = []string{}
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestCommandAppendsPath(t *testing.T) {
	cmd, err := Command(`"C:\some dir\edit.exe" --wait`, "C:\\tmp\\note.md")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	wantArgs := []string{`C:\some dir\edit.exe`, "--wait", "C:\\tmp\\note.md"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestFallback(t *testing.T) {
	got := Fallback("vi")
	if runtime.GOOS == "windows" {
		if got != "notepad" {
			t.Errorf("Fallback = %q, want notepad on windows", got)
		}
	} else {
		if got != "vi" {
			t.Errorf("Fallback = %q, want vi on non-windows", got)
		}
	}
}
