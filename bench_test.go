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
