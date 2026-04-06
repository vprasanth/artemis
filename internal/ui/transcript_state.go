package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"artemis/internal/youtubecaps"
)

const transcriptFilename = "live-captions.log"

func defaultTranscriptPath() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(workingDir, transcriptFilename), nil
}

func loadDefaultTranscriptArchive() ([]string, string, error) {
	path, err := defaultTranscriptPath()
	if err != nil {
		return nil, "", err
	}
	lines, err := loadTranscriptArchive(path)
	return lines, path, err
}

func loadTranscriptArchive(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func appendTranscriptChunk(path string, lines []string) error {
	if path == "" || len(lines) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
	return err
}

func (m *Model) appendTranscriptStatus(status *youtubecaps.Status) {
	if status == nil {
		return
	}
	if m.transcriptPath == "" {
		path, err := defaultTranscriptPath()
		if err != nil {
			return
		}
		m.transcriptPath = path
	}

	chunk := transcriptChunkForStatus(m.ytcapsStatus, status, len(m.transcriptArchive) > 0)
	if len(chunk) == 0 {
		return
	}
	if err := appendTranscriptChunk(m.transcriptPath, chunk); err != nil {
		return
	}
	m.transcriptArchive = append(m.transcriptArchive, chunk...)
}

func transcriptChunkForStatus(previous, current *youtubecaps.Status, hasExistingArchive bool) []string {
	if current == nil {
		return nil
	}

	streamChanged := previous == nil || previous.VideoID != current.VideoID || previous.StreamTitle != current.StreamTitle
	newLines := captionDelta(previous, current)
	if len(newLines) == 0 && !streamChanged {
		return nil
	}

	stamp := current.Timestamp.Local().Format("2006-01-02 15:04:05 MST")
	var chunk []string
	if streamChanged && hasExistingArchive {
		chunk = append(chunk, "")
	}
	if streamChanged {
		title := current.StreamTitle
		if title == "" {
			title = "NASA live stream"
		}
		chunk = append(chunk,
			fmt.Sprintf("=== %s ===", stamp),
			"Stream: "+title,
		)
		if current.VideoID != "" {
			chunk = append(chunk, "Video: "+current.VideoID)
		}
		chunk = append(chunk, "")
	}
	for _, line := range newLines {
		chunk = append(chunk, fmt.Sprintf("[%s] %s", stamp, line))
	}
	return chunk
}

func captionDelta(previous, current *youtubecaps.Status) []string {
	if current == nil || len(current.Lines) == 0 {
		return nil
	}
	if previous == nil || previous.VideoID != current.VideoID {
		return append([]string(nil), current.Lines...)
	}

	prev := previous.Lines
	curr := current.Lines
	maxOverlap := minInt(len(prev), len(curr))
	for overlap := maxOverlap; overlap > 0; overlap-- {
		matched := true
		for i := 0; i < overlap; i++ {
			if prev[len(prev)-overlap+i] != curr[i] {
				matched = false
				break
			}
		}
		if matched {
			return append([]string(nil), curr[overlap:]...)
		}
	}
	return append([]string(nil), curr...)
}
