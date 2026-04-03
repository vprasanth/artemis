package nasablog

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const blogURL = "https://www.nasa.gov/wp-json/wp/v2/nasa-blog?categories=2918&per_page=5&orderby=date&order=desc&_fields=id,date_gmt,title,excerpt,link"
const blogPostURL = "https://www.nasa.gov/wp-json/wp/v2/nasa-blog/%d?_fields=id,date_gmt,title,content,link"

type Entry struct {
	ID      int
	Time    time.Time
	Title   string
	Excerpt string
	Link    string
}

type Status struct {
	Entries   []Entry
	Timestamp time.Time
}

type Post struct {
	ID      int
	Time    time.Time
	Title   string
	Content string
	Link    string
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

type rawPost struct {
	ID      int    `json:"id"`
	DateGMT string `json:"date_gmt"`
	Title   struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered string `json:"rendered"`
	} `json:"content"`
	Link string `json:"link"`
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
var blockBreakRegex = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>|</\s*(p|div|section|article|h[1-6]|blockquote)\s*>`)
var listItemRegex = regexp.MustCompile(`(?i)<\s*li[^>]*>`)
var paragraphCollapseRegex = regexp.MustCompile(`\n{3,}`)

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

		title := stripHTMLCompact(r.Title.Rendered)
		excerpt := stripHTMLCompact(r.Excerpt.Rendered)
		excerpt = strings.TrimSpace(excerpt)

		status.Entries = append(status.Entries, Entry{
			ID:      r.ID,
			Time:    t,
			Title:   title,
			Excerpt: excerpt,
			Link:    r.Link,
		})
	}

	return status, nil
}

func (c *Client) FetchPost(id int) (*Post, error) {
	url := fmt.Sprintf(blogPostURL, id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("nasablog fetch post: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nasablog read post: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("nasablog post %d: http %s", id, resp.Status)
	}

	var raw rawPost
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("nasablog parse post: %w", err)
	}

	t, _ := time.Parse("2006-01-02T15:04:05", raw.DateGMT)
	return &Post{
		ID:      raw.ID,
		Time:    t,
		Title:   stripHTMLCompact(raw.Title.Rendered),
		Content: stripHTMLText(raw.Content.Rendered),
		Link:    raw.Link,
	}, nil
}

func stripHTMLCompact(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func stripHTMLText(s string) string {
	s = blockBreakRegex.ReplaceAllString(s, "\n\n")
	s = listItemRegex.ReplaceAllString(s, "\n- ")
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\t", " "))
		line = strings.Join(strings.Fields(line), " ")
		cleaned = append(cleaned, line)
	}
	text := strings.TrimSpace(strings.Join(cleaned, "\n"))
	text = paragraphCollapseRegex.ReplaceAllString(text, "\n\n")
	return text
}
