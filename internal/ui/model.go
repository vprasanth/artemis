package ui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
	"artemis/internal/mission"
	"artemis/internal/nasablog"
	"artemis/internal/spaceweather"
)

const (
	maxSpeedHistory      = 24
	maxPositionTrail     = 12
	maxMetricHistory     = 24
	trajectoryPathSample = 30 * time.Minute
	uiTickInterval       = 1 * time.Second
	sizeProbeInterval    = 5 * time.Second

	defaultScreenProtectDriftInterval = 60 * time.Second
	defaultScreenProtectIdleAfter     = 15 * time.Minute
)

type tickMsg time.Time

type dsnMsg struct {
	status *dsn.Status
	err    error
}

type horizonsMsg struct {
	state *horizons.State
	err   error
}

type horizonsPathMsg struct {
	points []horizons.Vector3
	err    error
}

type swMsg struct {
	status *spaceweather.Status
	err    error
}

type blogMsg struct {
	status *nasablog.Status
	err    error
}

type blogPostMsg struct {
	id   int
	post *nasablog.Post
	err  error
}

type openBrowserMsg struct{ err error }
type notificationResultMsg struct{ err error }
type notificationSender func(title, body string) tea.Cmd

type screenProtectMode int

const (
	screenProtectOff screenProtectMode = iota
	screenProtectDrift
	screenProtectDriftIdle
)

func (m screenProtectMode) next() screenProtectMode {
	switch m {
	case screenProtectOff:
		return screenProtectDrift
	case screenProtectDrift:
		return screenProtectDriftIdle
	default:
		return screenProtectOff
	}
}

func (m screenProtectMode) driftEnabled() bool {
	return m != screenProtectOff
}

func (m screenProtectMode) idleEnabled() bool {
	return m == screenProtectDriftIdle
}

func (m screenProtectMode) wideName() string {
	switch m {
	case screenProtectDrift:
		return "drift"
	case screenProtectDriftIdle:
		return "drift+idle"
	default:
		return "off"
	}
}

func (m screenProtectMode) compactName() string {
	switch m {
	case screenProtectDriftIdle:
		return "d+i"
	default:
		return m.wideName()
	}
}

func parseScreenProtectMode(raw string) screenProtectMode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "drift":
		return screenProtectDrift
	case "drift-idle":
		return screenProtectDriftIdle
	default:
		return screenProtectOff
	}
}

func parseEnvDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

type visualEffectsMode int

const (
	effectsStarsPulse visualEffectsMode = iota
	effectsStarsSprite
	effectsPulseOnly
)

func (m visualEffectsMode) next() visualEffectsMode {
	switch m {
	case effectsStarsPulse:
		return effectsStarsSprite
	case effectsStarsSprite:
		return effectsPulseOnly
	default:
		return effectsStarsPulse
	}
}

func (m visualEffectsMode) starsEnabled() bool {
	return m != effectsPulseOnly
}

func (m visualEffectsMode) spriteEnabled() bool {
	return m == effectsStarsSprite
}

func (m visualEffectsMode) wideName() string {
	switch m {
	case effectsStarsSprite:
		return "ship"
	case effectsPulseOnly:
		return "off"
	default:
		return "stars"
	}
}

func (m visualEffectsMode) compactName() string {
	return m.wideName()
}

