# Changelog

## Unreleased

### Added
- Native desktop notifications for mission phase changes and new mission log entries, with an in-app failure indicator and runtime uptime in the footer
- Fullscreen visualization mode with `MISSION CLOCK` and `SPACECRAFT STATE` embedded into the active visualization panel
- Derived spacecraft telemetry including Earth radial velocity, ecliptic longitude/latitude, and source data age
- Short instrument trend graphs for Earth radial velocity, DSN range, RTLT, and downlink rate
- Runtime unit toggle for switching the dashboard between metric and imperial telemetry
- Trajectory Sun-direction marker for quick orientation in the Earth-centered plot
- `RELEASE.md` documenting the manual release workflow

### Changed
- Trajectory rendering now uses live Horizons-sampled Earth-centered mission geometry instead of a scripted mission-progress arc
- Trajectory, orbital, and instrument distance displays now share the same effective Earth-distance source so labels stay consistent across views
- Instruments view now uses a wider split layout, clearer scope labels, and additional derived telemetry to better use available space
- Spacecraft state, instruments, trajectory labels, orbital labels, DSN range text, and solar-wind speed now follow the selected unit system
- Mission clock and Gantt panels now derive mission day totals from the actual timeline data instead of hard-coded day counts
- `make changefile TAG=...` now generates changelog entries from the latest tag through `HEAD`, and the old `make tag` helper has been removed

### Fixed
- Narrow-height instrument rendering regression where primary telemetry could disappear in short terminals
- Gantt/timeline caching drift so live mission timing indicators now refresh on tick
- Trajectory path visibility, compact distance formatting, and path legends (`away` / `return`) for clearer interpretation
- Proximity scope Moon bearing and Horizons sample selection so visualizations better match the underlying data source

## v0.6.0

- Hide footer view shortcut when visualization panel is hidden
- Refine TUI layout sizing and narrow-screen behavior
- Add sensitive file patterns to .gitignore
- Update screenshot
- Add changelog target to Makefile

## v0.5.0

### Added
- **Orbital Context view** — top-down Earth-Moon system map with spacecraft plotted at real ecliptic coordinates, Moon at actual angular position, orbit ring, distance scale rings, and position trail
- **Instrument Panel view** — six-instrument HUD with velocity gauge/sparkline, range finder, bearing compass, signal health, radiation environment, and proximity scope
- **View cycling** — press `v` to cycle between Trajectory, Orbital Context, and Instrument Panel views
- Speed history ring buffer (24 samples) powering sparkline and min/avg/max stats
- Position trail ring buffer (12 samples) for orbital view spacecraft trail

### Changed
- All trajectory, orbital, and instrument styles are now theme-aware — colors update when cycling themes with `c`
- Improved Retro theme visibility: brightened Dim (`#555500` → `#777711`) and Muted (`#999933` → `#AAAA44`)
- Improved Critical theme visibility: brightened Dim (`#553333` → `#774444`), Muted (`#886666` → `#AA8888`), Secondary (`#994444` → `#BB6655`), and Cyan (`#CC6666` → `#DD8888`)
- Help line now includes `v: view`

### Fixed
- Instrument panel layout — replaced broken canvas blitting (which split ANSI escape sequences) with lipgloss grid composition

## v0.4.0

### Added
- AGENTS.md with polling rates, build system, and panel descriptions
- MIT license
- Plain-language descriptions for signal and space weather panels
- Release binaries added to .gitignore

## v0.3.0

Initial tagged release with core dashboard panels: mission clock, spacecraft state, DSN tracking, space weather, mission timeline/Gantt, trajectory visualization, crew roster, and mission log.
