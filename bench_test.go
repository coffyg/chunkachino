package chunkachino

import (
	"strings"
	"testing"
)

// benchText is roughly what an LLM turn looks like coming out of Soulkyn:
// a few sentences, some punctuation, a decimal, an abbreviation.
const benchText = `Mr. Smith paid $3.5 today, then left the building. ` +
	`Dr. Who is real and so is Ph.D. Jane. ` +
	`alright, just a second, hope you like what you see. ` +
	`Version 1.2.0 is released today, enjoy it. ` +
	`She ran 26.2 miles yesterday, impressive result. ` +
	`President J. F. Kennedy spoke briefly to the crowd. ` +
	`The match is cats vs. dogs today, come watch. ` +
	`Really?! Yes, absolutely, without any doubt at all.`

// splitForBench mimics a typical LLM streaming tokenizer: short tokens of
// 1-5 runes, not aligned with word boundaries.
func splitForBench(s string) []string {
	rs := []rune(s)
	var out []string
	i := 0
	step := 3
	for i < len(rs) {
		j := i + step
		if j > len(rs) {
			j = len(rs)
		}
		out = append(out, string(rs[i:j]))
		i = j
		// Cycle step size to simulate real LLM streams.
		step++
		if step > 5 {
			step = 1
		}
	}
	return out
}

var benchTokens = splitForBench(benchText)

func BenchmarkChunkerAdd_Sentence(b *testing.B) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok := benchTokens[i%len(benchTokens)]
		_ = c.Add(tok)
		if i%len(benchTokens) == len(benchTokens)-1 {
			_ = c.Flush()
			c.Reset()
		}
	}
}

func BenchmarkChunkerAdd_Phrase(b *testing.B) {
	c := New(Config{Mode: ModePhrase, MaxWords: 40})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok := benchTokens[i%len(benchTokens)]
		_ = c.Add(tok)
		if i%len(benchTokens) == len(benchTokens)-1 {
			_ = c.Flush()
			c.Reset()
		}
	}
}

func BenchmarkChunkerAdd_Word(b *testing.B) {
	c := New(Config{Mode: ModeWord, MinWords: 5})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok := benchTokens[i%len(benchTokens)]
		_ = c.Add(tok)
		if i%len(benchTokens) == len(benchTokens)-1 {
			_ = c.Flush()
			c.Reset()
		}
	}
}

// BenchmarkChunkerFullTurn measures the entire stream end-to-end: reset +
// feed every token + flush. More representative of the per-LLM-turn cost
// than a per-Add microbench.
func BenchmarkChunkerFullTurn(b *testing.B) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Reset()
		for _, tok := range benchTokens {
			_ = c.Add(tok)
		}
		_ = c.Flush()
	}
}

// BenchmarkChunkerOneBigBlob measures the "everything arrived at once" edge
// case where Add must internally drain multiple chunks.
func BenchmarkChunkerOneBigBlob(b *testing.B) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40})
	blob := strings.Join(benchTokens, "")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Reset()
		_ = c.Add(blob)
		_ = c.Flush()
	}
}

// --- Per-locale hot-path benchmarks. The abbreviation map size differs
// across locales; these guard against one large locale silently tanking
// the zero-alloc hot path. Each uses a paragraph written natively in
// that language so the realistic abbreviation distribution shows up.

var benchTextES = `El Sr. García vive aquí, p.ej. en Madrid. ` +
	`La Sra. Pérez compró pan, leche, etc. ayer. ` +
	`Ud. y Uds. deben firmar, por favor. ` +
	`Telefónica S.A. cotiza en EE.UU. también.`

var benchTextPT = `O Prof. Dr. Santos chegou. ` +
	`V. Exa. foi muito gentil ontem. ` +
	`Compre fruta, p.ex. maçãs ou peras, por favor. ` +
	`Ver pág. 15, vol. 2 da colecção. ` +
	`A ACME Lda. abriu no séc. XX.`

var benchTextDE = `Hr. Schmidt und Fr. Weber sind z.B. hier. ` +
	`Der Dr. Braun, d.h. u.a. der Leiter, kommt ca. bald. ` +
	`Die Firma GmbH bzw. die AG, ggf. evtl. beide. ` +
	`200 v.Chr. bis 300 n.Chr. passierte viel.`

var benchTokensES = splitForBench(benchTextES)
var benchTokensPT = splitForBench(benchTextPT)
var benchTokensDE = splitForBench(benchTextDE)

func benchLocale(b *testing.B, lang string, tokens []string) {
	c := New(Config{Mode: ModeSentence, MaxWords: 40, Language: lang})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok := tokens[i%len(tokens)]
		_ = c.Add(tok)
		if i%len(tokens) == len(tokens)-1 {
			_ = c.Flush()
			c.Reset()
		}
	}
}

func BenchmarkChunkerAdd_Spanish(b *testing.B)    { benchLocale(b, "es-ES", benchTokensES) }
func BenchmarkChunkerAdd_Portuguese(b *testing.B) { benchLocale(b, "pt-PT", benchTokensPT) }
func BenchmarkChunkerAdd_German(b *testing.B)     { benchLocale(b, "de-DE", benchTokensDE) }
