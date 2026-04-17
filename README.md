# Coja

Coja is a local Wikipedia search engine in Go.

It currently supports:
- Streaming Wikipedia dump parsing
- Wikitext cleanup
- Tokenization + stemming + stopword filtering
- Inverted index + BM25 ranking
- Segment-based on-disk persistence (`.gob`)
- Startup loading from persisted segments
- HTTP search API and a minimal browser UI

## Current Architecture

### Indexing pipeline

1. Input source is either:
- a `.bz2` dump path (Go can read it directly), or
- stdin (`-`), typically fed by `pbzip2 -dc` for parallel decompression

2. Parser streams XML pages sequentially.
3. Worker pool processes pages in parallel:
- strip wikitext
- tokenize text
- stem/filter terms
- emit typed `{docID, title, tokens}` work items

4. Single collector goroutine updates in-memory index.
5. Collector checkpoints every `N` docs into segment files:
- `segment_000001.gob`, `segment_000002.gob`, ...
- `manifest.json` with metadata

### Search pipeline

1. Server loads `manifest.json`.
2. Server loads all segment files into memory and merges them.
3. Query is tokenized using same tokenizer pipeline.
4. BM25 ranks candidate documents.
5. Results are returned as JSON (`/search`) or rendered in minimal UI (`/`).

## Repository Layout

- `cmd/indexer/main.go`: dump indexing pipeline and checkpoint writer
- `cmd/server/main.go`: HTTP server + minimal UI
- `pkg/parser`: XML parsing and wikitext cleanup
- `pkg/tokenizer`: tokenization, stopwords, stemming
- `pkg/index`: inverted index, BM25 search, gob persistence, manifest loader

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
- postings map, doc store, corpus stats

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
