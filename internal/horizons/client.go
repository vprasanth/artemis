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
	Time          time.Time
	Position      Vector3
	Velocity      Vector3
	EarthDist     float64
	MoonDist      float64
	Speed         float64
	Timestamp     time.Time
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
	start := now.Add(-1 * time.Minute)
	stop := now.Add(1 * time.Minute)

	earthState, err := c.fetchVectors(start, stop, "500@399")
	if err != nil {
		return nil, fmt.Errorf("horizons earth-centered: %w", err)
	}

	moonState, err := c.fetchVectors(start, stop, "500@301")
	if err != nil {
		earthState.MoonDist = -1
		earthState.Timestamp = time.Now().UTC()
		return earthState, nil
	}

	earthState.MoonDist = moonState.Position.Magnitude()
	earthState.Timestamp = time.Now().UTC()
	return earthState, nil
}

func (c *Client) fetchVectors(start, stop time.Time, center string) (*State, error) {
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

	return parseVectors(string(body))
}

var soeRegex = regexp.MustCompile(`(?s)\$\$SOE\s*\n(.*?)\n\s*\$\$EOE`)

func parseVectors(text string) (*State, error) {
	matches := soeRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no ephemeris data found between $$SOE and $$EOE")
	}

	block := strings.TrimSpace(matches[1])
	lines := strings.Split(block, "\n")

	// Take the first data point. Format:
	// 2461132.916666667 = A.D. 2026-Apr-02 10:00:00.0000 TDB
	//  X = ... Y = ... Z = ...
	//  VX= ... VY= ... VZ= ...

	if len(lines) < 3 {
		return nil, fmt.Errorf("insufficient lines in ephemeris data")
	}

	state := &State{}

	// Parse position line
	posLine := lines[1]
	state.Position = parseXYZ(posLine, "X", "Y", "Z")

	// Parse velocity line
	velLine := lines[2]
	state.Velocity = parseXYZ(velLine, "VX", "VY", "VZ")

	state.EarthDist = state.Position.Magnitude()
	state.Speed = state.Velocity.Magnitude()

	return state, nil
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
