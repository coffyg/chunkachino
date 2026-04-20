// Package chunkachino is a tiny, fast, pure-Go streaming chunker built for
// the LLM-token → TTS-chunk pipeline.
//
// The hard invariant: Add never returns a chunk that ends mid-word. LLM
// providers love to split a token like ". ho" / "pe you..." across the wire;
// a naive chunker would flush the "." immediately and ship ". ho" to the TTS
// engine, producing "dot ho" / "pe you" garbage. Chunkachino waits for the
// next whitespace (or end-of-stream Flush) before emitting, so every chunk
// is guaranteed word-complete.
//
// Three modes cover the common TTS cases:
//
//   - ModeWord:     flush every N complete words at a whitespace boundary.
//     Useful for very low-latency "talk as tokens arrive" pipelines.
//   - ModeSentence: flush on . ! ? at a real sentence boundary (abbreviation
//     and decimal aware). Safety valve flushes at MaxWords if a terminator
//     never arrives.
//   - ModePhrase:   flush on . ! ? , ; : at a word boundary. Finer
//     granularity for prosody in long-form TTS. Same safety valve.
//
// The API is intentionally minimal:
//
//	c := chunkachino.New(chunkachino.Config{
//	    Mode:     chunkachino.ModeSentence,
//	    MinWords: 3,
//	    MaxWords: 30,
//	    Language: "en",
//	})
//	for tok := range llmStream {
//	    for _, chunk := range c.Add(tok) {
//	        tts.Speak(chunk)
//	    }
//	}
//	if tail := c.Flush(); tail != "" {
//	    tts.Speak(tail)
//	}
//
// Add returns a slice (usually empty, sometimes length 1, occasionally more)
// so a single large token carrying several sentences drains cleanly. This is
// the same shape stream2sentence and other production streaming chunkers
// converged on; the "one chunk per call" API reads nicer in docs but forces
// ugly re-entrancy in callers the first time a batched token arrives.
package chunkachino

import (
	"strings"
	"unicode"
)

// Mode controls how Chunker decides when a buffered chunk is ready.
type Mode int

const (
	// ModeWord emits at a whitespace boundary once MinWords complete words
	// have accumulated. Punctuation is carried along but never triggers a
	// flush on its own.
	ModeWord Mode = iota

	// ModeSentence emits at a sentence-ending terminator (. ! ? and unicode
	// equivalents) followed by whitespace, after passing abbreviation and
	// decimal filters. A safety valve flushes at the next whitespace once
	// MaxWords is exceeded so very long terminator-less runs never stall.
	ModeSentence

	// ModePhrase emits at any phrase terminator (. ! ? , ; : and unicode
	// equivalents) followed by whitespace. Same abbreviation, decimal, and
	// safety-valve rules as ModeSentence.
	ModePhrase
)

// Config configures a Chunker. Zero values are filled with sensible
// defaults (see New).
type Config struct {
	// Mode selects the flush strategy.
	Mode Mode

	// MinWords is the minimum number of complete words a word-mode chunk
	// must contain before it can flush. Ignored in sentence/phrase mode.
	// Default: 5.
	MinWords int

	// MaxWords is the safety-valve word count in sentence/phrase mode. If
	// the buffer hits this count without seeing a terminator, the chunker
	// flushes at the next whitespace instead of stalling. Default: 30.
	// Set to 0 to disable the safety valve (chunker will wait forever).
	MaxWords int

	// Language selects the abbreviation catalog. Accepted keys are the
	// five Soulkyn audio locales exactly — "en-US", "es-ES", "fr-FR",
	// "pt-PT", "de-DE" — plus the legacy short aliases "en" and "fr".
	// Anything else falls back to English. Default: "en".
	Language string
}

// Chunker is a stateful streaming text chunker. NOT safe for concurrent use
// from multiple goroutines; each stream should own its own Chunker.
type Chunker struct {
	cfg   Config
	abbrs map[string]struct{}

	// buf holds all runes not yet emitted. We keep it as []rune (not
	// []byte) because abbreviation matching, word-boundary detection, and
	// terminator recognition are all rune-level and this keeps the hot
	// path free of repeated utf8.DecodeRune calls.
	buf []rune

	// scanPos is the index of the next rune in buf that has not yet been
	// evaluated as a potential flush boundary. Everything before scanPos
	// has been scanned; everything at scanPos or after is new.
	scanPos int
}

