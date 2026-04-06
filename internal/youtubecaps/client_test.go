package youtubecaps

import "testing"

func TestParseSubtitleSegments(t *testing.T) {
	playlist := "#EXTM3U\n#EXTINF:5.0,\nhttps://example.test/1.webvtt\n#EXTINF:5.0,\nhttps://example.test/2.webvtt\n"
	got := parseSubtitleSegments(playlist)
	if len(got) != 2 || got[0] != "https://example.test/1.webvtt" || got[1] != "https://example.test/2.webvtt" {
		t.Fatalf("parseSubtitleSegments() = %#v", got)
	}
}

func TestParseWebVTTLines(t *testing.T) {
	vtt := "WEBVTT\n\n00:00:00.000 --> 00:00:02.000\nHello there\n\n00:00:02.000 --> 00:00:04.000\nGeneral Kenobi\n"
	got := parseWebVTTLines(vtt)
	if len(got) != 2 || got[0] != "Hello there" || got[1] != "General Kenobi" {
		t.Fatalf("parseWebVTTLines() = %#v", got)
	}
}

func TestParseCaptionLinesTimedTextCleansInlineMarkup(t *testing.T) {
	payload := `<timedtext><body><p t="1750" d="3000">mi&lt;00:00:01.750&gt;&lt;c&gt;ss&lt;/c&gt;&lt;00:00:01.783&gt;&lt;c&gt;io&lt;/c&gt;&lt;00:00:01.817&gt;&lt;c&gt;n&lt;/c&gt; specialist Jeremy Hansen</p></body></timedtext>`
	got := parseCaptionLines(payload)
	if len(got) != 1 || got[0] != "mission specialist Jeremy Hansen" {
		t.Fatalf("parseCaptionLines() = %#v", got)
	}
}

func TestParseWebVTTLinesStripsInlineCaptionTags(t *testing.T) {
	vtt := "WEBVTT\n\n00:00:03.000 --> 00:00:05.000\nL<00:00:03.518><c>oo</c><00:00:03.551><c>ki</c><00:00:03.585><c>ng</c> ahead\n"
	got := parseWebVTTLines(vtt)
	if len(got) != 1 || got[0] != "Looking ahead" {
		t.Fatalf("parseWebVTTLines() = %#v", got)
	}
}

func TestCaptionTailLinesDedupesAndLimits(t *testing.T) {
	got := captionTailLines([]string{"a", "a", "b", "c", "d"}, 3)
	want := []string{"b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("len(captionTailLines()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("captionTailLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