type Model struct {
	width  int
	height int

	showGantt               bool // toggle between Gantt chart and event timeline
	visualEffects           visualEffectsMode
	screenProtectMode       screenProtectMode
	notificationsEnabled    bool // toggle native desktop notifications
	debugKeysEnabled        bool // enable debug-only keybindings
	visualizationFullscreen bool // expand visualization into the primary content area
	units                   unitSystem
	tickCount               uint64 // monotonic frame counter for animation
	trajectoryView          int    // 0=Trajectory, 1=Orbital, 2=Instruments
	lastActivityAt          time.Time
	screenProtectNow        time.Time
	screenProtectDriftAfter time.Duration
	screenProtectIdleAfter  time.Duration

	speedHistory    []float64          // ring buffer (cap 24) for sparkline
	positionTrail   []horizons.Vector3 // ring buffer (cap 12) for recent live trail
	trajectoryPath  []horizons.Vector3 // Horizons-sampled mission arc for trajectory view
	radialHistory   []float64          // ring buffer for Earth radial velocity trend
	dsnRangeHistory []float64          // ring buffer for DSN downleg range trend
	rtltHistory     []float64          // ring buffer for DSN round-trip light time trend
	dsnRateHistory  []float64          // ring buffer for active DSN downlink rate trend

	dsnClient      *dsn.Client
	horizonsClient *horizons.Client
	swClient       *spaceweather.Client
	blogClient     *nasablog.Client

	dsnStatus  *dsn.Status
	dsnErr     error
	dsnLoading bool

	hzState       *horizons.State
	hzErr         error
	hzLoading     bool
	hzPathErr     error
	hzPathLoading bool

	swStatus  *spaceweather.Status
	swErr     error
	swLoading bool

	blogStatus       *nasablog.Status
	blogErr          error
	blogLoading      bool
	selectedLogEntry int
	blogPrimed       bool
	lastSeenBlogID   int
	blogPostCache    map[int]*nasablog.Post
	blogPostErr      error
	blogPostLoading  bool
	blogReaderOpen   bool
	blogReaderScroll int

	kpHistory          []float64
	bzHistory          []float64
	btHistory          []float64
	windSpeedHistory   []float64
	windDensityHistory []float64
	windTempHistory    []float64
	protonFluxHistory  []float64

	lastDSNFetch         time.Time
	lastHorizonFetch     time.Time
	lastHorizonPathFetch time.Time
	lastSWFetch          time.Time
	lastBlogFetch        time.Time
	startedAt            time.Time

	phasePrimed         bool
	lastPhaseIndex      int
	notifier            notificationSender
	notificationError   string
	notificationErrorAt time.Time

	// Layout computed from terminal dimensions.
	layout map[panelID]panelLayout

	// Pre-rendered panel strings, rebuilt only when data or size changes.
	cachedDSN        string
	cachedTrajectory string
	cachedCrew       string
	cachedSW         string
	cachedBlog       string
	cachedTimeline   string
}

