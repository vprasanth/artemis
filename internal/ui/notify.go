package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func nativeNotifyCmd(title, body string) tea.Cmd {
	return func() tea.Msg {
		cmd := notificationCommand(runtime.GOOS, title, body)
		if cmd == nil {
			return notificationResultMsg{}
		}
		return notificationResultMsg{err: cmd.Run()}
	}
}

func notificationCommand(goos, title, body string) *exec.Cmd {
	title = compactNotificationText(title, 80)
	body = compactNotificationText(body, 160)

	switch goos {
	case "darwin":
		script := fmt.Sprintf(
			"display notification %s with title %s",
			quoteAppleScript(body),
			quoteAppleScript(title),
		)
		return exec.Command("osascript", "-e", script)
	case "linux":
		return exec.Command("notify-send", title, body)
	default:
		return nil
	}
}

func compactNotificationText(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if limit <= 0 || len(s) <= limit {
		return s
	}
	if limit == 1 {
		return "…"
	}
	return s[:limit-1] + "…"
}

func quoteAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
