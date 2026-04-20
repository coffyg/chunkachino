package chunkachino

import (
	"strings"
	"testing"
	"unicode"
)

// feed streams the given text through a fresh Chunker one token at a time
// using the caller-supplied split strategy, then flushes. Returns all
// chunks in order. Used everywhere to exercise incremental correctness.
func feed(cfg Config, tokens []string) []string {
	c := New(cfg)
	var out []string
	for _, t := range tokens {
		out = append(out, c.Add(t)...)
	}
	if tail := c.Flush(); tail != "" {
		out = append(out, tail)
	}
	return out
}

// splitEveryN breaks s into runs of N runes. Handy for simulating LLM token
// boundaries that don't align with word boundaries.
func splitEveryN(s string, n int) []string {
	if n <= 0 {
		n = 1
	}
	rs := []rune(s)
	var out []string
	for i := 0; i < len(rs); i += n {
		j := i + n
		if j > len(rs) {
			j = len(rs)
		}
		out = append(out, string(rs[i:j]))
	}
	return out
}

// joinedEquals asserts that the concatenated chunks equal the expected
// whole text, modulo internal whitespace collapse (chunkachino trims
// trailing whitespace off each chunk). We reconstitute by joining on a
// single space.
func joinedEquals(t *testing.T, chunks []string, want string) {
	t.Helper()
	got := strings.Join(chunks, " ")
	// Normalize all runs of whitespace to a single space on both sides so
	// the assertion isn't brittle against trimming behavior.
	got = normSpaces(got)
	want = normSpaces(want)
	if got != want {
		t.Errorf("joined mismatch\n got:  %q\n want: %q", got, want)
	}
}

func normSpaces(s string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

// --- Mid-word protection (the whole reason this lib exists) ----------------

func TestMidWordProtection_VendorFailureCase(t *testing.T) {
	// The exact tokenization the TTS vendor complained about. A naive
	// chunker splits ". ho" → emits ". ho" → vendor reads "dot ho".
	tokens := []string{"alright just a second then", ". ho", "pe you like what you see."}
	chunks := feed(Config{Mode: ModeSentence, MaxWords: 40}, tokens)

	for i, ch := range chunks {
		if endsMidWord(ch) {
			t.Fatalf("chunk %d ended mid-word: %q", i, ch)
		}
	}
	joinedEquals(t, chunks, "alright just a second then. hope you like what you see.")
}

func TestMidWordProtection_TerminatorSplitFromNextWord(t *testing.T) {
	// Terminator arrives alone, followed by a leading-space+half-word
	// token. Never emit until the word after the period finishes.
	cases := [][]string{
		{"Hi there", ".", " fr", "iend."},
		{"Hi there.", " fr", "iend."},
		{"Hi there.", " fr", "iend", "."},
		{"Hi", " ", "there", ".", " ", "friend", "."},
	}
	for _, toks := range cases {
		chunks := feed(Config{Mode: ModeSentence, MaxWords: 40}, toks)
		for i, ch := range chunks {
			if endsMidWord(ch) {
				t.Fatalf("tokens=%v chunk %d=%q ended mid-word", toks, i, ch)
			}
		}
		joinedEquals(t, chunks, "Hi there. friend.")
	}
}

func endsMidWord(chunk string) bool {
	rs := []rune(chunk)
	if len(rs) == 0 {
		return false
	}
	last := rs[len(rs)-1]
	// Allowed endings: word rune only if it's genuinely the end of a word
	// (there's no half-word test from string alone, but we *can* check
	// that if the chunk ends in a letter, the text itself has to look
	// sane — i.e. the final "word" has length ≥ 1 and there's no trailing
	// whitespace). This test helper flags the specific failure shape of
	// ". ho" where a terminator is NOT the last rune but the chunk ends
	// mid-alphabetic-run; that case is impossible because chunkachino only
	// flushes on whitespace or EOF.
	//
	// Practically we flag: chunk ends with a letter/digit that is NOT
	// preceded by the same-word cluster (i.e. len < 2 is suspicious when
	// the chunk contains terminators earlier). Simpler: ending on a letter
	// is fine in itself — Flush can legitimately return a tail like "ok".
	// The real bug we're defending against is ending in the middle of a
	// punctuation-word split, e.g. ". ho". Assert that.
	_ = last
	// Specifically flag: chunk contains a sentence terminator followed by
	// a single-space followed by a short letter fragment at end-of-chunk.
	for i := 0; i < len(rs)-2; i++ {
		if !isSentenceTerminator(rs[i]) {
			continue
		}
		if rs[i+1] != ' ' {
			continue
		}
		// Fragment from i+2 to end; if it's all letters and < 3 runes it's
		// suspicious.
		frag := rs[i+2:]
		allLetters := true
		for _, r := range frag {
			if !unicode.IsLetter(r) {
				allLetters = false
				break
			}
		}
		if allLetters && len(frag) > 0 && len(frag) < 3 {
			return true
		}
	}
	return false
}

// --- Abbreviations ---------------------------------------------------------

func TestAbbreviations_English(t *testing.T) {
	cases := []struct {
		name   string
		tokens []string
		want   []string
	}{
		{
			name:   "Mr. does not split",
			tokens: splitEveryN("Mr. Smith went home. Goodbye.", 3),
			want:   []string{"Mr. Smith went home.", "Goodbye."},
		},
		{
			name:   "Dr. does not split",
			tokens: splitEveryN("Dr. Who is real. End.", 4),
			want:   []string{"Dr. Who is real.", "End."},
		},
		{
			name:   "e.g. mid-sentence",
			tokens: splitEveryN("Eat fruit, e.g. apples and pears. Then rest.", 5),
			want:   []string{"Eat fruit, e.g. apples and pears.", "Then rest."},
		},
		{
			name:   "Ph.D. after name",
			tokens: splitEveryN("Jane has a Ph.D. in physics. Amazing.", 4),
			want:   []string{"Jane has a Ph.D. in physics.", "Amazing."},
		},
		{
			name:   "vs. in title",
			tokens: splitEveryN("The match is cats vs. dogs today. Fun.", 4),
			want:   []string{"The match is cats vs. dogs today.", "Fun."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "en"}, tc.tokens)
			compareChunks(t, got, tc.want)
		})
	}
}