func NewModel() Model {
	now := time.Now()
	return Model{
		showGantt:               true,
		visualEffects:           effectsStarsPulse,
		screenProtectMode:       parseScreenProtectMode(os.Getenv("ARTEMIS_SCREEN_PROTECT")),
		notificationsEnabled:    true,
		debugKeysEnabled:        os.Getenv("ARTEMIS_DEBUG_KEYS") == "1",
		units:                   unitMetric,
		lastActivityAt:          now,
		screenProtectNow:        now,
		screenProtectDriftAfter: parseEnvDuration("ARTEMIS_SCREEN_PROTECT_DRIFT_INTERVAL", defaultScreenProtectDriftInterval),
		screenProtectIdleAfter:  parseEnvDuration("ARTEMIS_SCREEN_PROTECT_IDLE_AFTER", defaultScreenProtectIdleAfter),
		dsnClient:               dsn.NewClient(),
		horizonsClient:          horizons.NewClient(),
		swClient:                spaceweather.NewClient(),
		blogClient:              nasablog.NewClient(),
		dsnLoading:              true,
		hzLoading:               true,
		swLoading:               true,
		blogLoading:             true,
		hzPathLoading:           true,
		startedAt:               now,
		notifier:                nativeNotifyCmd,
		blogPostCache:           make(map[int]*nasablog.Post),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		fetchDSN(m.dsnClient),
		fetchHorizons(m.horizonsClient),
		fetchHorizonPath(m.horizonsClient),
		fetchSW(m.swClient),
		fetchBlog(m.blogClient),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		now := time.Now()
		m.screenProtectNow = now
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		if m.screenProtectIdleActiveAt(now) {
			m.markUserActivity(now)
			return m, nil
		}
		m.markUserActivity(now)
		if m.blogReaderOpen {
			return m.updateBlogReader(msg)
		}
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		case "t":
			m.showGantt = !m.showGantt
			m.buildCache()
		case "c":
			NextTheme()
			m.buildCache()
		case "s":
			m.visualEffects = m.visualEffects.next()
			m.buildCache()
		case "n":
			m.notificationsEnabled = !m.notificationsEnabled
			m.buildCache()
		case "u":
			m.units = m.units.next()
			m.buildCache()
		case "p":
			m.screenProtectMode = m.screenProtectMode.next()
			m.buildCache()
		case "N":
			if m.debugKeysEnabled {
				return m, m.debugPhaseNotificationCmd()
			}
		case "v":
			m.trajectoryView = (m.trajectoryView + 1) % 5
			m.buildCache()
		case "f":
			m.visualizationFullscreen = !m.visualizationFullscreen
			m.buildCache()
		case "r":
			var cmds []tea.Cmd
			if !m.dsnLoading {
				m.dsnLoading = true
				cmds = append(cmds, fetchDSN(m.dsnClient))
			}
			if !m.hzLoading {
				m.hzLoading = true
				cmds = append(cmds, fetchHorizons(m.horizonsClient))
			}
			if !m.hzPathLoading {
				m.hzPathLoading = true
				cmds = append(cmds, fetchHorizonPath(m.horizonsClient))
			}
			if !m.swLoading {
				m.swLoading = true
				cmds = append(cmds, fetchSW(m.swClient))
			}
			if !m.blogLoading {
				m.blogLoading = true
				cmds = append(cmds, fetchBlog(m.blogClient))
			}
			return m, tea.Batch(cmds...)
		case "tab", "j":
			if m.blogStatus != nil && len(m.blogStatus.Entries) > 0 {
				max := len(m.blogStatus.Entries) - 1
				if max > 4 {
					max = 4
				}
				m.selectedLogEntry = (m.selectedLogEntry + 1) % (max + 1)
				m.buildCache()
			}
		case "shift+tab", "k":
			if m.blogStatus != nil && len(m.blogStatus.Entries) > 0 {
				max := len(m.blogStatus.Entries) - 1
				if max > 4 {
					max = 4
				}
				m.selectedLogEntry = (m.selectedLogEntry - 1 + max + 1) % (max + 1)
				m.buildCache()
			}
		case "enter":
			if entry, ok := m.selectedBlogEntry(); ok {
				m.blogReaderOpen = true
				m.blogReaderScroll = 0
				m.blogPostErr = nil
				if _, cached := m.blogPostCache[entry.ID]; cached {
					return m, nil
				}
				if !m.blogPostLoading {
					m.blogPostLoading = true
					return m, fetchBlogPost(m.blogClient, entry.ID)
				}
			}
		case "o":
			if entry, ok := m.selectedBlogEntry(); ok && entry.Link != "" {
				return m, openBrowserCmd(entry.Link)
			}
		}

	case tea.WindowSizeMsg:
		if msg.Width != m.width || msg.Height != m.height {
			m.width = msg.Width
			m.height = msg.Height
			m.buildCache()
		}

	case tickMsg:
		m.tickCount++
		m.screenProtectNow = time.Time(msg)
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())
		if m.notificationError != "" && time.Since(m.notificationErrorAt) > 5*time.Second {
			m.notificationError = ""
		}
		if notifyCmd := m.handlePhaseNotification(time.Time(msg)); notifyCmd != nil {
			cmds = append(cmds, notifyCmd)
		}

		// Directly query terminal size to catch tmux pane resizes
		// that may not trigger SIGWINCH / WindowSizeMsg.
		if m.shouldProbeTerminalSize() {
			if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				if w != m.width || h != m.height {
					m.width = w
					m.height = h
					m.buildCache()
				}
			}
		}

		if m.layout != nil {
			if pl := m.layout[panelTrajectory]; pl.visible && m.shouldRefreshVisualizationOnTick() {
				m.cachedTrajectory = m.renderCachedTrajectoryPanel(pl.height)
			}
			if pl := m.layout[panelTimeline]; pl.visible {
				timeline := m.renderCachedTimelinePanel(pl.width)
				if measureHeight(timeline) != pl.height {
					m.buildCache()
				} else {
					m.cachedTimeline = timeline
				}
			}
		}

		now := time.Now()
		if now.Sub(m.lastDSNFetch) > 30*time.Second && !m.dsnLoading {
			m.dsnLoading = true
			cmds = append(cmds, fetchDSN(m.dsnClient))
		}
		if now.Sub(m.lastHorizonFetch) > 5*time.Minute && !m.hzLoading {
			m.hzLoading = true
			cmds = append(cmds, fetchHorizons(m.horizonsClient))
		}
		if now.Sub(m.lastHorizonPathFetch) > 5*time.Minute && !m.hzPathLoading {
			m.hzPathLoading = true
			cmds = append(cmds, fetchHorizonPath(m.horizonsClient))
		}
		if now.Sub(m.lastSWFetch) > 5*time.Minute && !m.swLoading {
			m.swLoading = true
			cmds = append(cmds, fetchSW(m.swClient))
		}
		if now.Sub(m.lastBlogFetch) > 60*time.Minute && !m.blogLoading {
			m.blogLoading = true
			cmds = append(cmds, fetchBlog(m.blogClient))
		}
		return m, tea.Batch(cmds...)

	case dsnMsg:
		m.dsnLoading = false
		m.lastDSNFetch = time.Now()
		if msg.err != nil {
			m.dsnErr = msg.err
		} else {
			m.dsnStatus = msg.status
			m.dsnErr = nil
			if msg.status.Range > 0 {
				m.dsnRangeHistory = appendMetricHistory(m.dsnRangeHistory, msg.status.Range)
			}
			if msg.status.RTLT > 0 {
				m.rtltHistory = appendMetricHistory(m.rtltHistory, msg.status.RTLT)
			}
			if rate := activeDSNRate(msg.status); rate > 0 {
				m.dsnRateHistory = appendMetricHistory(m.dsnRateHistory, rate)
			}
		}
		m.buildCache()

	case horizonsMsg:
		m.hzLoading = false
		m.lastHorizonFetch = time.Now()
		if msg.err != nil {
			m.hzErr = msg.err
		} else {
			m.hzState = msg.state
			m.hzErr = nil

			// Append to speed history ring buffer.
			m.speedHistory = append(m.speedHistory, msg.state.Speed)
			if len(m.speedHistory) > maxSpeedHistory {
				m.speedHistory = m.speedHistory[len(m.speedHistory)-maxSpeedHistory:]
			}
			if radial, ok := radialVelocity(msg.state.Position, msg.state.Velocity); ok {
				m.radialHistory = appendMetricHistory(m.radialHistory, radial)
			}

			// Append to position trail ring buffer.
			m.positionTrail = append(m.positionTrail, msg.state.Position)
			if len(m.positionTrail) > maxPositionTrail {
				m.positionTrail = m.positionTrail[len(m.positionTrail)-maxPositionTrail:]
			}
		}
		m.buildCache()

	case horizonsPathMsg:
		m.hzPathLoading = false
		m.lastHorizonPathFetch = time.Now()
		if msg.err != nil {
			m.hzPathErr = msg.err
		} else {
			m.trajectoryPath = msg.points
			m.hzPathErr = nil
		}
		m.buildCache()

	case swMsg:
		m.swLoading = false
		m.lastSWFetch = time.Now()
		if msg.err != nil {
			m.swErr = msg.err
		} else {
			m.swStatus = msg.status
			m.swErr = nil
			m.kpHistory = appendMetricHistory(m.kpHistory, msg.status.Kp)
			m.bzHistory = appendMetricHistory(m.bzHistory, msg.status.Bz)
			m.btHistory = appendMetricHistory(m.btHistory, msg.status.Bt)
			m.windSpeedHistory = appendMetricHistory(m.windSpeedHistory, msg.status.WindSpeed)
			m.windDensityHistory = appendMetricHistory(m.windDensityHistory, msg.status.WindDensity)
			m.windTempHistory = appendMetricHistory(m.windTempHistory, msg.status.WindTemp)
			m.protonFluxHistory = appendMetricHistory(m.protonFluxHistory, msg.status.ProtonFlux10MeV)
		}
		m.buildCache()

	case blogMsg:
		m.blogLoading = false
		m.lastBlogFetch = time.Now()
		var notifyCmd tea.Cmd
		if msg.err != nil {
			m.blogErr = msg.err
		} else {
			notifyCmd = m.handleBlogNotification(msg.status)
			m.blogStatus = msg.status
			m.blogErr = nil
		}
		// Clamp selection to valid range.
		if m.blogStatus != nil && len(m.blogStatus.Entries) > 0 {
			max := len(m.blogStatus.Entries) - 1
			if max > 4 {
				max = 4
			}
			if m.selectedLogEntry > max {
				m.selectedLogEntry = max
			}
		} else {
			m.selectedLogEntry = 0
		}
		m.buildCache()
		return m, notifyCmd

	case blogPostMsg:
		m.blogPostLoading = false
		if msg.err != nil {
			m.blogPostErr = msg.err
		} else if msg.post != nil {
			if m.blogPostCache == nil {
				m.blogPostCache = make(map[int]*nasablog.Post)
			}
			m.blogPostCache[msg.id] = msg.post
			m.blogPostErr = nil
		}

	case openBrowserMsg:
		// Silently consume browser result.

	case notificationResultMsg:
		if msg.err != nil {
			m.notificationError = "notify failed"
			m.notificationErrorAt = time.Now()
		}
	}

	return m, nil
}

