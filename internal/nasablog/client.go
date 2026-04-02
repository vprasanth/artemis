package nasablog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const blogURL = "https://www.nasa.gov/wp-json/wp/v2/nasa-blog?categories=2918&per_page=5&orderby=date&order=desc&_fields=id,date_gmt,title,excerpt,link"

type Entry struct {
	ID      int
	Time    time.Time
	Title   string
	Excerpt string
}

type Status struct {
	Entries   []Entry
	Timestamp time.Time
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type rawEntry struct {
	ID      int    `json:"id"`
	DateGMT string `json:"date_gmt"`
	Title   struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Excerpt struct {
		Rendered string `json:"rendered"`
	} `json:"excerpt"`
	Link string `json:"link"`
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func (c *Client) Fetch() (*Status, error) {
	resp, err := c.httpClient.Get(blogURL)
	if err != nil {
		return nil, fmt.Errorf("nasablog fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nasablog read: %w", err)
	}

	var raw []rawEntry
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("nasablog parse: %w", err)
	}

	status := &Status{Timestamp: time.Now().UTC()}
	for _, r := range raw {
		t, _ := time.Parse("2006-01-02T15:04:05", r.DateGMT)

		title := stripHTML(r.Title.Rendered)
		excerpt := stripHTML(r.Excerpt.Rendered)
		excerpt = strings.TrimSpace(excerpt)

		status.Entries = append(status.Entries, Entry{
			ID:      r.ID,
			Time:    t,
			Title:   title,
			Excerpt: excerpt,
		})
	}

	return status, nil
}

func stripHTML(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&#8217;", "'")
	s = strings.ReplaceAll(s, "&#8216;", "'")
	s = strings.ReplaceAll(s, "&#8220;", "\"")
	s = strings.ReplaceAll(s, "&#8221;", "\"")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&#8230;", "...")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
