# Artemis II Mission Dashboard — Agent Context

Real-time terminal dashboard for NASA's Artemis II crewed lunar flyby mission.
Built with Go, Bubble Tea (TUI framework), and Lipgloss (terminal styling).

## File Structure

```
artemis/
├── main.go                        # Entry point, --version flag, version via ldflags
├── go.mod / go.sum                # Go 1.26.1, bubbletea, lipgloss
├── Makefile                       # build, run, clean, tag, release targets
├── LICENSE                        # MIT
└── internal/
    ├── ui/                        # Terminal UI layer
    │   ├── model.go               # Bubble Tea model, event loop, caching
    │   ├── panels.go              # Render functions for all dashboard panels
    │   ├── trajectory.go          # ASCII Earth→Moon trajectory with animation
    │   ├── gantt.go               # Gantt chart for mission timeline
    │   ├── layout.go              # Responsive layout engine (fixed + flex panels)
    │   └── styles.go              # 4 color themes, all style definitions
    ├── mission/                   # Static mission data
    │   ├── timeline.go            # 25 events, MET calculations, crew roster
    │   └── phases.go              # 5 mission phases with MET ranges
    ├── horizons/                  # JPL Horizons API client
    │   └── client.go              # Spacecraft position/velocity, occultation check
    ├── dsn/                       # Deep Space Network client
    │   └── client.go              # Antenna tracking, signal status (XML feed)
    ├── spaceweather/              # NOAA SWPC client
    │   └── client.go              # R/S/G scales, Kp, solar wind, Bz (7 endpoints)
    └── nasablog/                  # NASA blog client
        └── client.go              # Mission log entries (WordPress REST API)
```

## Architecture

**Pattern:** Bubble Tea Model-Update-View with async command batching.

**Data flow:**
1. `model.go` creates HTTP clients for 4 external APIs
2. A 500ms tick drives animation and polls APIs at staggered intervals
3. Fetch results arrive as typed messages (dsnMsg, horizonsMsg, etc.)
4. `buildCache()` pre-renders all panels, measures heights, runs layout engine
5. `View()` assembles cached panel strings — no computation at render time

**Polling intervals:** DSN 30s, Horizons 5min, Space Weather 5min, Blog 1hr.
Press `r` to force-refresh all sources on demand.

**Layout engine** (`layout.go`): Fixed-height panels are measured after rendering,
then `computeLayout()` decides which fit. Trajectory is the flex panel that
expands to fill remaining vertical space. Minimum terminal: 60×14.

## Key Identifiers

- **Spacecraft ID (Horizons):** `-1024`
- **DSN target name:** `EM2`
- **Launch time:** April 1, 2026, 22:35:12 UTC
- **Mission duration:** ~9 days
- **NASA blog category ID:** `2918`

## Panel System

| Panel            | Type  | Render function              | Data source     |
|------------------|-------|------------------------------|-----------------|
| Header/Progress  | Fixed | `renderHeader()`             | mission pkg     |
| Mission Clock    | Fixed | `renderClockPanel()`         | mission pkg     |
| Spacecraft State | Fixed | `renderSpacecraftPanel()`    | horizons + DSN  |
| DSN Status       | Fixed | `renderDSNPanel()`           | dsn pkg         |
| Space Weather    | Fixed | `renderSpaceWeatherPanel()`  | spaceweather    |
| Timeline/Gantt   | Fixed | `renderTimelinePanel()` / `renderGanttPanel()` | mission pkg |
| Mission Log      | Fixed | `renderMissionLogPanel()`    | nasablog pkg    |
| Trajectory       | Flex  | `renderTrajectoryPanel()`    | horizons + mission |
| Crew             | Fixed | `renderCrewPanel()`          | mission pkg     |

## Styles & Themes

4 themes defined in `styles.go`: Default, Retro, Hi-Contrast, Mission Critical.
Trajectory uses fixed colors (not theme-dependent) for visual consistency:
Earth=blue, Moon=white, Spacecraft=orange (or red during LOS).

Theme-dependent styles are package-level vars reassigned by `applyTheme()`.

## Horizons Client Details

`Fetch()` makes two API calls per invocation:
1. Earth-centered vectors (`500@399`) → Position, Velocity, EarthDist, Speed
2. Moon-centered vectors (`500@301`) → MoonPosition, MoonDist

`IsOccluded()` checks geometric lunar occultation: whether the Earth→SC line
passes within 1737.4 km of the Moon's center. Used for AOS/LOS signal indicator
in the spacecraft panel and red spacecraft glyph in the trajectory view.

## Keybindings

- `q`/`Esc`/`Ctrl+C` — Quit
- `t` — Toggle Gantt chart vs scrolling timeline
- `c` — Cycle color themes
- `s` — Toggle starfield animation
- `r` — Force-refresh all data sources
- `j`/`k` — Navigate mission log entries
- `Enter` — Open selected blog post in browser

## Build & Release

Version is embedded via `ldflags` (`-X main.version=...`). The `version` var
in `main.go` defaults to `"dev"` for local builds. `--version`/`-v` flag prints
and exits.

**Makefile targets:**
- `make` / `make build` — build for current platform with git-derived version
- `make run` — build and run
- `make clean` — remove all binaries
- `make tag TAG=vX.Y.Z` — create annotated tag and push to remote
- `make release` — cross-compile for darwin/linux × amd64/arm64

Binary naming convention: `artemis-{os}-{arch}` (e.g. `artemis-darwin-arm64`).

## Plain-Language Panel Descriptions

The spacecraft panel shows AOS/LOS with a dim explanation ("acquisition of
signal — Earth contact nominal" or "loss of signal — Moon blocking Earth
contact"). The space weather panel includes a `swSummary()` one-liner that
translates NOAA scales into crew-impact language (e.g. "All quiet — nominal
conditions for crew and spacecraft").
