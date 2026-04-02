package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"artemis/internal/mission"
	"artemis/internal/nasablog"
)

type notificationCall struct {
	title string
	body  string
}

func testNotifier(calls *[]notificationCall) notificationSender {
	return func(title, body string) tea.Cmd {
		*calls = append(*calls, notificationCall{title: title, body: body})
		return func() tea.Msg { return notificationResultMsg{} }
	}
}

func TestHandleBlogNotificationPrimesWithoutNotifying(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notificationsEnabled: true,
		notifier:             testNotifier(&calls),
	}

	cmd := m.handleBlogNotification(&nasablog.Status{
		Entries: []nasablog.Entry{{ID: 101, Title: "Flight day 1"}},
	})

	if cmd != nil {
		t.Fatalf("expected no command during initial blog prime")
	}
	if !m.blogPrimed || m.lastSeenBlogID != 101 {
		t.Fatalf("expected blog baseline to be primed, got primed=%v id=%d", m.blogPrimed, m.lastSeenBlogID)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no notifications during initial blog prime, got %d", len(calls))
	}
}

func TestHandleBlogNotificationEmitsOnlyForNewTopEntry(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notificationsEnabled: true,
		blogPrimed:           true,
		lastSeenBlogID:       101,
		notifier:             testNotifier(&calls),
	}

	if cmd := m.handleBlogNotification(&nasablog.Status{
		Entries: []nasablog.Entry{{ID: 101, Title: "Existing update"}},
	}); cmd != nil {
		t.Fatalf("expected no command when top blog ID is unchanged")
	}

	cmd := m.handleBlogNotification(&nasablog.Status{
		Entries: []nasablog.Entry{{ID: 102, Title: "Crew checks complete"}},
	})
	if cmd == nil {
		t.Fatalf("expected command when top blog ID changes")
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 notification call, got %d", len(calls))
	}
	if calls[0].title != "New Mission Log Entry" || calls[0].body != "Crew checks complete" {
		t.Fatalf("unexpected notification payload: %+v", calls[0])
	}
	if m.lastSeenBlogID != 102 {
		t.Fatalf("expected lastSeenBlogID to advance, got %d", m.lastSeenBlogID)
	}
}

func TestHandleBlogNotificationRespectsDisabledState(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notificationsEnabled: false,
		blogPrimed:           true,
		lastSeenBlogID:       101,
		notifier:             testNotifier(&calls),
	}

	cmd := m.handleBlogNotification(&nasablog.Status{
		Entries: []nasablog.Entry{{ID: 102, Title: "Hidden update"}},
	})

	if cmd != nil {
		t.Fatalf("expected no command when notifications are disabled")
	}
	if len(calls) != 0 {
		t.Fatalf("expected disabled notifications to suppress notifier calls")
	}
	if m.lastSeenBlogID != 102 {
		t.Fatalf("expected baseline to advance while disabled, got %d", m.lastSeenBlogID)
	}
}

func TestHandlePhaseNotificationPrimesWithoutNotifying(t *testing.T) {
	var calls []notificationCall
	now := mission.LaunchTime.Add(2 * time.Hour)
	m := Model{
		notificationsEnabled: true,
		notifier:             testNotifier(&calls),
	}

	cmd := m.handlePhaseNotification(now)

	if cmd != nil {
		t.Fatalf("expected no command during initial phase prime")
	}
	if !m.phasePrimed || m.lastPhaseIndex != mission.CurrentPhase(now.Sub(mission.LaunchTime)) {
		t.Fatalf("expected phase baseline to be primed, got primed=%v index=%d", m.phasePrimed, m.lastPhaseIndex)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no notifications during initial phase prime, got %d", len(calls))
	}
}

func TestHandlePhaseNotificationEmitsOnlyOnPhaseAdvance(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notificationsEnabled: true,
		phasePrimed:          true,
		lastPhaseIndex:       0,
		notifier:             testNotifier(&calls),
	}

	if cmd := m.handlePhaseNotification(mission.LaunchTime.Add(12 * time.Hour)); cmd != nil {
		t.Fatalf("expected no command while remaining in the same phase")
	}

	cmd := m.handlePhaseNotification(mission.LaunchTime.Add(mission.Phases[1].StartMET + time.Minute))
	if cmd == nil {
		t.Fatalf("expected command when phase advances")
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 notification call, got %d", len(calls))
	}
	if calls[0].title != "Mission Phase Change" || calls[0].body != mission.Phases[1].Name {
		t.Fatalf("unexpected phase notification payload: %+v", calls[0])
	}
	if m.lastPhaseIndex != 1 {
		t.Fatalf("expected lastPhaseIndex to advance, got %d", m.lastPhaseIndex)
	}
}