func TestAbbreviations_French(t *testing.T) {
	cases := []struct {
		name   string
		tokens []string
		want   []string
	}{
		{
			name:   "M. Dupont",
			tokens: splitEveryN("M. Dupont est arrivé tard. Bienvenue.", 3),
			want:   []string{"M. Dupont est arrivé tard.", "Bienvenue."},
		},
		{
			name:   "Mme. Dubois",
			tokens: splitEveryN("Mme. Dubois parle vite. D'accord.", 4),
			want:   []string{"Mme. Dubois parle vite.", "D'accord."},
		},
		{
			name:   "Dr. in french",
			tokens: splitEveryN("Le Dr. Martin est là. Parfait.", 3),
			want:   []string{"Le Dr. Martin est là.", "Parfait."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "fr"}, tc.tokens)
			compareChunks(t, got, tc.want)
		})
	}
}

// --- Decimals --------------------------------------------------------------

func TestDecimals(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		want   []string
	}{
		{
			name: "dollar amount",
			text: "The price is $3.5 today. Buy now.",
			want: []string{"The price is $3.5 today.", "Buy now."},
		},
		{
			name: "pi",
			text: "3.14 is pi approximately. Math is fun.",
			want: []string{"3.14 is pi approximately.", "Math is fun."},
		},
		{
			name: "version string",
			text: "Version 1.2.0 is released today. Enjoy.",
			want: []string{"Version 1.2.0 is released today.", "Enjoy."},
		},
		{
			name: "decimal mid-sentence",
			text: "She ran 26.2 miles yesterday. Impressive.",
			want: []string{"She ran 26.2 miles yesterday.", "Impressive."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// exercise several tokenizations so we trust the incremental path
			for _, n := range []int{1, 2, 3, 5, 7, 1000} {
				got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
				compareChunks(t, got, tc.want)
			}
		})
	}
}

// --- Phrase mode -----------------------------------------------------------

func TestPhraseMode_ThreeChunks(t *testing.T) {
	text := "alright, just a second, hope you like it."
	got := feed(Config{Mode: ModePhrase, MaxWords: 40}, splitEveryN(text, 3))
	want := []string{"alright,", "just a second,", "hope you like it."}
	compareChunks(t, got, want)
}

