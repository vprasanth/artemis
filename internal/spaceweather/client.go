package spaceweather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	scalesURL    = "https://services.swpc.noaa.gov/products/noaa-scales.json"
	kpURL        = "https://services.swpc.noaa.gov/json/planetary_k_index_1m.json"
	plasmaURL    = "https://services.swpc.noaa.gov/products/solar-wind/plasma-5-minute.json"
	magURL       = "https://services.swpc.noaa.gov/products/solar-wind/mag-5-minute.json"
	xrayFlareURL = "https://services.swpc.noaa.gov/json/goes/primary/xray-flares-latest.json"
	protonURL    = "https://services.swpc.noaa.gov/json/goes/primary/integral-protons-1-day.json"
	alertsURL    = "https://services.swpc.noaa.gov/products/alerts.json"
)

type ScaleLevel struct {
	Scale int
	Text  string
}

type Status struct {
	// NOAA R/S/G scales (0-5)
	RadioBlackout  ScaleLevel
	SolarRadiation ScaleLevel
	GeomagStorm    ScaleLevel

	// Solar wind
	WindSpeed   float64 // km/s
	WindDensity float64 // n/cc
	WindTemp    float64 // K

	// Magnetic field
	Bz float64 // nT (negative = southward = bad)
	Bt float64 // nT total

	// Kp index
	Kp float64

	// X-ray flare class
	CurrentFlareClass string

	// Proton flux (>=10 MeV, pfu)
	ProtonFlux10MeV float64

	// Latest alert summary
	LatestAlert string

	Timestamp time.Time
}

