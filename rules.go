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

// isDigit reports whether r is any unicode decimal digit. The fast path
// covers ASCII 0-9 (the overwhelming majority) inline; we only defer to
// unicode.IsDigit for rarer digit scripts like Arabic-Indic (٠-٩),
// Extended Arabic-Indic (۰-۹), and Devanagari (०-९). Keeping this
// unicode-aware matters for decimals in non-Latin content — "٣.١٤"
// should be treated as a number, not a sentence boundary.
func isDigit(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	if r < 128 {
		return false
	}
	return unicode.IsDigit(r)
}

// isASCIILetter mirrors isDigit for ASCII letters; used in the single-letter
// initial rule ("J.F.K." style) where we only want to match basic latin.
func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// resolveLanguage maps one of the literal accepted language keys to the
// internal catalog key. The accepted keys are EXACTLY:
//
//   - "en-US", "es-ES", "fr-FR", "pt-PT", "de-DE"  (the five Soulkyn
//     audio locales)
//   - "en", "fr"  (legacy short aliases, kept for backwards compatibility)
//
// Matching is case-insensitive and tolerates '_' in place of '-' as a
// separator. Anything else (including "en-GB", "es", "pt", "de") falls
// back to English. There is intentionally no BCP 47 primary-subtag
// fallback tree — Flo's spec is a flat list of 5 locales + 2 aliases.
//
// Returns one of: "en", "fr", "es", "pt", "de". Never returns empty.
func resolveLanguage(lang string) string {
	if lang == "" {
		return "en"
	}
	l := strings.ToLower(lang)
	if strings.IndexByte(l, '_') >= 0 {
		l = strings.ReplaceAll(l, "_", "-")
	}
	switch l {
	case "en-us", "en":
		return "en"
	case "fr-fr", "fr":
		return "fr"
	case "es-es":
		return "es"
	case "pt-pt":
		return "pt"
	case "de-de":
		return "de"
	}
	return "en"
}

// abbreviationsForLang returns the curated abbreviation set for the given
// language key. Unknown or unsupported language strings fall back to
// English. Input is resolved through resolveLanguage; the accepted keys
// are listed there.
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
