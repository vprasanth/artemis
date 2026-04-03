package ui

import (
	"path/filepath"
	"testing"
	"time"

	"artemis/internal/horizons"
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
