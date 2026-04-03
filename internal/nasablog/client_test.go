package nasablog

import (
	"strings"
	"testing"
)

func TestStripHTMLTextPreservesParagraphsAndLists(t *testing.T) {
	in := `<p>Flight day <strong>two</strong>.</p><ul><li>Crew wake</li><li>Systems check</li></ul>`
	got := stripHTMLText(in)

	for _, want := range []string{"Flight day two.", "- Crew wake", "- Systems check"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected cleaned text to include %q, got %q", want, got)
		}
	}
}
