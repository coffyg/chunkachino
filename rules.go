package chunkachino

import (
	"strings"
	"unicode"
)

// isSentenceTerminator reports whether r is a primary sentence-ending
// punctuation mark (period, exclamation, question, or their unicode
// full-width equivalents commonly seen in multilingual LLM output).
func isSentenceTerminator(r rune) bool {
	switch r {
	case '.', '!', '?', '。', '！', '？', '…':
		return true
	}
	return false
}

// isPhraseTerminator reports whether r is a phrase-level punctuation mark
// (anything a TTS engine would naturally take a breath on). Includes
// everything isSentenceTerminator matches plus commas, semicolons, colons,
// and unicode equivalents.
func isPhraseTerminator(r rune) bool {
	if isSentenceTerminator(r) {
		return true
	}
	switch r {
	case ',', ';', ':', '，', '；', '：':
		return true
	}
	return false
}

// isClosingQuoteOrBracket reports whether r is a closing quotation mark or
// bracket that can legitimately trail a sentence terminator (e.g. "She said
// 'hi.'" — the terminator is logically the period but the boundary is after
// the closing quote).
func isClosingQuoteOrBracket(r rune) bool {
	switch r {
	case '"', '\'', ')', ']', '}', '»', '›', '”', '’', '」', '』':
		return true
	}
	return false
}

// isWordRune reports whether r can participate in a word token. Letters,
// digits, apostrophes (contractions), and hyphens count. This is NOT the
// same as "non-whitespace" — punctuation is explicitly excluded so we can
// detect "Mr." as the word "Mr" followed by a period.
func isWordRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	// Apostrophes used inside contractions / possessives.
	switch r {
	case '\'', '’', '‘', '-':
		return true
	}
	return false
}

// isDigit is a tiny helper over unicode.IsDigit kept local for clarity and
// a hair of inlining help.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isASCIILetter mirrors isDigit for ASCII letters; used in the single-letter
// initial rule ("J.F.K." style) where we only want to match basic latin.
func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// resolveLanguage normalizes a BCP 47 (or looser) language tag down to the
// primary subtag used as a key into the abbreviation catalog. The resolution
// is case-insensitive and accepts either '-' or '_' as the subtag separator.
//
// Fallback chain:
//  1. exact lowercase match against a known full tag ("pt-pt", "fr-fr", ...)
//  2. primary subtag match ("pt-PT" → "pt", "en-GB" → "en")
//  3. "en" as the ultimate default
//
// Returns the catalog key ("en", "fr", "es", "pt", "de"). Never returns an
// empty string.
func resolveLanguage(lang string) string {
	if lang == "" {
		return "en"
	}
	l := strings.ToLower(lang)
	l = strings.ReplaceAll(l, "_", "-")

	// 1. Exact full-tag matches for the five Soulkyn audio locales.
	switch l {
	case "en-us", "en-gb":
		return "en"
	case "fr-fr", "fr-ca":
		return "fr"
	case "es-es":
		return "es"
	case "pt-pt":
		return "pt"
	case "de-de":
		return "de"
	}

	// 2. Primary subtag fallback — take everything before the first '-'.
	primary := l
	if i := strings.IndexByte(l, '-'); i >= 0 {
		primary = l[:i]
	}
	switch primary {
	case "en", "fr", "es", "pt", "de":
		return primary
	}

	// 3. Default.
	return "en"
}

// abbreviationsForLang returns the curated abbreviation set for the given
// language code. Unknown languages fall back to English. Input is resolved
// through resolveLanguage so BCP 47 tags like "pt-PT" work directly.
func abbreviationsForLang(lang string) map[string]struct{} {
	switch resolveLanguage(lang) {
	case "fr":
		return frenchAbbreviations
	case "es":
		return spanishAbbreviations
	case "pt":
		return portugueseAbbreviations
	case "de":
		return germanAbbreviations
	default:
		return englishAbbreviations
	}
}

// isKnownAbbreviation reports whether the lowercase word ending just before
// a period is a known abbreviation for the configured language. The caller
// passes the raw word rune slice (WITHOUT the trailing period).
func isKnownAbbreviation(word []rune, abbrs map[string]struct{}) bool {
	if len(word) == 0 {
		return false
	}
	// Lowercase in-place into a small stack buffer when possible to avoid
	// allocation on the hot path. For longer words we fall back to a heap
	// allocation via string(lower).
	const stackN = 16
	var buf [stackN]rune
	var lower []rune
	if len(word) <= stackN {
		lower = buf[:len(word)]
	} else {
		lower = make([]rune, len(word))
	}
	for i, r := range word {
		lower[i] = unicode.ToLower(r)
	}
	_, ok := abbrs[string(lower)]
	return ok
}
