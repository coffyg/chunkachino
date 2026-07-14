package chunkachino

import "testing"

// --- MinChunkWords: the TTS runt gate ---------------------------------------
//
// A lone interjection ("haha...", "heh...", "Ah.") shipped to a TTS engine as
// its own utterance gets full sentence prosody and sounds wrong. The gate
// refuses to flush a terminator-ended chunk below N complete words, so runts
// merge into the sentence that follows.

func TestMinChunkWords_RuntOpenerMergesForward(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		// The motivating case: interjection opener rides with the sentence.
		{"haha... You should have seen his face.", []string{"haha... You should have seen his face."}},
		{"heh... that counting test broke me completely.", []string{"heh... that counting test broke me completely."}},
		{"Ah. So that is how it works then.", []string{"Ah. So that is how it works then."}},
		// A healthy first sentence still flushes on its own.
		{"That test was brutal. And funny too honestly.", []string{"That test was brutal.", "And funny too honestly."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 2, 3, 5, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, MinChunkWords: 3}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

func TestMinChunkWords_ConsecutiveRuntsAccumulate(t *testing.T) {
	// Two runts jointly clearing the bar flush together as one chunk.
	text := "Ha. No. Really. Fine we can go now."
	want := []string{"Ha. No. Really.", "Fine we can go now."}
	for _, n := range []int{1, 3, 7, 100} {
		got := feed(Config{Mode: ModeSentence, MaxWords: 40, MinChunkWords: 3}, splitEveryN(text, n))
		compareChunks(t, got, want)
	}
}

func TestMinChunkWords_TrailingRuntShipsViaFlush(t *testing.T) {
	// End-of-stream runt has no following sentence to merge into — it ships
	// via Flush (documented; merging backward would delay every chunk).
	text := "That was completely wild honestly. heh..."
	want := []string{"That was completely wild honestly.", "heh..."}
	for _, n := range []int{1, 4, 100} {
		got := feed(Config{Mode: ModeSentence, MaxWords: 40, MinChunkWords: 3}, splitEveryN(text, n))
		compareChunks(t, got, want)
	}
}

func TestMinChunkWords_ZeroKeepsLegacyBehavior(t *testing.T) {
	// Regression guard: the gate is opt-in. Zero-default = pre-v0.0.2
	// behavior, byte-exact — runts flush alone.
	text := "Wait... I see."
	want := []string{"Wait...", "I see."}
	for _, n := range []int{1, 2, 100} {
		got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, n))
		compareChunks(t, got, want)
	}
}

func TestMinChunkWords_SafetyValveStillFires(t *testing.T) {
	// The MaxWords valve is orthogonal: a terminator-less run past MaxWords
	// still flushes even with the runt gate configured.
	text := "one two three four five six seven eight nine ten eleven twelve"
	got := feed(Config{Mode: ModeSentence, MaxWords: 8, MinChunkWords: 3}, splitEveryN(text, 5))
	if len(got) < 2 {
		t.Fatalf("safety valve did not fire with runt gate on: %q", got)
	}
}

func TestMinChunkWords_PhraseMode(t *testing.T) {
	// Phrase mode honors the same gate: a runt comma-fragment merges forward.
	text := "So, we finally shipped the thing, and it works."
	want := []string{"So, we finally shipped the thing,", "and it works."}
	for _, n := range []int{1, 3, 100} {
		got := feed(Config{Mode: ModePhrase, MaxWords: 40, MinChunkWords: 3}, splitEveryN(text, n))
		compareChunks(t, got, want)
	}
}

func TestMinChunkWords_WordModeIgnoresGate(t *testing.T) {
	// ModeWord is governed by MinWords; MinChunkWords must be a no-op there.
	a := feed(Config{Mode: ModeWord, MinWords: 3}, splitEveryN("one two three four five six", 4))
	b := feed(Config{Mode: ModeWord, MinWords: 3, MinChunkWords: 10}, splitEveryN("one two three four five six", 4))
	compareChunks(t, b, a)
}