func (m *Model) markUserActivity(now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	m.lastActivityAt = now
	m.screenProtectNow = now
}

func (m Model) currentScreenProtectTime() time.Time {
	if !m.screenProtectNow.IsZero() {
		return m.screenProtectNow
	}
	return time.Now()
}

func (m Model) screenProtectIdleActive() bool {
	return m.screenProtectIdleActiveAt(m.currentScreenProtectTime())
}

func (m Model) screenProtectIdleActiveAt(now time.Time) bool {
	if !m.screenProtectMode.idleEnabled() || m.screenProtectIdleAfter <= 0 {
		return false
	}
	if now.IsZero() || m.lastActivityAt.IsZero() {
		return false
	}
	return !now.Before(m.lastActivityAt.Add(m.screenProtectIdleAfter))
}

func (m Model) screenProtectOffsetAt(now time.Time) (int, int) {
	if !m.screenProtectMode.driftEnabled() || m.screenProtectIdleActiveAt(now) || m.screenProtectDriftAfter <= 0 {
		return 0, 0
	}
	base := m.lastActivityAt
	if base.IsZero() {
		base = m.startedAt
	}
	if base.IsZero() || now.IsZero() || now.Before(base) {
		return 0, 0
	}

	pattern := [...]struct {
		x int
		y int
	}{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 1},
	}
	step := int(now.Sub(base) / m.screenProtectDriftAfter)
	offset := pattern[step%len(pattern)]
	return offset.x, offset.y
}

