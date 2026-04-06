package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"artemis/internal/horizons"
	"artemis/internal/youtubecaps"
)

func TestLoadDSNHistoryPrunesStaleSamples(t *testing.T) {
	now := time.Date(2026, time.April, 3, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "dsn-history.json")

	err := saveDSNHistory(path, dsnHistoryState{
		rangeValues: []float64{1, 2},
		rangeTimes: []time.Time{
			now.Add(-dsnHistoryMaxAge() - time.Minute),
			now.Add(-time.Minute),
		},
		rtltValues: []float64{3},
		rtltTimes:  []time.Time{now.Add(-2 * time.Minute)},
		rateValues: []float64{4},
		rateTimes:  []time.Time{now.Add(-dsnHistoryMaxAge() - time.Second)},
	})
	if err != nil {
		t.Fatalf("saveDSNHistory() error = %v", err)
	}

	got, err := loadDSNHistory(path, now)
	if err != nil {
		t.Fatalf("loadDSNHistory() error = %v", err)
	}

	if len(got.rangeValues) != 1 || got.rangeValues[0] != 2 {
		t.Fatalf("rangeValues = %#v, want [2]", got.rangeValues)
	}
	if len(got.rtltValues) != 1 || got.rtltValues[0] != 3 {
		t.Fatalf("rtltValues = %#v, want [3]", got.rtltValues)
	}
	if len(got.rateValues) != 0 {
		t.Fatalf("rateValues = %#v, want empty after pruning", got.rateValues)
	}
}

func TestMergeFloatHistorySkipsSingleOverlap(t *testing.T) {
	got := mergeFloatHistory([]float64{1, 2, 3}, []float64{3, 4}, 10)
	want := []float64{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("len(mergeFloatHistory()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mergeFloatHistory()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestMergeVectorHistorySkipsSingleOverlap(t *testing.T) {
	got := mergeVectorHistory(
		[]horizons.Vector3{{X: 1}, {X: 2}},
		[]horizons.Vector3{{X: 2}, {X: 3}},
		10,
	)
	want := []horizons.Vector3{{X: 1}, {X: 2}, {X: 3}}
	if len(got) != len(want) {
		t.Fatalf("len(mergeVectorHistory()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mergeVectorHistory()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTranscriptChunkForStatusAddsOnlyNewCaptionLines(t *testing.T) {
	previous := &youtubecaps.Status{
		StreamTitle: "NASA Live",
		VideoID:     "abc123",
		Lines:       []string{"line one", "line two"},
	}
	current := &youtubecaps.Status{
		StreamTitle: "NASA Live",
		VideoID:     "abc123",
		Lines:       []string{"line two", "line three"},
		Timestamp:   time.Date(2026, time.April, 5, 12, 30, 0, 0, time.FixedZone("CEST", 2*3600)),
	}

	got := transcriptChunkForStatus(previous, current, true)
	if len(got) != 1 {
		t.Fatalf("len(transcriptChunkForStatus()) = %d, want 1", len(got))
	}
	want := "[2026-04-05 12:30:00 CEST] line three"
	if got[0] != want {
		t.Fatalf("transcriptChunkForStatus()[0] = %q, want %q", got[0], want)
	}
}

func TestLoadTranscriptArchiveReadsPlainTextLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live-captions.log")
	body := "=== 2026-04-05 12:00:00 UTC ===\n[2026-04-05 12:00:01 UTC] line one\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := loadTranscriptArchive(path)
	if err != nil {
		t.Fatalf("loadTranscriptArchive() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(loadTranscriptArchive()) = %d, want 2", len(got))
	}
	if got[1] != "[2026-04-05 12:00:01 UTC] line one" {
		t.Fatalf("loadTranscriptArchive()[1] = %q", got[1])
	}
}

func TestDefaultTranscriptPathUsesWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", tempDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore Chdir(%q) error = %v", wd, err)
		}
	})

	got, err := defaultTranscriptPath()
	if err != nil {
		t.Fatalf("defaultTranscriptPath() error = %v", err)
	}

	want := filepath.Join(tempDir, transcriptFilename)
	gotDirResolved, err := filepath.EvalSymlinks(filepath.Dir(got))
	if err != nil {
		t.Fatalf("EvalSymlinks(filepath.Dir(got)) error = %v", err)
	}
	wantDirResolved, err := filepath.EvalSymlinks(filepath.Dir(want))
	if err != nil {
		t.Fatalf("EvalSymlinks(filepath.Dir(want)) error = %v", err)
	}
	if filepath.Base(got) != transcriptFilename {
		t.Fatalf("filepath.Base(defaultTranscriptPath()) = %q, want %q", filepath.Base(got), transcriptFilename)
	}
	if gotDirResolved != wantDirResolved {
		t.Fatalf("defaultTranscriptPath() dir = %q, want %q", gotDirResolved, wantDirResolved)
	}
}
