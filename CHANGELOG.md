# Changelog

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
