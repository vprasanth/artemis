package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
	"artemis/internal/mission"
	"artemis/internal/nasablog"
	"artemis/internal/spaceweather"
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

func TestUpdateTogglesUnits(t *testing.T) {
	model, cmd := Model{units: unitMetric}.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Fatalf("expected no command when toggling units")
	}

	got := model.(Model)
	if got.units != unitImperial {
		t.Fatalf("expected units to toggle to imperial, got %v", got.units)
	}
}

func TestNewModelReadsScreenProtectionEnv(t *testing.T) {
	t.Setenv("ARTEMIS_SCREEN_PROTECT", "drift-idle")
	t.Setenv("ARTEMIS_SCREEN_PROTECT_DRIFT_INTERVAL", "90s")
	t.Setenv("ARTEMIS_SCREEN_PROTECT_IDLE_AFTER", "20m")

	m := NewModel()

	if m.screenProtectMode != screenProtectDriftIdle {
		t.Fatalf("expected env to enable drift-idle mode, got %v", m.screenProtectMode)
	}
	if m.screenProtectDriftAfter != 90*time.Second {
		t.Fatalf("expected env drift interval to be 90s, got %v", m.screenProtectDriftAfter)
	}
	if m.screenProtectIdleAfter != 20*time.Minute {
		t.Fatalf("expected env idle timeout to be 20m, got %v", m.screenProtectIdleAfter)
	}
	if m.lastActivityAt.IsZero() || m.screenProtectNow.IsZero() {
		t.Fatalf("expected screen protection clocks to be initialized")
	}
}

func TestNewModelFallsBackForInvalidScreenProtectionEnv(t *testing.T) {
	t.Setenv("ARTEMIS_SCREEN_PROTECT", "invalid")
	t.Setenv("ARTEMIS_SCREEN_PROTECT_DRIFT_INTERVAL", "0s")
	t.Setenv("ARTEMIS_SCREEN_PROTECT_IDLE_AFTER", "not-a-duration")

	m := NewModel()

	if m.screenProtectMode != screenProtectOff {
		t.Fatalf("expected invalid mode to fall back to off, got %v", m.screenProtectMode)
	}
	if m.screenProtectDriftAfter != defaultScreenProtectDriftInterval {
		t.Fatalf("expected invalid drift interval to fall back, got %v", m.screenProtectDriftAfter)
	}
	if m.screenProtectIdleAfter != defaultScreenProtectIdleAfter {
		t.Fatalf("expected invalid idle timeout to fall back, got %v", m.screenProtectIdleAfter)
	}
}

func TestUpdateCyclesScreenProtectionMode(t *testing.T) {
	m := Model{screenProtectMode: screenProtectOff}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd != nil {
		t.Fatalf("expected no command when cycling screen protection")
	}
	got := model.(Model)
	if got.screenProtectMode != screenProtectDrift {
		t.Fatalf("expected first p press to enable drift mode, got %v", got.screenProtectMode)
	}

	model, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got = model.(Model)
	if got.screenProtectMode != screenProtectDriftIdle {
		t.Fatalf("expected second p press to enable drift-idle mode, got %v", got.screenProtectMode)
	}

	model, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got = model.(Model)
	if got.screenProtectMode != screenProtectOff {
		t.Fatalf("expected third p press to wrap back to off, got %v", got.screenProtectMode)
	}
}

func TestIdleScreenProtectionConsumesWakeKey(t *testing.T) {
	before := time.Now().Add(-2 * time.Minute)
	model, cmd := Model{
		units:                  unitMetric,
		screenProtectMode:      screenProtectDriftIdle,
		screenProtectIdleAfter: time.Minute,
		lastActivityAt:         before,
		screenProtectNow:       before.Add(2 * time.Minute),
	}.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	if cmd != nil {
		t.Fatalf("expected wake key to avoid follow-up command")
	}
	got := model.(Model)
	if got.units != unitMetric {
		t.Fatalf("expected wake key to avoid applying the original action, got units %v", got.units)
	}
	if !got.lastActivityAt.After(before) {
		t.Fatalf("expected wake key to refresh last activity, got %v", got.lastActivityAt)
	}
	if got.screenProtectIdleActive() {
		t.Fatalf("expected wake key to clear idle state")
	}
}

func TestUpdateCyclesVisualEffects(t *testing.T) {
	m := Model{visualEffects: effectsStarsPulse}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Fatalf("expected no command when cycling visual effects")
	}
	got := model.(Model)
	if got.visualEffects != effectsStarsSprite {
		t.Fatalf("expected first s press to switch to sprite mode, got %v", got.visualEffects)
	}

	model, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got = model.(Model)
	if got.visualEffects != effectsPulseOnly {
		t.Fatalf("expected second s press to switch to pulse-only mode, got %v", got.visualEffects)
	}

	model, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got = model.(Model)
	if got.visualEffects != effectsStarsPulse {
		t.Fatalf("expected third s press to wrap to stars+pulse mode, got %v", got.visualEffects)
	}
}

