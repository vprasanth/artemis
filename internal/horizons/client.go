package horizons

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const apiURL = "https://ssd.jpl.nasa.gov/api/horizons.api"

const SpacecraftID = "-1024"

type Vector3 struct {
	X, Y, Z float64
}

func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

type State struct {
	Time         time.Time
	Position     Vector3
	Velocity     Vector3
	MoonPosition Vector3
	EarthDist    float64
	MoonDist     float64
	Speed        float64
	Timestamp    time.Time
}

// IsOccluded returns true when the Moon blocks line-of-sight from Earth to
// the spacecraft. It checks whether the closest point on the Earth→SC line
// passes within the Moon's physical radius (1737.4 km) of the Moon's center.
func (s *State) IsOccluded() bool {
	const moonRadius = 1737.4 // km

	// Guard: MoonPosition not yet available.
	if s.MoonPosition.X == 0 && s.MoonPosition.Y == 0 && s.MoonPosition.Z == 0 {
		return false
	}

	// Earth is at origin. Spacecraft is at s.Position.
	// Moon center (Earth-centered) = s.Position - s.MoonPosition
	// because MoonPosition is the SC position relative to the Moon,
	// so Moon = SC_earth - SC_moon.
	moonX := s.Position.X - s.MoonPosition.X
	moonY := s.Position.Y - s.MoonPosition.Y
	moonZ := s.Position.Z - s.MoonPosition.Z

	// Parameterize Earth→SC line as P(t) = t * Position, t ∈ [0,1].
	// Vector from P(t) to Moon center: (moonX - t*Px, moonY - t*Py, moonZ - t*Pz)
	// Minimize distance² → t = dot(Moon, SC) / dot(SC, SC).
	dot_sc_sc := s.Position.X*s.Position.X + s.Position.Y*s.Position.Y + s.Position.Z*s.Position.Z
	if dot_sc_sc == 0 {
		return false
	}

	dot_moon_sc := moonX*s.Position.X + moonY*s.Position.Y + moonZ*s.Position.Z
	t := dot_moon_sc / dot_sc_sc

	// Moon must be between Earth and SC (not behind Earth or beyond SC).
	if t <= 0 || t >= 1 {
		return false
	}

	// Closest distance from the line to Moon center.
	dx := moonX - t*s.Position.X
	dy := moonY - t*s.Position.Y
	dz := moonZ - t*s.Position.Z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

	return dist < moonRadius
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Fetch() (*State, error) {
	now := time.Now().UTC()
	earthState, err := c.fetchVectors(now, "500@399")
	if err != nil {
		return nil, fmt.Errorf("horizons earth-centered: %w", err)
	}

	moonState, err := c.fetchVectors(now, "500@301")
	if err != nil {
		earthState.MoonDist = -1
		earthState.Timestamp = time.Now().UTC()
		return earthState, nil
	}

	earthState.MoonPosition = moonState.Position
	earthState.MoonDist = moonState.Position.Magnitude()
	earthState.Timestamp = time.Now().UTC()
	return earthState, nil
}

func (c *Client) fetchVectors(target time.Time, center string) (*State, error) {
	start := target.Add(-1 * time.Minute)
	stop := target.Add(1 * time.Minute)

	params := url.Values{}
	params.Set("format", "text")
	params.Set("COMMAND", "'"+SpacecraftID+"'")
	params.Set("MAKE_EPHEM", "'YES'")
	params.Set("EPHEM_TYPE", "'VECTORS'")
	params.Set("CENTER", "'"+center+"'")
	params.Set("START_TIME", "'"+start.Format("2006-01-02 15:04")+"'")
	params.Set("STOP_TIME", "'"+stop.Format("2006-01-02 15:04")+"'")
	params.Set("STEP_SIZE", "'1 min'")
	params.Set("REF_PLANE", "'ECLIPTIC'")
	params.Set("VEC_TABLE", "'2'")

	reqURL := apiURL + "?" + params.Encode()
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("horizons fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("horizons read: %w", err)
	}

	return parseVectors(string(body), target)
}

var soeRegex = regexp.MustCompile(`(?s)\$\$SOE\s*\n(.*?)\n\s*\$\$EOE`)

func parseVectors(text string, target time.Time) (*State, error) {
	matches := soeRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no ephemeris data found between $$SOE and $$EOE")
	}

	block := strings.TrimSpace(matches[1])
	lines := strings.Split(block, "\n")

	// Each sample is three logical lines:
	// 2461132.916666667 = A.D. 2026-Apr-02 10:00:00.0000 TDB
	//  X = ... Y = ... Z = ...
	//  VX= ... VY= ... VZ= ...
	var (
		bestSample *State
		bestDelta  time.Duration
	)

	for i := 0; i+2 < len(lines); i++ {
		timeLine := strings.TrimSpace(lines[i])
		if !strings.Contains(timeLine, "A.D.") {
			continue
		}

		sampleTime, err := parseEphemerisTime(timeLine)
		if err != nil {
			continue
		}

		state := &State{
			Time:     sampleTime,
			Position: parseXYZ(lines[i+1], "X", "Y", "Z"),
			Velocity: parseXYZ(lines[i+2], "VX", "VY", "VZ"),
		}
		state.EarthDist = state.Position.Magnitude()
		state.Speed = state.Velocity.Magnitude()

		delta := absDuration(sampleTime.Sub(target))
		if bestSample == nil || delta < bestDelta {
			bestSample = state
			bestDelta = delta
		}
	}

	if bestSample == nil {
		return nil, fmt.Errorf("no parseable ephemeris samples found")
	}

	return bestSample, nil
}

func parseXYZ(line, xKey, yKey, zKey string) Vector3 {
	v := Vector3{}
	v.X = extractValue(line, xKey)
	v.Y = extractValue(line, yKey)
	v.Z = extractValue(line, zKey)
	return v
}

func extractValue(line, key string) float64 {
	// Match patterns like "X =-2.348958094658357E+04" or "VX= 7.345021532670720E-01"
	pattern := regexp.MustCompile(key + `\s*=\s*([+-]?\d+\.\d+E[+-]\d+)`)
	match := pattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return 0
	}
	val, _ := strconv.ParseFloat(match[1], 64)
	return val
}

func parseEphemerisTime(line string) (time.Time, error) {
	parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid ephemeris time line: %q", line)
	}

	timestamp := strings.TrimSpace(parts[1])
	timestamp = strings.TrimPrefix(timestamp, "A.D. ")
	timestamp = strings.TrimSuffix(timestamp, " TDB")

	return time.ParseInLocation("2006-Jan-02 15:04:05.0000", timestamp, time.UTC)
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