func (m Model) updateBlogReader(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.blogReaderOpen = false
		m.blogReaderScroll = 0
		m.blogPostErr = nil
		return m, nil
	case "j", "down":
		m.blogReaderScroll++
	case "k", "up":
		if m.blogReaderScroll > 0 {
			m.blogReaderScroll--
		}
	case "pgdown", "space":
		m.blogReaderScroll += m.blogReaderPageStep()
	case "pgup":
		m.blogReaderScroll -= m.blogReaderPageStep()
		if m.blogReaderScroll < 0 {
			m.blogReaderScroll = 0
		}
	case "g":
		m.blogReaderScroll = 0
	case "G":
		m.blogReaderScroll = 1 << 20
	case "o":
		if entry, ok := m.selectedBlogEntry(); ok && entry.Link != "" {
			return m, openBrowserCmd(entry.Link)
		}
	case "r":
		if entry, ok := m.selectedBlogEntry(); ok && !m.blogPostLoading {
			m.blogPostLoading = true
			return m, fetchBlogPost(m.blogClient, entry.ID)
		}
	}
	return m, nil
}

func (m Model) selectedBlogEntry() (nasablog.Entry, bool) {
	if m.blogStatus == nil || m.selectedLogEntry < 0 || m.selectedLogEntry >= len(m.blogStatus.Entries) {
		return nasablog.Entry{}, false
	}
	return m.blogStatus.Entries[m.selectedLogEntry], true
}

func (m Model) blogReaderPageStep() int {
	step := m.height / 2
	if step < 4 {
		step = 4
	}
	return step
}

func appendMetricHistory(history []float64, value float64) []float64 {
	history = append(history, value)
	if len(history) > maxMetricHistory {
		history = history[len(history)-maxMetricHistory:]
	}
	return history
}

func activeDSNRate(status *dsn.Status) float64 {
	if status == nil {
		return 0
	}
	for _, dish := range status.Dishes {
		for _, signal := range dish.DownSignals {
			if signal.Active && signal.DataRate > 0 {
				return signal.DataRate
			}
		}
	}
	return 0
}