func TestEnterOpensBlogReaderAndFetchesPost(t *testing.T) {
	m := Model{
		blogClient: nasablog.NewClient(),
		blogStatus: &nasablog.Status{
			Entries: []nasablog.Entry{{ID: 42, Title: "Flight Day 2", Link: "https://example.test/post"}},
		},
	}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected enter to trigger blog post fetch when content is not cached")
	}

	got := model.(Model)
	if !got.blogReaderOpen {
		t.Fatalf("expected enter to open blog reader")
	}
	if !got.blogPostLoading {
		t.Fatalf("expected enter to mark blog post as loading")
	}
}

func TestEnterOpensBlogReaderFromCacheWithoutFetch(t *testing.T) {
	m := Model{
		blogStatus: &nasablog.Status{
			Entries: []nasablog.Entry{{ID: 42, Title: "Flight Day 2", Link: "https://example.test/post"}},
		},
		blogPostCache: map[int]*nasablog.Post{
			42: {ID: 42, Title: "Flight Day 2", Content: "Crew wake and systems check."},
		},
	}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected cached blog post to open without fetch command")
	}

	got := model.(Model)
	if !got.blogReaderOpen {
		t.Fatalf("expected blog reader to open from cache")
	}
}

func TestBlogReaderEscapeClosesReader(t *testing.T) {
	model, cmd := Model{blogReaderOpen: true}.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected esc to close reader without command")
	}
	if model.(Model).blogReaderOpen {
		t.Fatalf("expected esc to close blog reader")
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

func TestUpdateStoresTrajectoryPathAndRefreshTime(t *testing.T) {
	before := time.Now()
	model, cmd := Model{}.Update(horizonsPathMsg{
		points: []horizons.Vector3{{X: 1}, {X: 2}, {X: 3}},
	})
	if cmd != nil {
		t.Fatalf("expected no follow-up command for trajectory path update")
	}

	got := model.(Model)
	if len(got.trajectoryPath) != 3 {
		t.Fatalf("expected trajectory path to be stored, got %d points", len(got.trajectoryPath))
	}
	if got.hzPathErr != nil {
		t.Fatalf("expected no trajectory path error, got %v", got.hzPathErr)
	}
	if got.lastHorizonPathFetch.Before(before) {
		t.Fatalf("expected trajectory path refresh time to be updated, got %v before %v", got.lastHorizonPathFetch, before)
	}
}

func TestUpdateTracksHorizonsAndDSNMetricHistories(t *testing.T) {
	model, cmd := Model{}.Update(horizonsMsg{
		state: &horizons.State{
			Position: horizons.Vector3{X: 1000, Y: 0, Z: 0},
			Velocity: horizons.Vector3{X: 2.5, Y: 0.1, Z: 0},
			Speed:    2.502,
		},
	})
	if cmd != nil {
		t.Fatalf("expected no follow-up command for horizons update")
	}

	got := model.(Model)
	if len(got.speedHistory) != 1 || got.speedHistory[0] != 2.502 {
		t.Fatalf("unexpected speed history: %#v", got.speedHistory)
	}
	if len(got.radialHistory) != 1 || got.radialHistory[0] != 2.5 {
		t.Fatalf("unexpected radial history: %#v", got.radialHistory)
	}

	model, cmd = got.Update(dsnMsg{
		status: &dsn.Status{
			Range: 42000,
			RTLT:  0.35,
			Dishes: []dsn.Dish{
				{
					DownSignals: []dsn.Signal{{Active: true, DataRate: 2_000_000}},
				},
			},
		},
	})
	if cmd != nil {
		t.Fatalf("expected no follow-up command for dsn update")
	}

	got = model.(Model)
	if len(got.dsnRangeHistory) != 1 || got.dsnRangeHistory[0] != 42000 {
		t.Fatalf("unexpected DSN range history: %#v", got.dsnRangeHistory)
	}
	if len(got.rtltHistory) != 1 || got.rtltHistory[0] != 0.35 {
		t.Fatalf("unexpected RTLT history: %#v", got.rtltHistory)
	}
	if len(got.dsnRateHistory) != 1 || got.dsnRateHistory[0] != 2_000_000 {
		t.Fatalf("unexpected DSN rate history: %#v", got.dsnRateHistory)
	}
}

func TestUpdateTracksSpaceWeatherHistories(t *testing.T) {
	model, cmd := Model{}.Update(swMsg{
		status: &spaceweather.Status{
			Kp:              4,
			Bz:              -5.5,
			Bt:              7.2,
			WindSpeed:       510,
			WindDensity:     8.3,
			WindTemp:        120000,
			ProtonFlux10MeV: 1.25,
		},
	})
	if cmd != nil {
		t.Fatalf("expected no follow-up command for space weather update")
	}

	got := model.(Model)
	if len(got.kpHistory) != 1 || got.kpHistory[0] != 4 {
		t.Fatalf("unexpected kp history: %#v", got.kpHistory)
	}
	if len(got.windDensityHistory) != 1 || got.windDensityHistory[0] != 8.3 {
		t.Fatalf("unexpected wind density history: %#v", got.windDensityHistory)
	}
	if len(got.protonFluxHistory) != 1 || got.protonFluxHistory[0] != 1.25 {
		t.Fatalf("unexpected proton history: %#v", got.protonFluxHistory)
	}
}

func TestTrajectoryPathWindowClampsToFlownMissionSpan(t *testing.T) {
	start, stop := trajectoryPathWindow(mission.LaunchTime.Add(25 * time.Hour))
	if !start.Equal(mission.LaunchTime.UTC()) {
		t.Fatalf("start = %v, want %v", start, mission.LaunchTime.UTC())
	}
	wantStop := mission.LaunchTime.Add(25 * time.Hour).UTC()
	if !stop.Equal(wantStop) {
		t.Fatalf("stop = %v, want %v", stop, wantStop)
	}

	_, afterMissionStop := trajectoryPathWindow(mission.LaunchTime.Add(20 * 24 * time.Hour))
	wantEnd := mission.LaunchTime.Add(mission.Timeline[len(mission.Timeline)-1].METOffset).UTC()
	if !afterMissionStop.Equal(wantEnd) {
		t.Fatalf("after-mission stop = %v, want %v", afterMissionStop, wantEnd)
	}
}

func TestTickRefreshesCachedTimeline(t *testing.T) {
	originalLaunchTime := mission.LaunchTime
	defer func() {
		mission.LaunchTime = originalLaunchTime
	}()

	now := time.Now().UTC()
	mission.LaunchTime = now.Add(-1 * time.Hour)

	m := Model{
		width:     120,
		height:    40,
		showGantt: true,
	}
	m.buildCache()
	before := m.cachedTimeline
	if before == "" {
		t.Fatal("expected initial cached timeline")
	}

	mission.LaunchTime = now.Add(-26 * time.Hour)
	model, _ := m.Update(tickMsg(now))
	got := model.(Model)

	if got.cachedTimeline == before {
		t.Fatalf("expected tick to refresh cached timeline when MET changes")
	}
}

func TestShouldRefreshVisualizationOnTick(t *testing.T) {
	tests := []struct {
		name string
		m    Model
		want bool
	}{
		{name: "trajectory view animates", m: Model{trajectoryView: 0}, want: true},
		{name: "orbital view animates", m: Model{trajectoryView: 1}, want: true},
		{name: "instruments view static", m: Model{trajectoryView: 2}, want: false},
		{name: "dsn sky static", m: Model{trajectoryView: 3}, want: false},
		{name: "weather trends static", m: Model{trajectoryView: 4}, want: false},
		{name: "fullscreen forces refresh", m: Model{trajectoryView: 4, visualizationFullscreen: true}, want: true},
	}

	for _, tc := range tests {
		if got := tc.m.shouldRefreshVisualizationOnTick(); got != tc.want {
			t.Fatalf("%s: shouldRefreshVisualizationOnTick() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestShouldProbeTerminalSize(t *testing.T) {
	m := Model{tickCount: 1}
	if m.shouldProbeTerminalSize() {
		t.Fatalf("expected non-interval tick count to skip terminal size probe")
	}

	m.tickCount = uint64(sizeProbeInterval / uiTickInterval)
	if !m.shouldProbeTerminalSize() {
		t.Fatalf("expected terminal size probe at configured interval")
	}
}

func TestScreenProtectOffsetCyclesSlowDriftPattern(t *testing.T) {
	base := time.Unix(100, 0)
	m := Model{
		screenProtectMode:       screenProtectDrift,
		screenProtectDriftAfter: time.Minute,
		lastActivityAt:          base,
	}

	cases := []struct {
		at    time.Time
		wantX int
		wantY int
	}{
		{base, 0, 0},
		{base.Add(time.Minute), 1, 0},
		{base.Add(2 * time.Minute), 1, 1},
		{base.Add(3 * time.Minute), 0, 1},
		{base.Add(4 * time.Minute), 0, 0},
	}

	for _, tc := range cases {
		gotX, gotY := m.screenProtectOffsetAt(tc.at)
		if gotX != tc.wantX || gotY != tc.wantY {
			t.Fatalf("screenProtectOffsetAt(%v) = (%d,%d), want (%d,%d)", tc.at, gotX, gotY, tc.wantX, tc.wantY)
		}
	}
}
