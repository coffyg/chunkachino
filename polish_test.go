package chunkachino

import (
	"strings"
	"testing"
)

// --- Ellipsis behaviour ----------------------------------------------------

func TestEllipsis_AsciiDotsCollapsedIntoOneChunk(t *testing.T) {
	// Three ASCII dots should not cause three flushes. Because chunkachino
	// only flushes at a whitespace boundary, contiguous dots naturally stay
	// in one chunk. Prove it at every token granularity.
	cases := []struct {
		text string
		want []string
	}{
		{"Wait... I see.", []string{"Wait...", "I see."}},
		{"Well.... Then what?", []string{"Well....", "Then what?"}},
		{"Hmm… okay. Next.", []string{"Hmm…", "okay.", "Next."}},
		{"Three dots... then more.... then stop.", []string{"Three dots...", "then more....", "then stop."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 2, 3, 5, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

func TestEllipsis_DoesNotHangAtEndOfStream(t *testing.T) {
	// An unclosed ellipsis at end of stream must flush on Flush, not wait
	// forever for a post-terminator lookahead.
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	for _, tok := range []string{"Well", "..."} {
		_ = c.Add(tok)
	}
	tail := c.Flush()
	if tail != "Well..." {
		t.Errorf("flush hung on ellipsis: got %q want %q", tail, "Well...")
	}
}

// --- Smart / curly quotes --------------------------------------------------

func TestSmartQuotes_DoNotBreakClassification(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		// Double curly
		{"She said \u201cHello world.\u201d Then left.", []string{"She said \u201cHello world.\u201d", "Then left."}},
		// Single curly
		{"He said \u2018hi.\u2019 Then left.", []string{"He said \u2018hi.\u2019", "Then left."}},
		// ASCII straight
		{"Hi \"inline quote\" text. More.", []string{"Hi \"inline quote\" text.", "More."}},
		// Mixed: opening curly, closing ascii
		{"Say \u201chello.\" Next.", []string{"Say \u201chello.\"", "Next."}},
		// Nested brackets + quote
		{"(She said \u201chi.\u201d) Then left.", []string{"(She said \u201chi.\u201d)", "Then left."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 3, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

func TestSmartQuotes_DoNotFlushMidWord(t *testing.T) {
	// The invariant: smart quotes never cause a mid-word flush. Feed every
	// char one at a time to stress incremental state.
	text := "He\u2019s \u201charmless\u201d today."
	got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, 1))
	compareChunks(t, got, []string{text})
}

// --- URLs / emails ---------------------------------------------------------

func TestURLs_PeriodInsideDoesNotFlush(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"Visit www.example.com now. Bye.", []string{"Visit www.example.com now.", "Bye."}},
		{"Check https://example.com/x.html now. Bye.", []string{"Check https://example.com/x.html now.", "Bye."}},
		{"Go to http://a.b.c.d.e/path today. End.", []string{"Go to http://a.b.c.d.e/path today.", "End."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 3, 7, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

func TestEmails_PeriodInsideDoesNotFlush(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"Mail user@example.com today. End.", []string{"Mail user@example.com today.", "End."}},
		{"Email a.b@x.y.z now. Thanks.", []string{"Email a.b@x.y.z now.", "Thanks."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 3, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

// --- Numbers with thousands separators -------------------------------------

func TestThousandsSeparator_CommaDoesNotFlushInPhraseMode(t *testing.T) {
	// In phrase mode a standalone comma causes a flush. Inside a number
	// like 1,000 the comma has no adjacent whitespace so no flush fires.
	text := "Earned 1,000 today, spent 500 tomorrow."
	got := feed(Config{Mode: ModePhrase, MaxWords: 40}, splitEveryN(text, 1))
	compareChunks(t, got, []string{"Earned 1,000 today,", "spent 500 tomorrow."})
}

func TestDecimals_UnicodeDigitsAlsoProtectPeriod(t *testing.T) {
	// Decimal inside Arabic-Indic digits must be treated as a number, not
	// a sentence boundary. This is the reason isDigit now defers to
	// unicode.IsDigit for non-ASCII codepoints.
	text := "Pi is \u0663.\u0661\u0664 today. Math is fun."
	got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, 1))
	compareChunks(t, got, []string{
		"Pi is \u0663.\u0661\u0664 today.",
		"Math is fun.",
	})
}

// --- Stacked terminators ---------------------------------------------------

func TestStackedTerminators_CollapseToOneChunk(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"Really?! Yes.", []string{"Really?!", "Yes."}},
		{"Really??! Yes.", []string{"Really??!", "Yes."}},
		{"Wait!!! Okay.", []string{"Wait!!!", "Okay."}},
		{"Wait...? Okay.", []string{"Wait...?", "Okay."}},
		{"Wait...?! Okay.", []string{"Wait...?!", "Okay."}},
		{"Really?!?! Yes.", []string{"Really?!?!", "Yes."}},
		{"Hmm...! Next.", []string{"Hmm...!", "Next."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 2, 3, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

// --- Emoji / grapheme clusters --------------------------------------------

func TestEmoji_MultiRuneSequencesNotSplitMidCluster(t *testing.T) {
	// ZWJ joined "family" emoji, flag sequences, skin-tone modifiers — all
	// multi-rune grapheme clusters. They must never be cut mid-cluster.
	cases := []struct {
		text string
		want []string
	}{
		// ZWJ family
		{"Hi 👨\u200d👩\u200d👧 friend. Bye.", []string{"Hi 👨\u200d👩\u200d👧 friend.", "Bye."}},
		// Regional indicator flag
		{"From 🇫🇷 hello. Done.", []string{"From 🇫🇷 hello.", "Done."}},
		// Skin tone modifier
		{"Wave 👋🏽 bye. End.", []string{"Wave 👋🏽 bye.", "End."}},
		// Trailing emoji before terminator
		{"I'm happy 😊. Bye.", []string{"I'm happy 😊.", "Bye."}},
	}
	for _, tc := range cases {
		// Feed char-by-char (splitEveryN n=1) which splits ASCII 1/rune
		// but won't break a multi-byte rune into invalid UTF-8 because
		// splitEveryN operates on runes, not bytes.
		for _, n := range []int{1, 2, 3, 7, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

// --- Numbers / versions / phone numbers ------------------------------------

func TestNumbers_VersionAndPhoneSurvive(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"Version v2.3.1-alpha is out. Ship it.", []string{"Version v2.3.1-alpha is out.", "Ship it."}},
		{"Phone 555-123-4567 now. Call.", []string{"Phone 555-123-4567 now.", "Call."}},
		{"Rating 4.5/5 stars today. Nice.", []string{"Rating 4.5/5 stars today.", "Nice."}},
		{"It's 3:45 p.m. now. Go.", []string{"It's 3:45 p.m. now.", "Go."}},
	}
	for _, tc := range cases {
		for _, n := range []int{1, 3, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(tc.text, n))
			compareChunks(t, got, tc.want)
		}
	}
}

// --- Regressions: long-streaming stability --------------------------------

func TestPolish_LongStreamedParagraph(t *testing.T) {
	// A paragraph mixing many polish concerns in one run, fed one rune at
	// a time. Should never end a chunk mid-word, never hang, never split
	// a URL/email/decimal/emoji.
	text := "Mr. Smith wrote user@ex.com and said \u201cship v1.2.0 today.\u201d " +
		"He noted 1,000 users... maybe more?! " +
		"Visit https://ex.com/x.html for details. " +
		"I'm happy 😊 about it. " +
		"z.B. der Prof. Müller ist bereit."
	// Default to English: the z.B./Prof./Müller will still make it to the
	// end as one big chunk or split at real sentence ends. We don't pin
	// the exact chunking — just sanity-check: no chunk is empty, join
	// reconstructs the original, no chunk ends mid-word.
	for _, n := range []int{1, 3, 13, 100} {
		got := feed(Config{Mode: ModeSentence, MaxWords: 40}, splitEveryN(text, n))
		if len(got) == 0 {
			t.Fatalf("no chunks from text at n=%d", n)
		}
		joined := strings.Join(got, " ")
		if normSpaces(joined) != normSpaces(text) {
			t.Errorf("n=%d joined mismatch:\n got:  %q\n want: %q", n, joined, text)
		}
		for i, ch := range got {
			if ch == "" {
				t.Errorf("n=%d empty chunk at %d", n, i)
			}
			if endsMidWord(ch) {
				t.Errorf("n=%d chunk %d ended mid-word: %q", n, i, ch)
			}
		}
	}
}

// --- Compound dotted abbreviation (spaced form) ----------------------------

func TestCompoundAbbreviation_VExa_PortugueseSpaced(t *testing.T) {
	// The Portuguese "V. Exa." compound is a single-letter initial
	// followed by a spaced multi-letter abbreviation. The engine needs
	// forward-lookahead on the current initial to recognise it.
	text := "Saudações a V. Exa. hoje. Obrigado."
	for _, n := range []int{1, 2, 3, 5, 9, 100} {
		got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "pt-PT"}, splitEveryN(text, n))
		compareChunks(t, got, []string{"Saudações a V. Exa. hoje.", "Obrigado."})
	}
}

func TestCompoundAbbreviation_NeedsLookahead_FlushesOnFlush(t *testing.T) {
	// If the stream ends right after "V." we should NOT hang waiting for
	// the compound lookahead. Flush must return whatever is buffered.
	c := New(Config{Mode: ModeSentence, MaxWords: 40, Language: "pt-PT"})
	_ = c.Add("Saudações a V.")
	tail := c.Flush()
	if tail != "Saudações a V." {
		t.Errorf("flush hung on single-letter initial at EOS: got %q", tail)
	}
}