func TestPhraseMode_StillRespectsAbbrDecimals(t *testing.T) {
	text := "Mr. Smith paid $3.5, then left."
	got := feed(Config{Mode: ModePhrase, MaxWords: 40}, splitEveryN(text, 2))
	want := []string{"Mr. Smith paid $3.5,", "then left."}
	compareChunks(t, got, want)
}

// --- Sentence mode: collapsing what phrase mode would split ---------------

func TestSentenceMode_SingleChunkForCommaSentence(t *testing.T) {
	text := "alright, just a second, hope you like it."
	got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, 3))
	compareChunks(t, got, []string{"alright, just a second, hope you like it."})
}

// --- Safety valve ----------------------------------------------------------

func TestSafetyValve_LongSentenceFlushes(t *testing.T) {
	// 50-word run without a terminator. Expect flushes at whitespace
	// boundaries once MaxWords=10 is exceeded. The join of all chunks
	// must equal the original.
	words := strings.Fields(strings.Repeat("alpha ", 50))
	text := strings.Join(words, " ") + "." // terminator at the very end
	cfg := Config{Mode: ModeSentence, MaxWords: 10}
	got := feed(cfg, splitEveryN(text, 4))
	if len(got) < 4 {
		t.Fatalf("expected safety-valve to split long run, got %d chunks: %v", len(got), got)
	}
	joined := strings.Join(got, " ")
	if normSpaces(joined) != normSpaces(text) {
		t.Errorf("joined != original\n got:  %q\n want: %q", joined, text)
	}
	// None should end mid-word.
	for i, ch := range got {
		if endsMidWord(ch) {
			t.Fatalf("chunk %d ended mid-word: %q", i, ch)
		}
	}
}

func TestSafetyValve_NoTerminatorEver(t *testing.T) {
	text := strings.TrimSpace(strings.Repeat("beta ", 40)) // 40 words, no punctuation
	cfg := Config{Mode: ModeSentence, MaxWords: 8}
	got := feed(cfg, splitEveryN(text, 5))
	if len(got) < 4 {
		t.Fatalf("expected multi-chunk output, got %v", got)
	}
	joinedEquals(t, got, text)
}

// --- Word mode -------------------------------------------------------------

func TestWordMode_FiveWords(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven"
	got := feed(Config{Mode: ModeWord, MinWords: 5}, splitEveryN(text, 3))
	// Expect at least two chunks (first with >=5 words, second with rest).
	if len(got) < 2 {
		t.Fatalf("want >=2 chunks, got %v", got)
	}
	for _, ch := range got[:len(got)-1] {
		if countWords(ch) < 5 {
			t.Errorf("chunk %q has fewer than 5 words", ch)
		}
	}
	joinedEquals(t, got, text)
}

func TestWordMode_PunctuationDoesNotTriggerFlush(t *testing.T) {
	text := "hi, there. how are you doing today friend"
	got := feed(Config{Mode: ModeWord, MinWords: 4}, splitEveryN(text, 2))
	// Word mode ignores punctuation as a flush trigger; the only reason
	// to flush is reaching MinWords at a whitespace. Just check the join.
	joinedEquals(t, got, text)
	for _, ch := range got[:max(0, len(got)-1)] {
		if countWords(ch) < 4 {
			t.Errorf("chunk %q has fewer than 4 words in word-mode", ch)
		}
	}
}

func countWords(s string) int {
	return len(strings.Fields(s))
}

// --- Flush -----------------------------------------------------------------

func TestFlush_EmitsRemainder(t *testing.T) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	_ = c.Add("hello world without terminator")
	tail := c.Flush()
	if tail != "hello world without terminator" {
		t.Errorf("flush got %q", tail)
	}
}

func TestFlush_EmptyWhenNothingBuffered(t *testing.T) {
	c := New(Config{Mode: ModeSentence})
	if tail := c.Flush(); tail != "" {
		t.Errorf("expected empty, got %q", tail)
	}
}

func TestFlush_AfterEmit(t *testing.T) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	chunks := []string{}
	for _, tok := range splitEveryN("One. Two. Three", 2) {
		chunks = append(chunks, c.Add(tok)...)
	}
	if tail := c.Flush(); tail != "" {
		chunks = append(chunks, tail)
	}
	compareChunks(t, chunks, []string{"One.", "Two.", "Three"})
}

// --- Empty / whitespace noop ----------------------------------------------

