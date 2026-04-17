# cója

`cója` is a local search engine over Wikipedia dump data, implemented in Go.

This document explains the system design, core concepts, indexing model, ranking logic, and internal data flow.

---

## Project Purpose

The project is built to implement a practical information-retrieval system from first principles, with:
- streamed corpus ingestion,
- fielded inverted indexing,
- lexical ranking with BM25,
- phrase/proximity-aware relevance adjustments,
- and reproducible offline evaluation.

The design favors clarity of IR concepts and debuggability over distributed scale.

---

## Core Concepts Used

### 1) Inverted index

The index maps normalized terms to posting lists.

A posting represents one document-level occurrence summary:
- `DocID`
- `Frequency` (term frequency in that field/document)
- `Positions` (token positions, enabling phrase/proximity logic)

### 2) Fielded retrieval

The same document is indexed across multiple fields:
- `title`
- `intro`
- `body`

Each field has separate posting lists and separate length statistics.  
This enables field-aware scoring (for example, a match in title can be weighted more than body).

### 3) BM25 ranking

BM25 is used as the base lexical scoring function.  
Scoring is computed independently per field and combined as a weighted sum:

`score = w_title * BM25(title) + w_intro * BM25(intro) + w_body * BM25(body) + phraseBoost`

Current relative weighting emphasizes title and intro over body.

### 4) Phrase and positional relevance

Because postings store token positions, the system can reward exact in-order adjacency for query phrases.

Phrase boosts are field-sensitive:
- strongest in `title`,
- then `intro`,
- then `body`.

### 5) Two-stage retrieval behavior

For multi-term queries:
1. strict stage requires all query terms,
2. fallback stage allows partial matches if strict stage yields no results.

This balances precision and recall for entity-style queries.

### 6) Offline relevance evaluation

A benchmark runner evaluates ranked output using judged query sets and computes:
- `MRR@10`
- `nDCG@10`
- `Recall@50`

This provides objective tuning feedback for ranking changes.

---

## System Architecture

The system has three major runtime paths:

1. **Indexing path**: converts Wikipedia dump pages into persisted index segments.
2. **Serving path**: loads persisted segments into memory and answers queries.
3. **Evaluation path**: runs judged queries and computes quality metrics.

---

## Indexing Path (Conceptual Flow)

1. **Input stream**
   - Wikipedia XML is consumed either from a file stream or stdin stream.
   - Parsing is sequential because XML token order is linear.

2. **Document extraction**
   - Only article namespace pages are kept.
   - Redirect pages are skipped.

3. **Text normalization**
   - MediaWiki markup and HTML-like artifacts are removed.
   - Entities and whitespace are normalized.

4. **Field preparation**
   - `title` comes from page metadata.
   - `body` comes from cleaned article text.
   - `intro` is a compact first-sentence-like snippet derived from body.

5. **Tokenization and stemming**
   - lowercasing
   - boundary-based token splitting
   - stopword and numeric filtering
   - lightweight stemming
   - positional emission

6. **Index construction**
   - for each field, per-term positions/frequencies are aggregated per document.
   - posting lists are updated.
   - document length and corpus stats are updated.

7. **Segment persistence**
   - documents are indexed in bounded chunks.
   - each chunk is serialized as a segment file.
   - a manifest records segment inventory and corpus-level metadata.

---

## Serving Path (Conceptual Flow)

1. Manifest is read.
2. Referenced segments are loaded and merged into one in-memory index.
3. Query text is parsed into:
   - normalized terms,
   - optional quoted phrases,
   - optional soft phrase for unquoted multi-term queries.
4. Candidate scoring is computed with weighted fielded BM25.
5. Phrase/proximity boosts are applied using positional postings.
6. Strict-or-fallback retrieval mode selects result set.
7. Ranked results are returned with canonical Wikipedia URLs.

---

## Query Interpretation Model

### Terms

All query text goes through the same normalization pipeline used during indexing to ensure term-space consistency.

### Quoted phrases

Quoted substrings are parsed as explicit phrase intents and receive targeted positional boosts.

### Soft phrase behavior

When a query has multiple unquoted terms, the full term sequence is treated as a soft phrase candidate, improving name/entity ranking without requiring explicit quotes.

---

## Ranking Details

### Base scoring

For each query term:
- compute field-specific BM25 using field-specific document length normalization.
- accumulate weighted contribution across fields.

### Coverage scaling

Documents that match a higher fraction of query terms are scaled upward.

### Phrase boosts

For each phrase candidate:
- detect adjacent in-order term positions by field.
- apply additive boosts with highest priority for title matches.

### Final ordering

Results are sorted by descending score, with deterministic tie-break behavior.

---

## Data Model

### Index structures

- **Posting lists**
  - `PostingLists` (body)
  - `TitlePostingLists` (title)
  - `IntroPostingLists` (intro)

- **Document store**
  - document title
  - per-field lengths

- **Corpus stats**
  - total docs
  - per-field total token counts
  - derived average field lengths

### Persistence structures

- **Segment files**
  - serialized index shards
  - intended for bounded-memory indexing and resumable artifacts

- **Manifest**
  - segment inventory
  - indexing parameters
  - corpus aggregate stats

---

## Codebase Map (Responsibilities)

### `cmd/`

- `cmd/indexer/main.go`: indexing orchestration, worker/collector pipeline, segment writing.
- `cmd/server/main.go`: search serving, query handling, result projection.
- `cmd/benchmark/main.go`: judged-query evaluation and metrics reporting.

### `pkg/parser/`

- `parser.go`: streaming XML page extraction.
- `wikitext.go`: wiki markup cleaning and intro extraction.

### `pkg/tokenizer/`

- `tokenizer.go`: normalization/tokenization pipeline with positions.
- `stemmer.go`: stemming logic.

### `pkg/index/`

- `index.go`: core index schema, per-field stats, merge.
- `search.go`: query parsing + scoring + ranking.
- `persist.go`: segment serialization/deserialization.
- `manifest.go`: manifest parsing and segment loading.

### `ui/`

UI layer consuming `/search` responses for result presentation.

### `benchmarks/`

Judged query files used by the evaluation runner.

---

## Design Tradeoffs

### Chosen tradeoffs

- **Startup-heavy, query-fast serving**
  - all segments are loaded into RAM before serving.
- **Lexical and explainable ranking**
  - BM25 + explicit boosts rather than opaque ML ranking.
- **Simple local persistence**
  - gob-based segments for straightforward serialization.

### Consequences

- restart requires reload/merge of persisted segments,
- memory usage grows with indexed corpus size,
- quality is bounded by lexical features and judged-set coverage.

---

## Current Limitations

- Single-node in-memory serving model.
- No typo correction or synonym/alias expansion yet.
- No learned reranking stage.
- No distributed indexing/search execution.
- No production hardening layer (auth, quotas, multi-tenant controls).

---

## Summary

`cója` currently represents a complete lexical IR stack:
- streamed ingestion,
- fielded positional indexing,
- weighted BM25 + phrase-aware ranking,
- persistent segments,
- online serving,
- and offline quality measurement.

Its architecture is intentionally modular so ranking and storage strategies can evolve independently.