func TestHandlePhaseNotificationRespectsDisabledState(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notificationsEnabled: false,
		phasePrimed:          true,
		lastPhaseIndex:       0,
		notifier:             testNotifier(&calls),
	}

	cmd := m.handlePhaseNotification(mission.LaunchTime.Add(mission.Phases[1].StartMET + time.Minute))

	if cmd != nil {
		t.Fatalf("expected no command when notifications are disabled")
	}
	if len(calls) != 0 {
		t.Fatalf("expected disabled phase notifications to suppress notifier calls")
	}
	if m.lastPhaseIndex != 1 {
		t.Fatalf("expected phase baseline to advance while disabled, got %d", m.lastPhaseIndex)
	}
}

func TestDebugPhaseNotificationCmdTargetsNextPhase(t *testing.T) {
	var calls []notificationCall
	m := Model{
		notifier: testNotifier(&calls),
	}

	cmd := m.debugPhaseNotificationCmd()
	if cmd == nil {
		t.Fatalf("expected debug phase notification command")
	}
	if len(calls) != 1 {
		t.Fatalf("expected debug notifier call, got %d", len(calls))
	}

	wantIdx := mission.CurrentPhase(mission.MET())
	if wantIdx < len(mission.Phases)-1 {
		wantIdx++
	}
	if calls[0].title != "Mission Phase Change" || calls[0].body != mission.Phases[wantIdx].Name {
		t.Fatalf("unexpected debug notification payload: %+v", calls[0])
	}
}

func TestUpdateIgnoresDebugKeyWhenDisabled(t *testing.T) {
	var calls []notificationCall
	model, cmd := Model{
		debugKeysEnabled: false,
		notifier:         testNotifier(&calls),
	}.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	if cmd != nil {
		t.Fatalf("expected no command when debug keybindings are disabled")
	}
	got := model.(Model)
	if got.debugKeysEnabled {
		t.Fatalf("expected debug mode to remain disabled")
	}
	if len(calls) != 0 {
		t.Fatalf("expected no notifications when debug keybindings are disabled")
	}
}

func TestUpdateFiresDebugNotificationWhenEnabled(t *testing.T) {
	var calls []notificationCall
	_, cmd := Model{
		debugKeysEnabled: true,
		notifier:         testNotifier(&calls),
	}.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	if cmd == nil {
		t.Fatalf("expected debug keybinding to return a command")
	}
	if len(calls) != 1 {
		t.Fatalf("expected debug keybinding to invoke notifier, got %d calls", len(calls))
	}
}

func TestUpdateStoresTransientNotificationFailure(t *testing.T) {
	model, cmd := Model{}.Update(notificationResultMsg{err: errors.New("notify backend unavailable")})

	if cmd != nil {
		t.Fatalf("expected no follow-up command for notification result")
	}
	got := model.(Model)
	if got.notificationError != "notify failed" {
		t.Fatalf("expected notification error banner, got %q", got.notificationError)
	}
	if got.notificationErrorAt.IsZero() {
		t.Fatalf("expected notification error timestamp to be set")
	}
}

func TestFooterNotificationErrorExpires(t *testing.T) {
	fresh := Model{
		notificationError:   "notify failed",
		notificationErrorAt: time.Now(),
	}
	if fresh.footerNotificationError() != "notify failed" {
		t.Fatalf("expected recent notification error to be visible")
	}

	expired := Model{
		notificationError:   "notify failed",
		notificationErrorAt: time.Now().Add(-6 * time.Second),
	}
	if expired.footerNotificationError() != "" {
		t.Fatalf("expected stale notification error to be hidden")
	}
}

func TestNotificationCommandSelectsSupportedPlatforms(t *testing.T) {
	darwin := notificationCommand("darwin", "Mission Update", "Crew \"ready\" now")
	if darwin == nil || darwin.Path == "" || !strings.Contains(darwin.Path, "osascript") {
		t.Fatalf("expected darwin command to use osascript, got %#v", darwin)
	}
	if len(darwin.Args) < 3 || !strings.Contains(darwin.Args[2], `display notification "Crew \"ready\" now" with title "Mission Update"`) {
		t.Fatalf("unexpected darwin args: %#v", darwin.Args)
	}

	linux := notificationCommand("linux", "Mission Update", "Crew ready")
	if linux == nil || linux.Path == "" || !strings.Contains(linux.Path, "notify-send") {
		t.Fatalf("expected linux command to use notify-send, got %#v", linux)
	}
	if len(linux.Args) != 3 || linux.Args[1] != "Mission Update" || linux.Args[2] != "Crew ready" {
		t.Fatalf("unexpected linux args: %#v", linux.Args)
	}

	if unsupported := notificationCommand("windows", "Mission Update", "Crew ready"); unsupported != nil {
		t.Fatalf("expected unsupported OS to return nil, got %#v", unsupported)
	}
}