func (m *Model) handleBlogNotification(status *nasablog.Status) tea.Cmd {
	if status == nil || len(status.Entries) == 0 {
		return nil
	}

	latest := status.Entries[0]
	if !m.blogPrimed {
		m.blogPrimed = true
		m.lastSeenBlogID = latest.ID
		return nil
	}
	if latest.ID == m.lastSeenBlogID {
		return nil
	}

	m.lastSeenBlogID = latest.ID
	if !m.notificationsEnabled {
		return nil
	}

	return m.notificationCmd("New Mission Log Entry", latest.Title)
}

func (m *Model) handlePhaseNotification(now time.Time) tea.Cmd {
	if now.IsZero() {
		now = time.Now()
	}

	phaseIdx := mission.CurrentPhase(now.Sub(mission.LaunchTime))
	if phaseIdx < 0 || phaseIdx >= len(mission.Phases) {
		return nil
	}
	if !m.phasePrimed {
		m.phasePrimed = true
		m.lastPhaseIndex = phaseIdx
		return nil
	}
	if phaseIdx <= m.lastPhaseIndex {
		return nil
	}

	m.lastPhaseIndex = phaseIdx
	if !m.notificationsEnabled {
		return nil
	}

	return m.notificationCmd("Mission Phase Change", mission.Phases[phaseIdx].Name)
}

func (m Model) notificationCmd(title, body string) tea.Cmd {
	if m.notifier != nil {
		return m.notifier(title, body)
	}
	return nativeNotifyCmd(title, body)
}

func (m Model) debugPhaseNotificationCmd() tea.Cmd {
	phaseIdx := mission.CurrentPhase(mission.MET())
	if phaseIdx < len(mission.Phases)-1 {
		phaseIdx++
	}
	return m.notificationCmd("Mission Phase Change", mission.Phases[phaseIdx].Name)
}

// buildCache pre-renders expensive panels so View() only assembles strings.
// Fixed-height panels are rendered and measured first, then the layout engine
// decides which fit. Trajectory is a flex panel that expands to fill remaining space.
func (m *Model) buildCache() {
	if m.width == 0 {
		return
	}

	w := m.width

	// Phase 1: Render fixed-height panels.
	m.cachedDSN = renderDSNPanel(*m, w)
	m.cachedSW = renderSpaceWeatherPanel(*m, w)
	m.cachedBlog = renderMissionLogPanel(*m, w, 5, m.selectedLogEntry)
	m.cachedCrew = renderCrewPanel(w)

	m.cachedTimeline = m.renderCachedTimelinePanel(w)

	// Phase 2: Measure fixed-height panels.
	measured := map[panelID]int{
		panelDSN:          measureHeight(m.cachedDSN),
		panelTimeline:     measureHeight(m.cachedTimeline),
		panelSpaceWeather: measureHeight(m.cachedSW),
		panelMissionLog:   measureHeight(m.cachedBlog),
		panelCrew:         measureHeight(m.cachedCrew),
	}

	header := renderHeader(w)
	topRow := renderTopRow(*m, w)

	if m.visualizationFullscreen {
		m.buildFullscreenLayout(w, header)
	} else {
		fixedHeight := measureHeight(header) + measureHeight(topRow) + 1 // +1 for help line

		// Phase 3: Layout decides which fixed panels fit; trajectory gets remaining space.
		var trajectoryAvail int
		m.layout, trajectoryAvail = computeLayout(w, m.height, fixedHeight, measured)

		// Phase 4: Render trajectory at the allocated height (flex panel).
		m.cachedTrajectory = ""
		if m.layout[panelTrajectory].visible {
			m.cachedTrajectory = m.renderCachedTrajectoryPanel(trajectoryAvail)
			if actualHeight := measureHeight(m.cachedTrajectory); actualHeight > 0 {
				m.layout[panelTrajectory] = panelLayout{visible: true, height: actualHeight, width: w}
			}
		}
	}

	// Store effective width for View().
	m.layout[panelHeader] = panelLayout{visible: true, height: measureHeight(header), width: w}
	m.layout[panelTopRow] = panelLayout{visible: !m.visualizationFullscreen, height: measureHeight(topRow), width: w}
	m.layout[panelHelp] = panelLayout{visible: true, height: 1, width: w}

	// Clear hidden panels.
	if !m.layout[panelDSN].visible {
		m.cachedDSN = ""
	}
	if !m.layout[panelSpaceWeather].visible {
		m.cachedSW = ""
	}
	if !m.layout[panelTimeline].visible {
		m.cachedTimeline = ""
	}
	if !m.layout[panelMissionLog].visible {
		m.cachedBlog = ""
	}
	if !m.layout[panelCrew].visible {
		m.cachedCrew = ""
	}
}

