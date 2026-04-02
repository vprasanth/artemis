package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary   = lipgloss.Color("#4FC3F7")
	colorSecondary = lipgloss.Color("#81C784")
	colorAccent    = lipgloss.Color("#FFB74D")
	colorDim       = lipgloss.Color("#666666")
	colorBright    = lipgloss.Color("#FFFFFF")
	colorMuted     = lipgloss.Color("#888888")
	colorGreen     = lipgloss.Color("#4CAF50")
	colorRed       = lipgloss.Color("#EF5350")
	colorYellow    = lipgloss.Color("#FFF176")
	colorCyan      = lipgloss.Color("#4DD0E1")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBright)

	metStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	activeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	completedStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	pendingStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	currentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	signalUpStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	signalDownStyle = lipgloss.NewStyle().
			Foreground(colorCyan)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	progressFullStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	crewRoleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	crewNameStyle = lipgloss.NewStyle().
			Foreground(colorBright)

	crewAgencyStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Gantt chart styles
	ganttCompletedBar = lipgloss.NewStyle().
				Foreground(colorGreen)

	ganttActiveBar = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	ganttCursorBar = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	ganttPendingBar = lipgloss.NewStyle().
			Foreground(colorDim)

	ganttNowMarker = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	// Mission log styles
	logTimeStyle = lipgloss.NewStyle().
			Foreground(colorCyan)

	logTitleStyle = lipgloss.NewStyle().
			Foreground(colorBright)
)
