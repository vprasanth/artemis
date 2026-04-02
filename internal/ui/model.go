package ui

import (
	"os"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"artemis/internal/dsn"
	"artemis/internal/horizons"
	"artemis/internal/nasablog"
	"artemis/internal/spaceweather"
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

type swMsg struct {
	status *spaceweather.Status
	err    error
}

type blogMsg struct {
	status *nasablog.Status
	err    error
}

type openBrowserMsg struct{ err error }

type Model struct {
	width  int
	height int

	showGantt bool   // toggle between Gantt chart and event timeline
	showStars bool   // toggle starfield in trajectory
	tickCount uint64 // monotonic frame counter for animation

	dsnClient      *dsn.Client
	horizonsClient *horizons.Client
	swClient       *spaceweather.Client
	blogClient     *nasablog.Client

	dsnStatus  *dsn.Status
	dsnErr     error
	dsnLoading bool

	hzState   *horizons.State
	hzErr     error
	hzLoading bool

	swStatus  *spaceweather.Status
	swErr     error
	swLoading bool

	blogStatus       *nasablog.Status
	blogErr          error
	blogLoading      bool
	selectedLogEntry int

	lastDSNFetch     time.Time
	lastHorizonFetch time.Time
	lastSWFetch      time.Time
	lastBlogFetch    time.Time

	// Layout computed from terminal dimensions.
	layout map[panelID]panelLayout

	// Pre-rendered panel strings, rebuilt only when data or size changes.
	cachedDSN        string
	cachedSpacecraft string
	cachedTrajectory string
	cachedCrew       string
	cachedSW         string
	cachedBlog       string
	cachedTimeline   string
}

func NewModel() Model {
	return Model{
		showGantt:      true,
		showStars:      true,
		dsnClient:      dsn.NewClient(),
		horizonsClient: horizons.NewClient(),
		swClient:       spaceweather.NewClient(),
		blogClient:     nasablog.NewClient(),
		dsnLoading:     true,
		hzLoading:      true,
		swLoading:      true,
		blogLoading:    true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		fetchDSN(m.dsnClient),
		fetchHorizons(m.horizonsClient),
		fetchSW(m.swClient),
		fetchBlog(m.blogClient),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "t":
			m.showGantt = !m.showGantt
			m.buildCache()
		case "c":
			NextTheme()
			m.buildCache()
		case "s":
			m.showStars = !m.showStars
			m.buildCache()
		case "tab":
			if m.blogStatus != nil && len(m.blogStatus.Entries) > 0 {
				max := len(m.blogStatus.Entries) - 1
				if max > 4 {
					max = 4
				}
				m.selectedLogEntry = (m.selectedLogEntry + 1) % (max + 1)
				m.buildCache()
			}
		case "shift+tab":
			if m.blogStatus != nil && len(m.blogStatus.Entries) > 0 {
				max := len(m.blogStatus.Entries) - 1
				if max > 4 {
					max = 4
				}
				m.selectedLogEntry = (m.selectedLogEntry - 1 + max + 1) % (max + 1)
				m.buildCache()
			}
		case "enter":
			if m.blogStatus != nil && m.selectedLogEntry < len(m.blogStatus.Entries) {
				link := m.blogStatus.Entries[m.selectedLogEntry].Link
				if link != "" {
					return m, openBrowserCmd(link)
				}
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
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())

		// Directly query terminal size to catch tmux pane resizes
		// that may not trigger SIGWINCH / WindowSizeMsg.
		if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			if w != m.width || h != m.height {
				m.width = w
				m.height = h
				m.buildCache()
			}
		}

		// Re-render trajectory every tick for animation (stars twinkle, spacecraft pulse).
		if m.layout != nil {
			if pl := m.layout[panelTrajectory]; pl.visible {
				plotH := pl.height - 3
				if plotH < 6 {
					plotH = 6
				}
				m.cachedTrajectory = renderTrajectoryPanel(m, m.width, plotH)
			}
		}

		now := time.Now()
		if now.Sub(m.lastDSNFetch) > 5*time.Second && !m.dsnLoading {
			m.dsnLoading = true
			cmds = append(cmds, fetchDSN(m.dsnClient))
		}
		if now.Sub(m.lastHorizonFetch) > 30*time.Second && !m.hzLoading {
			m.hzLoading = true
			cmds = append(cmds, fetchHorizons(m.horizonsClient))
		}
		if now.Sub(m.lastSWFetch) > 60*time.Second && !m.swLoading {
			m.swLoading = true
			cmds = append(cmds, fetchSW(m.swClient))
		}
		if now.Sub(m.lastBlogFetch) > 60*time.Second && !m.blogLoading {
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
		}
		m.buildCache()

	case blogMsg:
		m.blogLoading = false
		m.lastBlogFetch = time.Now()
		if msg.err != nil {
			m.blogErr = msg.err
		} else {
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

	case openBrowserMsg:
		// Silently consume browser result.
	}

	return m, nil
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
	m.cachedSpacecraft = renderSpacecraftPanel(*m, w-w/3)
	m.cachedDSN = renderDSNPanel(*m, w)
	m.cachedSW = renderSpaceWeatherPanel(*m, w)
	m.cachedBlog = renderMissionLogPanel(*m, w, 5, m.selectedLogEntry)
	m.cachedCrew = renderCrewPanel(w)

	if m.showGantt {
		m.cachedTimeline = renderGanttPanel(w)
	} else {
		m.cachedTimeline = renderTimelinePanel(w)
	}

	// Phase 2: Measure fixed-height panels.
	measured := map[panelID]int{
		panelDSN:          countLines(m.cachedDSN),
		panelTimeline:     countLines(m.cachedTimeline),
		panelSpaceWeather: countLines(m.cachedSW),
		panelMissionLog:   countLines(m.cachedBlog),
		panelCrew:         countLines(m.cachedCrew),
	}

	// Measure always-visible panels for the fixed height budget.
	header := renderHeader(w)
	clockPanel := renderClockPanel(w / 3)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, clockPanel, m.cachedSpacecraft)
	fixedHeight := countLines(header) + countLines(topRow) + 1 // +1 for help line

	// Phase 3: Layout decides which fixed panels fit; trajectory gets remaining space.
	var trajectoryAvail int
	m.layout, trajectoryAvail = computeLayout(w, m.height, fixedHeight, measured)

	// Phase 4: Render trajectory at the allocated height (flex panel).
	m.cachedTrajectory = ""
	if m.layout[panelTrajectory].visible {
		plotH := trajectoryAvail - 3 // subtract border(2) + title(1)
		if plotH < 6 {
			plotH = 6
		}
		m.cachedTrajectory = renderTrajectoryPanel(*m, w, plotH)
	}

	// Store effective width for View().
	m.layout[panelHeader] = panelLayout{visible: true, height: countLines(header), width: w}
	m.layout[panelTopRow] = panelLayout{visible: true, height: countLines(topRow), width: w}
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

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
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
