package dsn

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const feedURL = "https://eyes.nasa.gov/dsn/data/dsn.xml"

const TargetSpacecraft = "EM2"

type Signal struct {
	Active     bool
	SignalType string
	DataRate   float64
	Band       string
	Power      float64
}

type Target struct {
	Name          string
	UplegRange    float64
	DownlegRange  float64
	RTLT          float64
}

type Dish struct {
	Name      string
	Azimuth   float64
	Elevation float64
	Station   string
	UpSignals []Signal
	DownSignals []Signal
	Targets   []Target
}

type StationInfo struct {
	Name      string
	Location  string
}

type Status struct {
	Dishes    []Dish
	Range     float64
	RTLT      float64
	Timestamp time.Time
}

var stationNames = map[string]string{
	"gdscc": "Goldstone, CA",
	"cdscc": "Canberra, AU",
	"mdscc": "Madrid, ES",
}

// XML structures for parsing DSN feed.
// In the DSN XML, <station> and <dish> are siblings under <dsn>,
// not parent-child. We parse them both as direct children.
type xmlDSN struct {
	XMLName  xml.Name      `xml:"dsn"`
	Stations []xmlStation  `xml:"station"`
	Dishes   []xmlDish     `xml:"dish"`
}

type xmlStation struct {
	Name         string `xml:"name,attr"`
	FriendlyName string `xml:"friendlyName,attr"`
}

type xmlDish struct {
	Name        string          `xml:"name,attr"`
	Azimuth     string          `xml:"azimuthAngle,attr"`
	Elevation   string          `xml:"elevationAngle,attr"`
	Activity    string          `xml:"activity,attr"`
	UpSignals   []xmlUpSignal   `xml:"upSignal"`
	DownSignals []xmlDownSignal `xml:"downSignal"`
	Targets     []xmlTarget     `xml:"target"`
}

type xmlUpSignal struct {
	Active      string `xml:"active,attr"`
	SignalType  string `xml:"signalType,attr"`
	DataRate    string `xml:"dataRate,attr"`
	Band        string `xml:"band,attr"`
	Power       string `xml:"power,attr"`
	Spacecraft  string `xml:"spacecraft,attr"`
	SpacecraftID string `xml:"spacecraftID,attr"`
}

type xmlDownSignal struct {
	Active      string `xml:"active,attr"`
	SignalType  string `xml:"signalType,attr"`
	DataRate    string `xml:"dataRate,attr"`
	Band        string `xml:"band,attr"`
	Power       string `xml:"power,attr"`
	Spacecraft  string `xml:"spacecraft,attr"`
	SpacecraftID string `xml:"spacecraftID,attr"`
}

type xmlTarget struct {
	Name          string `xml:"name,attr"`
	ID            string `xml:"id,attr"`
	UplegRange    string `xml:"uplegRange,attr"`
	DownlegRange  string `xml:"downlegRange,attr"`
	RTLT          string `xml:"rtlt,attr"`
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
	resp, err := c.httpClient.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("dsn fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dsn read body: %w", err)
	}

	// Use streaming decoder: <station> and <dish> are siblings under <dsn>.
	// Each <station> tag sets the current station context for subsequent <dish> elements.
	status := &Status{Timestamp: time.Now().UTC()}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	currentStation := ""

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "station":
			var station xmlStation
			if err := decoder.DecodeElement(&station, &se); err == nil {
				loc := stationNames[station.Name]
				if loc == "" {
					loc = station.FriendlyName
				}
				if loc == "" {
					loc = station.Name
				}
				currentStation = loc
			}

		case "dish":
			var dish xmlDish
			if err := decoder.DecodeElement(&dish, &se); err != nil {
				continue
			}
			if !dishTracksEM2(dish) {
				continue
			}

			d := Dish{
				Name:      dish.Name,
				Azimuth:   parseFloat(dish.Azimuth),
				Elevation: parseFloat(dish.Elevation),
				Station:   currentStation,
			}

			for _, us := range dish.UpSignals {
				if us.Spacecraft != TargetSpacecraft {
					continue
				}
				d.UpSignals = append(d.UpSignals, Signal{
					Active:     us.Active == "true",
					SignalType: us.SignalType,
					DataRate:   parseFloat(us.DataRate),
					Band:       us.Band,
					Power:      parseFloat(us.Power),
				})
			}

			for _, ds := range dish.DownSignals {
				if ds.Spacecraft != TargetSpacecraft {
					continue
				}
				d.DownSignals = append(d.DownSignals, Signal{
					Active:     ds.Active == "true",
					SignalType: ds.SignalType,
					DataRate:   parseFloat(ds.DataRate),
					Band:       ds.Band,
					Power:      parseFloat(ds.Power),
				})
			}

			for _, t := range dish.Targets {
				if t.Name != TargetSpacecraft {
					continue
				}
				target := Target{
					Name:         t.Name,
					UplegRange:   parseFloat(t.UplegRange),
					DownlegRange: parseFloat(t.DownlegRange),
					RTLT:         parseFloat(t.RTLT),
				}
				d.Targets = append(d.Targets, target)

				if target.DownlegRange > 0 {
					status.Range = target.DownlegRange
				}
				if target.RTLT > 0 {
					status.RTLT = target.RTLT
				}
			}

			status.Dishes = append(status.Dishes, d)
		}
	}

	return status, nil
}

func dishTracksEM2(dish xmlDish) bool {
	for _, t := range dish.Targets {
		if t.Name == TargetSpacecraft {
			return true
		}
	}
	for _, s := range dish.UpSignals {
		if s.Spacecraft == TargetSpacecraft {
			return true
		}
	}
	for _, s := range dish.DownSignals {
		if s.Spacecraft == TargetSpacecraft {
			return true
		}
	}
	return false
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
