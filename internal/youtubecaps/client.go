package youtubecaps

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const liveURL = "https://www.youtube.com/watch?v=m3kR2KK8TEs"

type Status struct {
	StreamTitle string
	VideoID     string
	Live        bool
	Lines       []string
	Timestamp   time.Time
}

type Client struct {
	httpClient *http.Client
	command    string
	liveURL    string
}

func NewClient() *Client {
	path, _ := exec.LookPath("yt-dlp")
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		command:    path,
		liveURL:    liveURL,
	}
}

type ytTrack struct {
	Ext  string `json:"ext"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ytMetadata struct {
	ID                string               `json:"id"`
	Title             string               `json:"title"`
	LiveStatus        string               `json:"live_status"`
	AutomaticCaptions map[string][]ytTrack `json:"automatic_captions"`
	Subtitles         map[string][]ytTrack `json:"subtitles"`
}

func (c *Client) Fetch() (*Status, error) {
	if c.command == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	meta, err := c.resolveLiveMetadata()
	if err != nil {
		return nil, err
	}

	trackURL, err := subtitleTrackURL(meta)
	if err != nil {
		return &Status{
			StreamTitle: meta.Title,
			VideoID:     meta.ID,
			Live:        meta.LiveStatus == "is_live",
			Timestamp:   time.Now().UTC(),
		}, nil
	}

	lines, err := c.fetchCaptionLines(trackURL)
	if err != nil {
		return nil, err
	}

	return &Status{
		StreamTitle: meta.Title,
		VideoID:     meta.ID,
		Live:        meta.LiveStatus == "is_live",
		Lines:       lines,
		Timestamp:   time.Now().UTC(),
	}, nil
}

func (c *Client) resolveLiveMetadata() (*ytMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.command, "-J", "--skip-download", c.liveURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp metadata: %w", err)
	}

	var meta ytMetadata
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("yt metadata parse: %w", err)
	}
	return &meta, nil
}

func subtitleTrackURL(meta *ytMetadata) (string, error) {
	if meta == nil {
		return "", fmt.Errorf("missing metadata")
	}

	for _, tracks := range []map[string][]ytTrack{meta.AutomaticCaptions, meta.Subtitles} {
		for _, key := range []string{"en", "en-uYU-mmqFLq8", "en-JkeT_87f4cc"} {
			for _, track := range tracks[key] {
				if track.URL != "" && (track.Ext == "vtt" || strings.Contains(track.URL, "captions.webvtt") || strings.Contains(track.URL, "timedtext")) {
					return track.URL, nil
				}
			}
		}
		for lang, variants := range tracks {
			if !strings.HasPrefix(lang, "en") {
				continue
			}
			for _, track := range variants {
				if track.URL != "" && (track.Ext == "vtt" || strings.Contains(track.URL, "captions.webvtt") || strings.Contains(track.URL, "timedtext")) {
					return track.URL, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no english subtitle track found")
}

func (c *Client) fetchCaptionLines(trackURL string) ([]string, error) {
	playlist, err := c.fetchText(trackURL)
	if err != nil {
		return nil, fmt.Errorf("caption playlist fetch: %w", err)
	}

	if strings.Contains(playlist, "#EXTM3U") {
		segments := parseSubtitleSegments(playlist)
		if len(segments) == 0 {
			return nil, nil
		}
		var all []string
		start := max(0, len(segments)-4)
		for _, segmentURL := range segments[start:] {
			segment, err := c.fetchText(segmentURL)
			if err != nil {
				continue
			}
			all = append(all, parseCaptionLines(segment)...)
		}
		return captionTailLines(all, 8), nil
	}

	return captionTailLines(parseCaptionLines(playlist), 8), nil
}

func (c *Client) fetchText(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "artemis-dashboard/1.0")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %s", resp.Status)
	}
	return string(body), nil
}

func parseSubtitleSegments(m3u8 string) []string {
	lines := strings.Split(m3u8, "\n")
	var segments []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		segments = append(segments, line)
	}
	return segments
}

func parseWebVTTLines(vtt string) []string {
	var lines []string
	for _, line := range strings.Split(vtt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" ||
			line == "WEBVTT" ||
			strings.HasPrefix(line, "Kind:") ||
			strings.HasPrefix(line, "Language:") ||
			strings.Contains(line, "-->") ||
			strings.HasPrefix(line, "NOTE") ||
			strings.HasPrefix(line, "X-TIMESTAMP-MAP") {
			continue
		}
		line = sanitizeCaptionText(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func parseCaptionLines(payload string) []string {
	trimmed := strings.TrimSpace(payload)
	switch {
	case trimmed == "":
		return nil
	case strings.HasPrefix(trimmed, "WEBVTT") || strings.Contains(trimmed, "-->"):
		return parseWebVTTLines(trimmed)
	case strings.Contains(trimmed, "<transcript") || strings.Contains(trimmed, "<timedtext") || strings.Contains(trimmed, "<text") || strings.Contains(trimmed, "<p"):
		return parseTimedTextLines(trimmed)
	default:
		return parseWebVTTLines(trimmed)
	}
}

var (
	timedTextBlockPattern  = regexp.MustCompile(`(?s)<(?:text|p)\b[^>]*>(.*?)</(?:text|p)>`)
	timedTextMarkerPattern = regexp.MustCompile(`</?[\d:.]+>`)
	timedTextTagPattern    = regexp.MustCompile(`</?[^>]+>`)
)

func parseTimedTextLines(payload string) []string {
	matches := timedTextBlockPattern.FindAllStringSubmatch(payload, -1)
	if len(matches) == 0 {
		return nil
	}

	var lines []string
	for _, match := range matches {
		for _, line := range cleanupCaptionFragment(match[1]) {
			lines = append(lines, line)
		}
	}
	if len(lines) > 0 {
		return lines
	}

	// Fall back to XML token content if regex extraction fails on odd payloads.
	decoder := xml.NewDecoder(strings.NewReader(payload))
	var chunks []string
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch tok := token.(type) {
		case xml.CharData:
			chunks = append(chunks, string(tok))
		}
	}
	return cleanupCaptionFragment(strings.Join(chunks, " "))
}

func cleanupCaptionFragment(fragment string) []string {
	if fragment == "" {
		return nil
	}

	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n")
	cleaned := replacer.Replace(fragment)

	var lines []string
	for _, part := range strings.Split(cleaned, "\n") {
		part = sanitizeCaptionText(part)
		if part != "" {
			lines = append(lines, part)
		}
	}
	return lines
}

func sanitizeCaptionText(text string) string {
	if text == "" {
		return ""
	}
	text = html.UnescapeString(text)
	text = timedTextMarkerPattern.ReplaceAllString(text, "")
	text = timedTextTagPattern.ReplaceAllString(text, "")
	return strings.Join(strings.Fields(text), " ")
}

func captionTailLines(lines []string, limit int) []string {
	if limit <= 0 || len(lines) == 0 {
		return nil
	}

	var deduped []string
	last := ""
	for _, line := range lines {
		if line == "" || line == last {
			continue
		}
		deduped = append(deduped, line)
		last = line
	}
	if len(deduped) > limit {
		deduped = deduped[len(deduped)-limit:]
	}
	return deduped
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
