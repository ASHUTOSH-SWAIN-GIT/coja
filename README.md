# Coja

Coja is a local Wikipedia search engine in Go.

It currently supports:
- Streaming Wikipedia dump parsing
- Wikitext cleanup
- Tokenization + stemming + stopword filtering
- Fielded inverted index (`title`, `intro`, `body`) + weighted BM25 ranking
- Phrase-aware ranking (quoted and soft phrase matching)
- Segment-based on-disk persistence (`.gob`)
- Startup loading from persisted segments
- HTTP search API and a minimal browser UI
- Offline benchmark harness (`MRR@10`, `nDCG@10`, `Recall@50`)

## Current Architecture

### Indexing pipeline

1. Input source is either:
- a `.bz2` dump path (Go can read it directly), or
- stdin (`-`), typically fed by `pbzip2 -dc` for parallel decompression

2. Parser streams XML pages sequentially.
3. Worker pool processes pages in parallel:
- strip wikitext
- extract intro snippet from cleaned text
- tokenize `title`, `intro`, and `body`
- stem/filter terms
- emit typed `{docID, title, titleTokens, introTokens, bodyTokens}` work items

4. Single collector goroutine updates in-memory index.
5. Collector checkpoints every `N` docs into segment files:
- `segment_000001.gob`, `segment_000002.gob`, ...
- `manifest.json` with metadata

### Search pipeline

1. Server loads `manifest.json`.
2. Server loads all segment files into memory and merges them.
3. Query is parsed into terms + phrases:
- quoted phrases: `"..."` become explicit phrase constraints/boosts
- unquoted multi-term query also gets a soft phrase boost
4. Weighted fielded BM25 ranks candidates:
- `score = w_title*BM25(title) + w_intro*BM25(intro) + w_body*BM25(body) + phrase_boost`
5. Retrieval uses strict all-term matching first, then partial-match fallback if needed.
6. Results are returned as JSON (`/search`) or rendered in minimal UI (`/`).

## Repository Layout

### Directory map

- `cmd/`: runnable binaries (entrypoints)
- `pkg/`: reusable core library code (indexing, parsing, ranking)
- `ui/`: browser UI assets served by server
- `benchmarks/`: judged query datasets for offline relevance evaluation
- `data/`: generated runtime artifacts (index segments + manifest)

### File-level responsibilities

#### `cmd/`

- `cmd/indexer/main.go`
  - Runs the full indexing pipeline
  - Reads XML dump input (file or stdin stream)
  - Uses worker pool to process docs (`title`, `intro`, `body` tokenization)
  - Writes segmented index files and `manifest.json`

- `cmd/server/main.go`
  - Loads persisted segments from disk on startup
  - Exposes `/health` and `/search` HTTP endpoints
  - Serves UI files (`ui/index.html`, `/static/*`)
  - Converts ranked titles into Wikipedia links for client responses

- `cmd/benchmark/main.go`
  - Loads saved index segments
  - Loads judged query file
  - Executes search and computes relevance metrics:
    - `MRR@10`
    - `nDCG@10`
    - `Recall@50`

#### `pkg/`

- `pkg/parser/parser.go`
  - Streaming XML parser for Wikipedia dump pages
  - Handles file path input, stdin input, `.bz2` and plain XML streams
  - Emits article docs while skipping non-article and redirect pages

- `pkg/parser/wikitext.go`
  - Cleans MediaWiki/HTML markup into plain text (`StripWikitext`)
  - Extracts compact intro snippet from cleaned text (`ExtractIntro`)

- `pkg/tokenizer/tokenizer.go`
  - Tokenization over alphanumeric boundaries
  - Lowercasing, stopword filtering, digit-only filtering
  - Emits token + position pairs

- `pkg/tokenizer/stemmer.go`
  - Lightweight stemming rules for normalization

- `pkg/index/index.go`
  - Core in-memory index data structures
  - Fielded postings:
    - `PostingLists` (body)
    - `TitlePostingLists` (title)
    - `IntroPostingLists` (intro)
  - Document metadata and corpus statistics
  - Segment merge behavior

- `pkg/index/search.go`
  - Query parsing (`ParseQuery`) with quoted phrase extraction
  - Weighted fielded BM25 scoring over title/intro/body
  - Phrase boost scoring (title > intro > body)
  - Strict all-term retrieval with partial-match fallback

- `pkg/index/persist.go`
  - Gob serialization/deserialization for index segments
  - Backward-safe map initialization during load

- `pkg/index/manifest.go`
  - Manifest file parsing
  - Segment loading and merged index materialization

#### `ui/`

- `ui/index.html`
  - Search page structure

- `ui/script.js`
  - Calls `/search` API
  - Renders results list and metadata

- `ui/style.css`
  - UI styling

#### `benchmarks/`

- `benchmarks/queries.sample.json`
  - Example judged query file with graded relevance labels

## Quick Start

## 1) Build index segments

Recommended (faster decompression using all cores):

```bash
pbzip2 -dc enwiki-2026-04-01-p10p1141529.xml.bz2 | go run cmd/indexer/main.go -workers 12 -checkpoint-docs 100000 -out-dir data/index -
```

Alternative (single-process decompression in Go):

```bash
go run cmd/indexer/main.go -workers 12 -checkpoint-docs 100000 -out-dir data/index enwiki-2026-04-01-p10p1141529.xml.bz2
```

Useful flags:
- `-workers`: text processing workers (default: CPU count)
- `-checkpoint-docs`: docs per segment (default: `100000`)
- `-out-dir`: index output directory (default: `data/index`)

## 2) Start search server

```bash
go run cmd/server/main.go -index-dir data/index -port 8080
```

Then open:
- UI: `http://localhost:8080/`
- Health: `http://localhost:8080/health`
- Search API: `http://localhost:8080/search?q=india&k=5`

## 3) Run relevance benchmark

```bash
go run cmd/benchmark/main.go -index-dir data/index -queries benchmarks/queries.sample.json -verbose
```

## API

## `GET /health`

Returns basic server/index status:

```json
{
  "ok": true,
  "source": "-",
  "segments": 4,
  "total_docs": 355166,
  "total_terms": 2848252
}
```

## `GET /search?q=<query>&k=<int>`

Parameters:
- `q` (required): query string
- `k` (optional): top-k results, clamped to `1..100` (default from `-topk-default`)

Example response:

```json
{
  "query": "india",
  "results_count": 2,
  "duration_ms": 11,
  "results": [
    {
      "title": "List of Indian Mutiny Victoria Cross recipients",
      "url": "https://en.wikipedia.org/wiki/List_of_Indian_Mutiny_Victoria_Cross_recipients"
    }
  ]
}
```

## Index Output Format

`data/index/manifest.json`:
- source input (`source`)
- worker/checkpoint config
- segment list and per-segment stats
- total docs/tokens

`data/index/segment_*.gob`:
- serialized `pkg/index.Index`
- fielded postings maps (`title`, `intro`, `body`), doc store, corpus stats

## Notes and Limitations (Current)

- Server loads all segments into RAM at startup for faster query latency.
- Startup time depends on index size (for ~2.2GB segments, roughly tens of seconds).
- No distributed indexing/search; single-node local process.
- No incremental merge/compaction pipeline yet.
- No authentication/rate limiting; intended for local usage.

## Development

Run tests:

```bash
go test ./...
```
