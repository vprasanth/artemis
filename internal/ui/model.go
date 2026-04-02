package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

type Model struct {
	width  int
	height int

	showGantt bool // toggle between Gantt chart and event timeline

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

	blogStatus  *nasablog.Status
	blogErr     error
	blogLoading bool

	lastDSNFetch     time.Time
	lastHorizonFetch time.Time
	lastSWFetch      time.Time
	lastBlogFetch    time.Time

	// Pre-rendered panel strings, rebuilt only when data or size changes.
	cachedDSN        string
	cachedSpacecraft string
	cachedTrajectory string
	cachedCrew       string
	cachedSW         string
	cachedBlog       string
}

func NewModel() Model {
	return Model{
		showGantt:      true,
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.buildCache()

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())

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
		m.buildCache()
	}

	return m, nil
}

// buildCache pre-renders expensive panels so View() only assembles strings.
func (m *Model) buildCache() {
	if m.width == 0 {
		return
	}
	w := m.width
	if w > 120 {
		w = 120
	}
	m.cachedDSN = renderDSNPanel(*m, w)
	m.cachedSpacecraft = renderSpacecraftPanel(*m, w-w/3)
	m.cachedTrajectory = renderTrajectoryPanel(*m, w)
	m.cachedCrew = renderCrewPanel(w)
	m.cachedSW = renderSpaceWeatherPanel(*m, w)
	m.cachedBlog = renderMissionLogPanel(*m, w)
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