func TestEmptyToken_NoOp(t *testing.T) {
	c := New(Config{Mode: ModeSentence})
	if got := c.Add(""); len(got) != 0 {
		t.Errorf("empty token should not flush, got %v", got)
	}
}

func TestWhitespaceOnlyToken_DoesNotFalseFlush(t *testing.T) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	if got := c.Add("   "); len(got) != 0 {
		t.Errorf("whitespace-only should not flush on its own, got %v", got)
	}
	if got := c.Add("hello"); len(got) != 0 {
		t.Errorf("still no terminator, should not flush, got %v", got)
	}
}

// --- Unicode ---------------------------------------------------------------

func TestUnicode_FrenchAccents(t *testing.T) {
	text := "Héllo. Ça va?"
	got := feed(Config{Mode: ModeSentence, Language: "fr", MaxWords: 40}, splitEveryN(text, 2))
	compareChunks(t, got, []string{"Héllo.", "Ça va?"})
}

func TestUnicode_SmartQuotes(t *testing.T) {
	text := "She said \u201chi.\u201d Then left."
	got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, 3))
	// The terminator is inside smart quotes; our closing-quote skip should
	// recognize the boundary at the closing 201D rune.
	compareChunks(t, got, []string{"She said \u201chi.\u201d", "Then left."})
}

// --- Incremental correctness ----------------------------------------------

func TestIncremental_SameResultAtAnyTokenGranularity(t *testing.T) {
	text := "Mr. Smith paid $3.5 today, then left. Dr. Who is real. End."
	cfg := Config{Mode: ModeSentence, MaxWords: 40}
	reference := feed(cfg, []string{text})
	for _, n := range []int{1, 2, 3, 4, 5, 7, 13, 100} {
		got := feed(cfg, splitEveryN(text, n))
		compareChunks(t, got, reference)
	}
}

func TestIncremental_PhraseMode(t *testing.T) {
	text := "alright, just a second, hope you like it. And one more, please."
	cfg := Config{Mode: ModePhrase, MaxWords: 40}
	reference := feed(cfg, []string{text})
	for _, n := range []int{1, 2, 3, 5, 11} {
		got := feed(cfg, splitEveryN(text, n))
		compareChunks(t, got, reference)
	}
}

// --- Reset -----------------------------------------------------------------

func TestReset_ClearsBuffer(t *testing.T) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	_ = c.Add("partial text no terminator")
	c.Reset()
	if tail := c.Flush(); tail != "" {
		t.Errorf("after reset flush should be empty, got %q", tail)
	}
	// should behave like fresh chunker
	chunks := []string{}
	for _, tok := range splitEveryN("Fresh start. New message.", 3) {
		chunks = append(chunks, c.Add(tok)...)
	}
	if tail := c.Flush(); tail != "" {
		chunks = append(chunks, tail)
	}
	compareChunks(t, chunks, []string{"Fresh start.", "New message."})
}

// --- Edge cases catalog ----------------------------------------------------

func TestEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		text string
		mode Mode
		want []string
	}{
		{
			name: "multiple punctuation",
			text: "Really?! Yes. Wow!",
			mode: ModeSentence,
			want: []string{"Really?!", "Yes.", "Wow!"},
		},
		{
			name: "ellipsis mid-text",
			text: "Well… I don't know. Maybe.",
			mode: ModeSentence,
			want: []string{"Well…", "I don't know.", "Maybe."},
		},
		{
			name: "trailing space",
			text: "Hello world. ",
			mode: ModeSentence,
			want: []string{"Hello world."},
		},
		{
			name: "no terminator ever",
			text: "just words without end",
			mode: ModeSentence,
			want: []string{"just words without end"},
		},
		{
			name: "initials JFK",
			text: "President J. F. Kennedy spoke. Then left.",
			mode: ModeSentence,
			want: []string{"President J. F. Kennedy spoke.", "Then left."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{1, 2, 3, 5, 9} {
				got := feed(Config{Mode: tc.mode, MaxWords: 40}, splitEveryN(tc.text, n))
				compareChunks(t, got, tc.want)
			}
		})
	}
}

// --- helpers ---------------------------------------------------------------

func compareChunks(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("chunk count mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if normSpaces(got[i]) != normSpaces(want[i]) {
			t.Errorf("chunk %d mismatch\n got:  %q\n want: %q", i, got[i], want[i])
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