// New returns a ready-to-use Chunker with the given config. Missing fields
// are filled with defaults: MinWords=5, MaxWords=30, Language="en".
func New(cfg Config) *Chunker {
	if cfg.MinWords <= 0 {
		cfg.MinWords = 5
	}
	if cfg.MaxWords < 0 {
		cfg.MaxWords = 0
	}
	if cfg.MaxWords == 0 && (cfg.Mode == ModeSentence || cfg.Mode == ModePhrase) {
		cfg.MaxWords = 30
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	return &Chunker{
		cfg:   cfg,
		abbrs: abbreviationsForLang(cfg.Language),
		buf:   make([]rune, 0, 128),
	}
}

// Reset clears all buffered state but keeps the config. Use this to reuse
// a Chunker across independent messages.
func (c *Chunker) Reset() {
	c.buf = c.buf[:0]
	c.scanPos = 0
}

// Add feeds a token from the LLM stream into the chunker and returns any
// chunks that just became ready. Most tokens yield zero chunks; a token that
// completes a sentence/phrase/word-count boundary yields one; a single big
// token containing multiple complete sentences can yield several at once.
//
// Empty tokens are a no-op and return a nil slice. Whitespace-only tokens
// are accepted — they often carry the boundary signal (the space after ". ").
func (c *Chunker) Add(token string) []string {
	if token == "" {
		return nil
	}
	// Append as runes so all downstream scanning can be index-based.
	for _, r := range token {
		c.buf = append(c.buf, r)
	}
	var out []string
	for {
		chunk, ok := c.scan()
		if !ok {
			break
		}
		out = append(out, chunk)
	}
	return out
}

// Flush returns any remaining buffered text at end-of-stream. The returned
// string may be empty. After Flush the chunker is empty but NOT reset —
// config and language are preserved; call Reset to reuse for a new message.
func (c *Chunker) Flush() string {
	if len(c.buf) == 0 {
		c.scanPos = 0
		return ""
	}
	out := strings.TrimSpace(string(c.buf))
	c.buf = c.buf[:0]
	c.scanPos = 0
	if out == "" {
		return ""
	}
	return out
}

// scan walks buf from scanPos forward looking for a flush boundary. Returns
// ("", false) if no boundary is ready yet. Otherwise returns the chunk and
// shifts buf to hold only the remainder.
//
// The scan is O(n) amortized across calls because scanPos only moves
// forward; each rune is examined at most once. One subtlety: some decisions
// (is this period an initial? an abbreviation?) need lookahead into the
// NEXT word. When lookahead isn't available yet we rewind scanPos to the
// whitespace so the next Add re-evaluates it once more data arrives.
func (c *Chunker) scan() (string, bool) {
	for c.scanPos < len(c.buf) {
		r := c.buf[c.scanPos]
		wsIdx := c.scanPos
		c.scanPos++

		// A flush decision can only be made at a whitespace rune — that is
		// the only place we KNOW the preceding word is complete. Without
		// whitespace we could be staring at the left half of a word the
		// LLM is about to finish.
		if !unicode.IsSpace(r) {
			continue
		}

		// Prev-significant rune = the last non-space rune strictly before
		// this whitespace. Skip backward past closing quotes/brackets so
		// that "She said 'ok.'" flushes on the . not on the '.
		prevIdx := wsIdx - 1
		for prevIdx >= 0 && isClosingQuoteOrBracket(c.buf[prevIdx]) {
			prevIdx--
		}
		if prevIdx < 0 {
			continue
		}
		prev := c.buf[prevIdx]

		// Build a flush candidate, then check lookahead sufficiency.
		var terminator bool
		var needsLookahead bool
		switch c.cfg.Mode {
		case ModeWord:
			if countWordsInPrefix(c.buf, wsIdx) >= c.cfg.MinWords {
				return c.emit(c.scanPos)
			}

		case ModeSentence:
			if isSentenceTerminator(prev) {
				ok, need := c.classifySentenceEnd(prevIdx)
				terminator = ok
				needsLookahead = need
			}

		case ModePhrase:
			if isPhraseTerminator(prev) {
				if isSentenceTerminator(prev) {
					ok, need := c.classifySentenceEnd(prevIdx)
					terminator = ok
					needsLookahead = need
				} else {
					terminator = true
				}
			}
		}

		if needsLookahead {
			// Rewind so the next Add re-examines this whitespace with more
			// context. This is the key move for initial/abbreviation
			// detection across streaming token boundaries.
			c.scanPos = wsIdx
			return "", false
		}
		if terminator {
			return c.emit(c.scanPos)
		}
		if (c.cfg.Mode == ModeSentence || c.cfg.Mode == ModePhrase) &&
			c.cfg.MaxWords > 0 && countWordsInPrefix(c.buf, wsIdx) >= c.cfg.MaxWords {
			return c.emit(c.scanPos)
		}
	}
	return "", false
}

// countWordsInPrefix returns the number of complete words in buf[:upTo].
// A word is a maximal run of word-runes terminated by a non-word-rune.
// This is O(n) per call but is only invoked on whitespace boundaries in
// word-mode or when the safety valve is active, not per-rune — net cost
// stays linear in buffer size across a full stream.
func countWordsInPrefix(buf []rune, upTo int) int {
	count := 0
	inWord := false
	for i := 0; i < upTo && i < len(buf); i++ {
		if isWordRune(buf[i]) {
			if !inWord {
				inWord = true
			}
		} else if inWord {
			count++
			inWord = false
		}
	}
	if inWord {
		count++ // trailing word before upTo counts only if next rune is non-word; since upTo points at a whitespace, this is correct
	}
	return count
}

// emit slices out buf[:upTo], shifts buf, and resets scan/word state.
// The returned chunk has trailing whitespace trimmed (TTS engines don't
// need it) but internal whitespace is preserved exactly.
func (c *Chunker) emit(upTo int) (string, bool) {
	raw := c.buf[:upTo]
	// Trim trailing whitespace from the chunk we emit; keep the remainder
	// in buf untouched so follow-on tokens concatenate cleanly.
	end := len(raw)
	for end > 0 && unicode.IsSpace(raw[end-1]) {
		end--
	}
	// Also trim leading whitespace — important when the previous emit left
	// us pointed at a lonely space.
	start := 0
	for start < end && unicode.IsSpace(raw[start]) {
		start++
	}
	chunk := string(raw[start:end])

	// Shift: copy the unsent tail down to index 0, reslice, reset scanPos
	// so the next scan() re-walks whatever is left over. That is important:
	// a single large token can carry several complete sentences and we
	// want the second one to surface on the very next scan iteration.
	remaining := len(c.buf) - upTo
	if remaining > 0 {
		copy(c.buf, c.buf[upTo:])
	}
	c.buf = c.buf[:remaining]
	c.scanPos = 0

	if chunk == "" {
		return "", false
	}
	return chunk, true
}

// classifySentenceEnd decides whether the terminator at buf[tIdx] truly
// ends a sentence. Returns (real, needsLookahead):
//
//   - (true,  false): definite sentence end — flush now.
//   - (false, false): definite non-terminator (e.g. "Mr.", "3.14") — keep
//     scanning.
//   - (_,     true):  decision depends on runes that haven't arrived yet
//     (e.g. is this period an initial "J." waiting for "F."?). Caller
//     should rewind and wait for more input.
func (c *Chunker) classifySentenceEnd(tIdx int) (real, needsLookahead bool) {
	t := c.buf[tIdx]
	if t != '.' {
		// ! ? … and their unicode siblings always terminate a sentence.
		return true, false
	}
	return c.classifyPeriod(tIdx)
}

// classifyPeriod analyzes the period at buf[tIdx] and decides whether it
// is a real sentence terminator. Returns (real, needsLookahead) with the
// same semantics as classifySentenceEnd.
func (c *Chunker) classifyPeriod(tIdx int) (real, needsLookahead bool) {
	// Decimal / version check: digit on both sides of the period means we
	// are inside a number like "3.14" or "1.2.0" — definitely not a
	// sentence boundary.
	if tIdx > 0 && tIdx+1 < len(c.buf) {
		if isDigit(c.buf[tIdx-1]) && isDigit(c.buf[tIdx+1]) {
			return false, false
		}
	}

	// Grab the word immediately preceding the period (word runes only).
	wordEnd := tIdx
	wordStart := wordEnd
	for wordStart > 0 && isWordRune(c.buf[wordStart-1]) {
		wordStart--
	}
	if wordStart == wordEnd {
		// No word before the period (e.g. "!.") — treat as real terminator.
		return true, false
	}
	word := c.buf[wordStart:wordEnd]

	// Multi-period abbreviations such as "e.g." / "Ph.D." / "U.S.A." /
	// "V. Exa." / "z. B." walk backward through inner ". <letter>"
	// segments. The walk tolerates at most one ASCII space between an
	// inner dot and the previous word so both "V.Exa." (squeezed) and
	// "V. Exa." (spaced) forms share the same lookup key — the abbrev
	// map is built space-free ("v.exa") and the key passed to the
	// classifier is likewise built with spaces squeezed out.
	extStart := wordStart
	sawSpaceBetween := false
	for {
		// Step past an optional ASCII space (at most one) between the
		// current extStart and a candidate inner dot.
		probe := extStart - 1
		skippedSpace := false
		if probe >= 0 && c.buf[probe] == ' ' {
			probe--
			skippedSpace = true
		}
		if probe < 1 || c.buf[probe] != '.' {
			break
		}
		innerDot := probe
		newStart := innerDot
		for newStart > 0 && isWordRune(c.buf[newStart-1]) {
			newStart--
		}
		if newStart == innerDot {
			break
		}
		extStart = newStart
		if skippedSpace {
			sawSpaceBetween = true
		}
	}
	if extStart < wordStart {
		combined := c.buf[extStart:wordEnd]
		if sawSpaceBetween {
			// Build a space-squeezed copy for the abbreviation lookup so
			// "V. Exa." and "V.Exa." share the same key.
			squeezed := make([]rune, 0, len(combined))
			for _, r := range combined {
				if r == ' ' {
					continue
				}
				squeezed = append(squeezed, r)
			}
			combined = squeezed
		}
		if isKnownAbbreviation(combined, c.abbrs) {
			return false, false
		}
	}

	if isKnownAbbreviation(word, c.abbrs) {
		return false, false
	}

	// Single-letter initial like "J." — need to look both back (prior dot
	// confirms we are mid-run) and forward (next uppercase+dot confirms an
	// incoming initial). When lookahead hasn't arrived yet, we return the
	// needsLookahead flag so the caller can rewind and wait.
	if len(word) == 1 && isASCIILetter(word[0]) {
		// Back-look: is there a "<letter>." immediately before this word
		// (ignoring spaces)?
		i := wordStart - 1
		for i >= 0 && c.buf[i] == ' ' {
			i--
		}
		if i >= 0 && c.buf[i] == '.' {
			// Previous token also ended in a period — likely an initial.
			return false, false
		}

		// Forward look: if the next non-space rune hasn't arrived, we
		// can't tell whether this is a real end or the start of "J. F."
		// style. Ask for lookahead.
		j := tIdx + 1
		for j < len(c.buf) && unicode.IsSpace(c.buf[j]) {
			j++
		}
		if j >= len(c.buf) {
			return false, true // need more input to decide
		}
		if unicode.IsUpper(c.buf[j]) {
			// Walk the candidate next word.
			k := j
			for k < len(c.buf) && isWordRune(c.buf[k]) {
				k++
			}
			if k-j == 1 {
				// Next "word" is a single uppercase letter. Need to know
				// what follows it.
				if k >= len(c.buf) {
					return false, true
				}
				if c.buf[k] == '.' {
					return false, false // "J. F." confirmed initial run
				}
			} else if k-j > 1 {
				// Next word is multi-letter capitalized; it might form a
				// compound dotted abbreviation with the current single-
				// letter initial (e.g. Portuguese "V. Exa.", German
				// "z. B." spaced form when the second token is multiletter).
				// If the next rune after the word is a period AND the
				// space-squeezed "<letter>.<word>" is a known abbreviation,
				// this period is NOT a sentence end.
				if k >= len(c.buf) {
					// Don't know yet if a period follows the next word.
					return false, true
				}
				if c.buf[k] == '.' {
					// Build lowercase "<letter>.<word>" (no intervening space).
					const stackN = 32
					var buf [stackN]rune
					var key []rune
					need := 1 + 1 + (k - j)
					if need <= stackN {
						key = buf[:0]
					} else {
						key = make([]rune, 0, need)
					}
					key = append(key, unicode.ToLower(word[0]), '.')
					for p := j; p < k; p++ {
						key = append(key, unicode.ToLower(c.buf[p]))
					}
					if _, ok := c.abbrs[string(key)]; ok {
						return false, false
					}
				}
			}
			// Next word is a regular capitalized word (start of next
			// sentence) — this period IS a sentence end.
		}
	}

	return true, false
}
