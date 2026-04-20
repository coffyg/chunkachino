# chunkachino

A tiny, fast, pure-Go streaming chunker for the LLM-token → TTS-chunk pipeline.

The hard rule: **`Add` never returns a chunk that ends mid-word.** LLM providers love to split a single logical token like `". ho"` / `"pe you..."` across the wire. A naive chunker flushes the `"."` immediately and ships `". ho"` to the TTS engine, which dutifully reads "dot ho". chunkachino waits for the next whitespace (or end-of-stream `Flush`) before emitting, so every chunk is guaranteed word-complete.

## Install

```sh
go get github.com/coffyg/chunkachino
```

Zero runtime dependencies. Go 1.23+.

## Quick example

```go
import "github.com/coffyg/chunkachino"

c := chunkachino.New(chunkachino.Config{
    Mode:     chunkachino.ModeSentence,
    MaxWords: 30,
    Language: "en",
})

for tok := range llmStream {
    for _, chunk := range c.Add(tok) {
        tts.Speak(chunk)
    }
}
if tail := c.Flush(); tail != "" {
    tts.Speak(tail)
}
```

`Add` returns a slice (usually empty, sometimes one, occasionally several) because a single large token can carry multiple complete sentences.

## Modes

| Mode           | Flushes on                                | Abbreviation/decimal aware | Safety valve   |
| -------------- | ----------------------------------------- | -------------------------- | -------------- |
| `ModeWord`     | every `MinWords` complete words           | n/a                        | n/a            |
| `ModeSentence` | `. ! ? …` at a real sentence boundary     | yes                        | `MaxWords`     |
| `ModePhrase`   | `. ! ? , ; :` at a word boundary          | yes (for `. ! ?`)          | `MaxWords`     |

`ModeSentence` is the default recommendation for long-form TTS. `ModePhrase` gives you commas-and-colons granularity for prosody at the cost of more, shorter chunks. `ModeWord` is for very low-latency "talk as tokens arrive" pipelines.

Supported languages for the abbreviation catalog: `en` and `fr`.

### What counts as a "real" sentence end

The following do **not** trigger a flush:

- `Mr. Smith went home.` — Mr, Mrs, Ms, Dr, Prof, Sr, Jr, St, Rev, Hon, Capt, etc.
- `Eat fruit, e.g. apples.` — e.g, i.e, etc, vs, viz, cf, al, Ph.D, M.D, a.m, p.m, Jan, Feb, ...
- `Le Dr. Martin est là.` — M, Mm, Mme, Mmes, Mlle, Mgr, Me, Dr, Pr, Ste, p.ex, c.-à-d, ...
- `The price is $3.5 today.` — digit-on-both-sides decimals, version strings like `1.2.0`.
- `President J. F. Kennedy spoke.` — single-letter initial runs detected with one-rune lookahead.
- `She said "hi."` — terminator inside closing quote/bracket still flushes at the quote.

### Safety valve

In `ModeSentence` and `ModePhrase`, if the buffered word count exceeds `MaxWords` without a real terminator, chunkachino flushes at the next whitespace so a rambling LLM response never stalls the TTS pipeline. Default `MaxWords` is 30. Set to 0 to disable (the chunker will then wait forever for a terminator — not recommended in production).

## Performance

Measured on a 13th Gen Intel i7-13700:

```
BenchmarkChunkerAdd_Sentence    73_814_702    47.00 ns/op    4 B/op    0 allocs/op
BenchmarkChunkerAdd_Phrase      85_452_708    42.37 ns/op    5 B/op    0 allocs/op
BenchmarkChunkerAdd_Word        89_201_500    40.01 ns/op    5 B/op    0 allocs/op
BenchmarkChunkerFullTurn           617_582  5_857    ns/op  592 B/op   17 allocs/op
BenchmarkChunkerOneBigBlob         646_702  5_383    ns/op  704 B/op   13 allocs/op
```

Sub-50ns per `Add` on the common path and zero heap allocations. A full LLM turn (~30 tokens of text) measures around 6 microseconds end-to-end.

## Run tests

```sh
./test.sh     # verbose + race detector
./bench.sh    # benchmarks
```

Coverage: 94% of statements.

## Design notes

- `Chunker` is a `[]rune` buffer with a single forward-moving scan cursor. Abbreviation/decimal/initial checks are rune-indexed so unicode (French accents, smart quotes, ellipsis) works without special-casing.
- When disambiguation requires lookahead that hasn't arrived yet (the classic "`J.` — is this the end of a sentence or the start of `J. F. K.`?"), the scanner rewinds the cursor to the whitespace and waits for the next `Add` instead of emitting a wrong boundary.
- Not safe for concurrent use from multiple goroutines. Give each stream its own `Chunker` (or call `Reset` between messages).