type TrendHistory struct {
	Kp              []float64
	Bz              []float64
	Bt              []float64
	WindSpeed       []float64
	WindDensity     []float64
	WindTemp        []float64
	ProtonFlux10MeV []float64
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Fetch() (*Status, error) {
	s := &Status{Timestamp: time.Now().UTC()}
	var lastErr error

	if err := c.fetchScales(s); err != nil {
		lastErr = err
	}
	if err := c.fetchKp(s); err != nil {
		lastErr = err
	}
	if err := c.fetchPlasma(s); err != nil {
		lastErr = err
	}
	if err := c.fetchMag(s); err != nil {
		lastErr = err
	}
	if err := c.fetchXray(s); err != nil {
		lastErr = err
	}
	if err := c.fetchProtons(s); err != nil {
		lastErr = err
	}
	if err := c.fetchAlerts(s); err != nil {
		lastErr = err
	}

	// Return partial data even if some fetches failed
	if s.Kp > 0 || s.WindSpeed > 0 || s.RadioBlackout.Scale >= 0 {
		return s, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("spaceweather: all fetches failed, last: %w", lastErr)
	}
	return s, nil
}

func (c *Client) FetchTrendHistory(limit int) (*TrendHistory, error) {
	if limit <= 0 {
		return &TrendHistory{}, nil
	}

	h := &TrendHistory{}
	var lastErr error

	if err := c.fetchKpHistory(h, limit); err != nil {
		lastErr = err
	}
	if err := c.fetchPlasmaHistory(h, limit); err != nil {
		lastErr = err
	}
	if err := c.fetchMagHistory(h, limit); err != nil {
		lastErr = err
	}
	if err := c.fetchProtonHistory(h, limit); err != nil {
		lastErr = err
	}

	if len(h.Kp) > 0 || len(h.WindSpeed) > 0 || len(h.Bz) > 0 || len(h.ProtonFlux10MeV) > 0 {
		return h, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("spaceweather history: all fetches failed, last: %w", lastErr)
	}
	return h, nil
}

func (c *Client) fetchScales(s *Status) error {
	body, err := c.get(scalesURL)
	if err != nil {
		return err
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("parse scales: %w", err)
	}

	current, ok := data["0"]
	if !ok {
		return fmt.Errorf("no current scales data")
	}

	var entry map[string]json.RawMessage
	if err := json.Unmarshal(current, &entry); err != nil {
		return fmt.Errorf("parse current scales: %w", err)
	}

	s.RadioBlackout = parseScale(entry, "R")
	s.SolarRadiation = parseScale(entry, "S")
	s.GeomagStorm = parseScale(entry, "G")
	return nil
}

func parseScale(entry map[string]json.RawMessage, key string) ScaleLevel {
	// The JSON has nested objects: "R": {"Scale": "0", "Text": "none"}, etc.
	sl := ScaleLevel{}
	raw, ok := entry[key]
	if !ok {
		return sl
	}
	var obj struct {
		Scale string `json:"Scale"`
		Text  string `json:"Text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		sl.Scale, _ = strconv.Atoi(obj.Scale)
		sl.Text = obj.Text
	}
	return sl
}

func (c *Client) fetchKp(s *Status) error {
	body, err := c.get(kpURL)
	if err != nil {
		return err
	}

	// Fields can be int or float; use json.Number for flexibility
	var entries []map[string]interface{}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse kp: %w", err)
	}
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		s.Kp = parseJSONFloat(last["estimated_kp"])
		if s.Kp == 0 {
			s.Kp = parseJSONFloat(last["kp_index"])
		}
	}
	return nil
}

func (c *Client) fetchKpHistory(h *TrendHistory, limit int) error {
	body, err := c.get(kpURL)
	if err != nil {
		return err
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse kp history: %w", err)
	}

	values := make([]float64, 0, len(entries))
	for _, entry := range entries {
		kp := parseJSONFloat(entry["estimated_kp"])
		if kp == 0 {
			kp = parseJSONFloat(entry["kp_index"])
		}
		values = append(values, kp)
	}
	h.Kp = tailFloatSlice(values, limit)
	return nil
}

func (c *Client) fetchPlasma(s *Status) error {
	body, err := c.get(plasmaURL)
	if err != nil {
		return err
	}

	// JSON is array-of-arrays: [["time_tag","density","speed","temperature"], [...], ...]
	var rows [][]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("parse plasma: %w", err)
	}
	if len(rows) < 2 {
		return nil
	}

	// Last row is most recent
	last := rows[len(rows)-1]
	if len(last) >= 4 {
		s.WindDensity = parseJSONFloat(last[1])
		s.WindSpeed = parseJSONFloat(last[2])
		s.WindTemp = parseJSONFloat(last[3])
	}
	return nil
}

func (c *Client) fetchPlasmaHistory(h *TrendHistory, limit int) error {
	body, err := c.get(plasmaURL)
	if err != nil {
		return err
	}

	var rows [][]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("parse plasma history: %w", err)
	}
	if len(rows) < 2 {
		return nil
	}

	var density, speed, temp []float64
	for _, row := range rows[1:] {
		if len(row) < 4 {
			continue
		}
		density = append(density, parseJSONFloat(row[1]))
		speed = append(speed, parseJSONFloat(row[2]))
		temp = append(temp, parseJSONFloat(row[3]))
	}
	h.WindDensity = tailFloatSlice(density, limit)
	h.WindSpeed = tailFloatSlice(speed, limit)
	h.WindTemp = tailFloatSlice(temp, limit)
	return nil
}

func (c *Client) fetchMag(s *Status) error {
	body, err := c.get(magURL)
	if err != nil {
		return err
	}

	// [["time_tag","bx_gsm","by_gsm","bz_gsm","lon_gsm","lat_gsm","bt"], [...], ...]
	var rows [][]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("parse mag: %w", err)
	}
	if len(rows) < 2 {
		return nil
	}

	last := rows[len(rows)-1]
	if len(last) >= 7 {
		s.Bz = parseJSONFloat(last[3])
		s.Bt = parseJSONFloat(last[6])
	}
	return nil
}

func (c *Client) fetchMagHistory(h *TrendHistory, limit int) error {
	body, err := c.get(magURL)
	if err != nil {
		return err
	}

	var rows [][]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("parse mag history: %w", err)
	}
	if len(rows) < 2 {
		return nil
	}

	var bz, bt []float64
	for _, row := range rows[1:] {
		if len(row) < 7 {
			continue
		}
		bz = append(bz, parseJSONFloat(row[3]))
		bt = append(bt, parseJSONFloat(row[6]))
	}
	h.Bz = tailFloatSlice(bz, limit)
	h.Bt = tailFloatSlice(bt, limit)
	return nil
}

func (c *Client) fetchXray(s *Status) error {
	body, err := c.get(xrayFlareURL)
	if err != nil {
		return err
	}

	var entries []struct {
		CurrentClass string `json:"current_class"`
		MaxClass     string `json:"max_class"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse xray: %w", err)
	}
	if len(entries) > 0 {
		s.CurrentFlareClass = entries[0].CurrentClass
		if s.CurrentFlareClass == "" {
			s.CurrentFlareClass = entries[0].MaxClass
		}
	}
	return nil
}

func (c *Client) fetchProtons(s *Status) error {
	body, err := c.get(protonURL)
	if err != nil {
		return err
	}

	var entries []struct {
		TimeTag string  `json:"time_tag"`
		Flux    float64 `json:"flux"`
		Energy  string  `json:"energy"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse protons: %w", err)
	}

	// Find latest >=10 MeV entry
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Energy == ">=10 MeV" {
			s.ProtonFlux10MeV = entries[i].Flux
			break
		}
	}
	return nil
}

func (c *Client) fetchProtonHistory(h *TrendHistory, limit int) error {
	body, err := c.get(protonURL)
	if err != nil {
		return err
	}

	var entries []struct {
		TimeTag string  `json:"time_tag"`
		Flux    float64 `json:"flux"`
		Energy  string  `json:"energy"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse proton history: %w", err)
	}

	values := make([]float64, 0, len(entries))
	for _, entry := range entries {
		if entry.Energy != ">=10 MeV" {
			continue
		}
		values = append(values, entry.Flux)
	}
	h.ProtonFlux10MeV = tailFloatSlice(values, limit)
	return nil
}

func (c *Client) fetchAlerts(s *Status) error {
	body, err := c.get(alertsURL)
	if err != nil {
		return err
	}

	var entries []struct {
		ProductID     string `json:"product_id"`
		IssueDatetime string `json:"issue_datetime"`
		Message       string `json:"message"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return fmt.Errorf("parse alerts: %w", err)
	}

	if len(entries) > 0 {
		last := entries[len(entries)-1]
		// Extract first meaningful line from message
		msg := last.Message
		if len(msg) > 120 {
			msg = msg[:120]
		}
		s.LatestAlert = msg
	}
	return nil
}

func (c *Client) get(url string) ([]byte, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func parseJSONFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	}
	return 0
}

func tailFloatSlice(values []float64, limit int) []float64 {
	if limit <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) > limit {
		values = values[len(values)-limit:]
	}
	out := make([]float64, len(values))
	copy(out, values)
	return out
}
