package workspace

import (
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/marcus/sidecar/internal/app"
)

// openInBrowser opens the URL in the default browser.
// Only http(s) URLs are launched: the URL can derive from untrusted input
// (e.g. remote/PR metadata), so it must never reach a shell or a non-URL
// protocol handler.
func openInBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			return nil
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			// Never use `cmd /c start <url>` here: cmd.exe re-parses the
			// argument, so metacharacters like "&" in the URL become command
			// injection. rundll32 receives the URL as a plain argv element.
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			return nil
		}
		_ = cmd.Start()
		return nil
	}
}

// openInGitTab opens the selected worktree in the git status tab.
// It switches to the worktree directory and focuses the git-status plugin.
func (p *Plugin) openInGitTab(wt *Worktree) tea.Cmd {
	if wt == nil {
		return nil
	}
	// Sequence: switch to worktree first (triggers plugin reinit), then focus git-status plugin.
	// Must use Sequence not Batch to avoid deadlock during concurrent plugin reinit + fork/exec.
	return tea.Sequence(
		app.SwitchWorktree(wt.Path),
		app.FocusPlugin("git-status"),
	)
}