func (m *Model) buildFullscreenLayout(w int, header string) {
	m.layout = map[panelID]panelLayout{
		panelDSN:          {visible: false, height: 0, width: w},
		panelTimeline:     {visible: false, height: 0, width: w},
		panelSpaceWeather: {visible: false, height: 0, width: w},
		panelMissionLog:   {visible: false, height: 0, width: w},
		panelCrew:         {visible: false, height: 0, width: w},
		panelTopRow:       {visible: false, height: 0, width: w},
	}

	available := m.height - measureHeight(header) - 1 // footer
	if available < 0 {
		available = 0
	}

	const minTrajectoryH = 9
	m.cachedTrajectory = ""
	if available >= minTrajectoryH {
		m.cachedTrajectory = m.renderCachedTrajectoryPanel(available)
		if actualHeight := measureHeight(m.cachedTrajectory); actualHeight > 0 && actualHeight <= available {
			m.layout[panelTrajectory] = panelLayout{visible: true, height: actualHeight, width: w}
			return
		}
	}

	m.layout[panelTrajectory] = panelLayout{visible: false, height: 0, width: w}
}

func (m Model) renderCachedTrajectoryPanel(availableHeight int) string {
	if availableHeight < 9 {
		return ""
	}

	plotH := availableHeight - 3
	if plotH < 6 {
		plotH = 6
	}

	for plotH >= 6 {
		panel := renderVisualizationPanel(m, m.width, plotH, m.visualizationFullscreen)
		actualHeight := measureHeight(panel)
		if actualHeight <= availableHeight {
			return panel
		}

		plotH -= actualHeight - availableHeight
	}

	return ""
}

func (m Model) renderCachedTimelinePanel(w int) string {
	if m.showGantt {
		return renderGanttPanelAt(w, mission.MET())
	}
	return renderTimelinePanelAt(w, mission.MET())
}

func (m Model) shouldRefreshVisualizationOnTick() bool {
	if m.visualizationFullscreen {
		return true
	}

	switch m.trajectoryView {
	case 0, 1:
		return true
	default:
		return false
	}
}

func (m Model) shouldProbeTerminalSize() bool {
	intervalTicks := int(sizeProbeInterval / uiTickInterval)
	if intervalTicks <= 1 {
		return true
	}
	return m.tickCount%uint64(intervalTicks) == 0
}

func tickCmd() tea.Cmd {
	return tea.Tick(uiTickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDSN(client *dsn.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := client.Fetch()
		return dsnMsg{status: status, err: err}
	}
}

func fetchHorizons(client *horizons.Client) tea.Cmd {
	return func() tea.Msg {
		state, err := client.Fetch()
		return horizonsMsg{state: state, err: err}
	}
}

func fetchHorizonPath(client *horizons.Client) tea.Cmd {
	return func() tea.Msg {
		start, stop := trajectoryPathWindow(time.Now())
		points, err := client.FetchTrajectoryPath(start, stop, trajectoryPathSample)
		return horizonsPathMsg{points: points, err: err}
	}
}

func trajectoryPathWindow(now time.Time) (time.Time, time.Time) {
	start := mission.LaunchTime.UTC()
	stop := now.UTC()
	missionEnd := mission.LaunchTime.Add(mission.Timeline[len(mission.Timeline)-1].METOffset).UTC()

	if stop.After(missionEnd) {
		stop = missionEnd
	}
	if stop.Before(start.Add(trajectoryPathSample)) {
		stop = start.Add(trajectoryPathSample)
		if stop.After(missionEnd) {
			stop = missionEnd
		}
	}

	return start, stop
}

func fetchSW(client *spaceweather.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := client.Fetch()
		return swMsg{status: status, err: err}
	}
}

func fetchBlog(client *nasablog.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := client.Fetch()
		return blogMsg{status: status, err: err}
	}
}

func fetchBlogPost(client *nasablog.Client, id int) tea.Cmd {
	return func() tea.Msg {
		post, err := client.FetchPost(id)
		return blogPostMsg{id: id, post: post, err: err}
	}
}

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		return openBrowserMsg{err: cmd.Start()}
	}
}
