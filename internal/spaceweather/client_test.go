package spaceweather

import "testing"

func TestTailFloatSlice(t *testing.T) {
	got := tailFloatSlice([]float64{1, 2, 3, 4}, 2)
	want := []float64{3, 4}
	if len(got) != len(want) {
		t.Fatalf("len(tailFloatSlice()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tailFloatSlice()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
