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

Supported language keys for the abbreviation catalog:

| Key     | Catalog    | Notes                                        |
| ------- | ---------- | -------------------------------------------- |
| `en-US` | English    |                                              |
| `es-ES` | Spanish    | Peninsular Spanish (Sr., EE.UU., p.ej., ...) |
| `fr-FR` | French     | M., Mme., p.ex., c.-à-d., ...                |
| `pt-PT` | Portuguese | European Portuguese (Lda., V. Exa., séc., ...)|
| `de-DE` | German     | z.B., d.h., u.a., v.Chr., Str., ...          |
| `en`    | English    | legacy short alias for `en-US`               |
| `fr`    | French     | legacy short alias for `fr-FR`               |

Anything else falls back to English. There is intentionally no BCP 47 primary-subtag fallback tree — `en-GB`, `es`, `pt`, `de`, `ja-JP`, etc. all resolve to English.

### What counts as a "real" sentence end

The following do **not** trigger a flush:

- `Mr. Smith went home.` — Mr, Mrs, Ms, Dr, Prof, Sr, Jr, St, Rev, Hon, Capt, etc.
- `Eat fruit, e.g. apples.` — e.g, i.e, etc, vs, viz, cf, al, Ph.D, M.D, a.m, p.m, Jan, Feb, ...
- `Le Dr. Martin est là.` — M, Mm, Mme, Mmes, Mlle, Mgr, Me, Dr, Pr, Ste, p.ex, c.-à-d, ...
- `El Sr. García vive aquí.` — Sr, Sra, Srta, Dr, Dra, Ud, Uds, p.ej, EE.UU, S.A, pág, ...
- `O Prof. Dr. Santos chegou.` — Sr, Dr, Prof, Eng, Exmo, V. Exa, Lda, séc, pág, p.ex, ...
- `z.B. der Dr. Müller ist hier.` — z.B, d.h, u.a, bzw, usw, ggf, ca, Nr, v.Chr, Str, Bd, ...
- `The price is $3.5 today.` — digit-on-both-sides decimals, version strings like `1.2.0`.
- `Pi is ٣.١٤ approximately.` — Arabic-Indic and other Unicode digits count too.
- `President J. F. Kennedy spoke.` — single-letter initial runs detected with one-rune lookahead.
- `Cumprimentos a V. Exa. hoje.` — spaced compound abbreviations (`letter.` + `Word.`) checked as a squashed key.
- `She said "hi."` — terminator inside closing quote/bracket still flushes at the quote.
- `Visit www.example.com today.` — periods inside URLs, emails, phone numbers have no adjacent whitespace so never trigger a flush.
- `Earned 1,000 today.` — commas inside numbers don't flush in phrase mode.
- `Wait... Really?! Okay.` — stacked terminators (`...`, `?!`, `!!!`) collapse to one chunk each.
- `Hi 👨‍👩‍👧 friend.` — multi-rune emoji sequences (ZWJ, skin tones, regional indicators) are never split mid-cluster.

### Safety valve

In `ModeSentence` and `ModePhrase`, if the buffered word count exceeds `MaxWords` without a real terminator, chunkachino flushes at the next whitespace so a rambling LLM response never stalls the TTS pipeline. Default `MaxWords` is 30. Set to 0 to disable (the chunker will then wait forever for a terminator — not recommended in production).

## Performance

Measured on a 13th Gen Intel i7-13700:

```
BenchmarkChunkerAdd_Sentence      62_897_626    48.43 ns/op    5 B/op    0 allocs/op
BenchmarkChunkerAdd_Phrase        69_535_870    43.84 ns/op    6 B/op    0 allocs/op
BenchmarkChunkerAdd_Word          93_981_572    38.17 ns/op    5 B/op    0 allocs/op
BenchmarkChunkerAdd_Spanish       48_046_845    49.34 ns/op    5 B/op    0 allocs/op
BenchmarkChunkerAdd_Portuguese    44_895_519    53.27 ns/op    6 B/op    0 allocs/op
BenchmarkChunkerAdd_German        36_095_850    60.81 ns/op    7 B/op    0 allocs/op
BenchmarkChunkerFullTurn             592_878  6_203    ns/op  656 B/op   19 allocs/op
BenchmarkChunkerOneBigBlob           584_074  5_677    ns/op  768 B/op   15 allocs/op
```

Sub-100 ns per `Add` on every locale's common path and zero heap allocations. A full LLM turn (~30 tokens of text) measures around 6 microseconds end-to-end.

## Run tests

```sh
./test.sh     # verbose + race detector
./bench.sh    # benchmarks
```

Coverage: 95.6% of statements.

## Design notes

- `Chunker` is a `[]rune` buffer with a single forward-moving scan cursor. Abbreviation/decimal/initial checks are rune-indexed so unicode (French accents, smart quotes, ellipsis) works without special-casing.
- When disambiguation requires lookahead that hasn't arrived yet (the classic "`J.` — is this the end of a sentence or the start of `J. F. K.`?"), the scanner rewinds the cursor to the whitespace and waits for the next `Add` instead of emitting a wrong boundary.
- Not safe for concurrent use from multiple goroutines. Give each stream its own `Chunker` (or call `Reset` between messages).

## License

MIT — see [LICENSE](LICENSE)
