package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"artemis/internal/horizons"
)

const (
	dsnPersistInterval      = 30 * time.Second
	dsnHistoryFilename      = "dsn-history.json"
	historyBootstrapSpacing = 5 * time.Minute
)

type historySample struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

type persistedDSNHistory struct {
	Range []historySample `json:"range,omitempty"`
	RTLT  []historySample `json:"rtlt,omitempty"`
	Rate  []historySample `json:"rate,omitempty"`
}

type dsnHistoryState struct {
	rangeValues []float64
	rangeTimes  []time.Time
	rtltValues  []float64
	rtltTimes   []time.Time
	rateValues  []float64
	rateTimes   []time.Time
}

func dsnHistoryMaxAge() time.Duration {
	return time.Duration(maxMetricHistory) * dsnPersistInterval
}

func appendTimedMetricHistory(values []float64, times []time.Time, value float64, at time.Time, limit int) ([]float64, []time.Time) {
	values = append(values, value)
	times = append(times, at.UTC())
	if limit > 0 && len(values) > limit {
		values = values[len(values)-limit:]
		times = times[len(times)-limit:]
	}
	return values, times
}

func mergeFloatHistory(history, current []float64, limit int) []float64 {
	out := append([]float64(nil), history...)
	if len(out) > 0 && len(current) > 0 && out[len(out)-1] == current[0] {
		current = current[1:]
	}
	out = append(out, current...)
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func mergeVectorHistory(history, current []horizons.Vector3, limit int) []horizons.Vector3 {
	out := append([]horizons.Vector3(nil), history...)
	if len(out) > 0 && len(current) > 0 && out[len(out)-1] == current[0] {
		current = current[1:]
	}
	out = append(out, current...)
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func defaultDSNHistoryPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "artemis", dsnHistoryFilename), nil
}

func loadDefaultDSNHistory(now time.Time) (dsnHistoryState, error) {
	path, err := defaultDSNHistoryPath()
	if err != nil {
		return dsnHistoryState{}, err
	}
	return loadDSNHistory(path, now)
}

func loadDSNHistory(path string, now time.Time) (dsnHistoryState, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return dsnHistoryState{}, nil
		}
		return dsnHistoryState{}, err
	}

	var persisted persistedDSNHistory
	if err := json.Unmarshal(body, &persisted); err != nil {
		return dsnHistoryState{}, err
	}

	cutoff := now.UTC().Add(-dsnHistoryMaxAge())
	rangeSamples := pruneHistorySamples(persisted.Range, cutoff)
	rtltSamples := pruneHistorySamples(persisted.RTLT, cutoff)
	rateSamples := pruneHistorySamples(persisted.Rate, cutoff)

	return dsnHistoryState{
		rangeValues: samplesToValues(rangeSamples),
		rangeTimes:  samplesToTimes(rangeSamples),
		rtltValues:  samplesToValues(rtltSamples),
		rtltTimes:   samplesToTimes(rtltSamples),
		rateValues:  samplesToValues(rateSamples),
		rateTimes:   samplesToTimes(rateSamples),
	}, nil
}

func saveDefaultDSNHistory(state dsnHistoryState) error {
	path, err := defaultDSNHistoryPath()
	if err != nil {
		return err
	}
	return saveDSNHistory(path, state)
}

func saveDSNHistory(path string, state dsnHistoryState) error {
	persisted := persistedDSNHistory{
		Range: historySamples(state.rangeValues, state.rangeTimes),
		RTLT:  historySamples(state.rtltValues, state.rtltTimes),
		Rate:  historySamples(state.rateValues, state.rateTimes),
	}

	body, err := json.Marshal(persisted)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func pruneHistorySamples(samples []historySample, cutoff time.Time) []historySample {
	var pruned []historySample
	for _, sample := range samples {
		if sample.At.IsZero() || sample.At.Before(cutoff) {
			continue
		}
		pruned = append(pruned, sample)
	}
	if len(pruned) > maxMetricHistory {
		pruned = pruned[len(pruned)-maxMetricHistory:]
	}
	return pruned
}

func historySamples(values []float64, times []time.Time) []historySample {
	limit := len(values)
	if len(times) < limit {
		limit = len(times)
	}
	out := make([]historySample, 0, limit)
	for i := 0; i < limit; i++ {
		if times[i].IsZero() {
			continue
		}
		out = append(out, historySample{At: times[i].UTC(), Value: values[i]})
	}
	return out
}

func samplesToValues(samples []historySample) []float64 {
	out := make([]float64, 0, len(samples))
	for _, sample := range samples {
		out = append(out, sample.Value)
	}
	return out
}

func samplesToTimes(samples []historySample) []time.Time {
	out := make([]time.Time, 0, len(samples))
	for _, sample := range samples {
		out = append(out, sample.At.UTC())
	}
	return out
}
