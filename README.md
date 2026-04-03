# Artemis II Mission Dashboard

A real-time terminal dashboard for tracking NASA's [Artemis II](https://www.nasa.gov/humans-in-space/artemis/) crewed lunar flyby mission, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Artemis II Dashboard](screenshot.png)

## Live Data Sources

- **Deep Space Network** -- real-time antenna tracking, signal status, and range via [DSN Now](https://eyes.nasa.gov/dsn/data/dsn.xml)
- **JPL Horizons** -- spacecraft position, velocity, and Earth/Moon distance via the [Horizons API](https://ssd.jpl.nasa.gov/api/horizons.api) (spacecraft ID `-1024`)
- **NOAA SWPC** -- space weather conditions (Kp index, solar wind, Bz, proton flux, flare class) via [SWPC services](https://services.swpc.noaa.gov)
- **NASA Blog** -- mission log entries from the [Artemis blog](https://www.nasa.gov/wp-json/wp/v2/nasa-blog?categories=2918) WordPress REST API

## Requirements

- Go 1.22+
- A terminal emulator with 256-color support (most modern terminals)
- Minimum terminal size: 60 columns x 14 rows (more space shows more panels)

## Build and Run

```sh
go build -o artemis ./main.go
./artemis
```

Or run directly:

```sh
go run main.go
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` / `Esc` | Quit |
| `t` | Toggle between Gantt chart and event timeline |
| `c` | Cycle color theme (Default, Retro, Hi-Con, Critical) |
| `v` | Switch between Trajectory, Orbital Context, and Instruments views |
| `f` | Toggle fullscreen visualization mode |
| `s` | Toggle star animation in trajectory view |
| `n` | Toggle native notifications |
| `r` | Force-refresh all data sources |
| `j` / `Tab` | Select next mission log entry |
| `k` / `Shift+Tab` | Select previous mission log entry |
| `Enter` | Open selected log entry in browser |

## Panels

The dashboard shows panels based on available terminal height, in priority order:

1. **Mission Clock** -- MET, UTC time, mission day, next event countdown
2. **Spacecraft State** -- distance from Earth/Moon, speed, Earth radial rate, ecliptic lon/lat, position vector, RTLT, AOS/LOS signal status
3. **Space Weather** -- NOAA R/S/G scales, Kp index, solar wind, Bz, proton flux
4. **Deep Space Network** -- active dishes, signal bands, data rates, range
5. **Mission Timeline** -- Gantt chart or scrolling event list with 25 mission events
6. **Mission Log** -- latest NASA blog posts with selection and browser opening
7. **Trajectory** -- Earth-centered Horizons mission path with sampled arc status, twinkling stars, and current Earth/Moon/Orion positions
8. **Crew** -- the four [Artemis II astronauts](https://www.nasa.gov/feature/our-artemis-crew/) and their roles

Visualization quick read:
- **Trajectory** shows the sampled Earth-centered mission path, with the current Orion, Earth, and Moon positions overlaid.
- **Orbital Context** shows the current Earth-Moon-Orion geometry in a fixed top-down Earth-centered map with reference rings.
- **Instruments** shows the same current state as telemetry gauges, short trend graphs, and directional scopes rather than a literal map.

Instruments quick read:
- **Velocity** shows current speed, Earth radial velocity, inertial velocity components, and short speed/radial trendlines.
- **Range** shows current Earth and Moon distance, Earth/Moon split, Earth-Moon baseline, and DSN range trend.
- **Signal** shows AOS/LOS, active DSN dish, RTLT, downlink rate, and short RTLT/downlink trendlines.
- **Bearing** is Orion's Earth-centered heading in the ecliptic plane.
- **Proximity** plots the Moon relative to Orion, with the center crosshair representing the spacecraft.

Press `f` to expand the active visualization into fullscreen mode with `MISSION CLOCK` and `SPACECRAFT STATE` embedded inside the visualization panel.

## Data Refresh Rates

Polling intervals are tuned for long-running sessions to minimize battery and network usage:

| Source | Interval |
|--------|----------|
| Deep Space Network | 30 seconds |
| JPL Horizons | 5 minutes |
| Space Weather | 5 minutes |
| NASA Blog | 1 hour |

Press `r` at any time to force an immediate refresh of all sources.
Trendlines advance when fresh source samples arrive, so `r` will also force the instruments sparklines to append a new point immediately.

## Color Themes

Cycle through four themes with `c`:

- **Default** -- blue/green on dark background
- **Retro** -- amber/green phosphor terminal
- **Hi-Con** -- high-contrast white/green/yellow
- **Critical** -- dark red mission-critical aesthetic
