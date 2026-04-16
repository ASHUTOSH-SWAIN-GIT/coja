package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coja/pkg/index"
	"coja/pkg/tokenizer"
)

type searchResponse struct {
	Query        string         `json:"query"`
	Terms        []string       `json:"terms"`
	ResultsCount int            `json:"results_count"`
	DurationMS   int64          `json:"duration_ms"`
	Results      []index.Result `json:"results"`
}

type healthResponse struct {
	OK         bool   `json:"ok"`
	Source     string `json:"source"`
	Segments   int    `json:"segments"`
	TotalDocs  int    `json:"total_docs"`
	TotalTerms int    `json:"total_terms"`
}

func main() {
	indexDir := flag.String("index-dir", "data/index", "directory containing manifest.json and segment files")
	port := flag.Int("port", 8080, "HTTP port")
	topKDefault := flag.Int("topk-default", 10, "default top-K results when k is omitted")
	flag.Parse()

	start := time.Now()
	idx, manifest, err := index.LoadFromManifest(*indexDir)
	if err != nil {
		log.Fatalf("failed to load index from %s: %v", *indexDir, err)
	}
	log.Printf("loaded index from %s in %s", *indexDir, time.Since(start))
	log.Printf("segments=%d docs=%d terms=%d tokens=%d", len(manifest.Segments), idx.TotalDocs, len(idx.PostingLists), idx.TotalTokens)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			OK:         true,
			Source:     manifest.Source,
			Segments:   len(manifest.Segments),
			TotalDocs:  idx.TotalDocs,
			TotalTerms: len(idx.PostingLists),
		})
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		if rawQuery == "" {
			http.Error(w, "missing query parameter: q", http.StatusBadRequest)
			return
		}

		k := *topKDefault
		if rawK := strings.TrimSpace(r.URL.Query().Get("k")); rawK != "" {
			parsedK, err := strconv.Atoi(rawK)
			if err != nil || parsedK < 1 {
				http.Error(w, "invalid k, expected positive integer", http.StatusBadRequest)
				return
			}
			if parsedK > 100 {
				parsedK = 100
			}
			k = parsedK
		}

		queryTokens := tokenizer.Tokenize(rawQuery)
		if len(queryTokens) == 0 {
			http.Error(w, "query has no searchable terms after tokenization", http.StatusBadRequest)
			return
		}

		terms := make([]string, len(queryTokens))
		for i, t := range queryTokens {
			terms[i] = t.Term
		}

		searchStart := time.Now()
		results := idx.Search(terms, k)
		writeJSON(w, http.StatusOK, searchResponse{
			Query:        rawQuery,
			Terms:        terms,
			ResultsCount: len(results),
			DurationMS:   time.Since(searchStart).Milliseconds(),
			Results:      results,
		})
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("serving on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Coja Search</title>
  <style>
    :root {
      --bg: #f4f1ea;
      --paper: #fffdf9;
      --text: #1f2937;
      --muted: #6b7280;
      --line: #d1d5db;
      --accent: #0f766e;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: radial-gradient(circle at top, #f9f7f2, var(--bg) 60%);
      color: var(--text);
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    }
    .wrap {
      max-width: 820px;
      margin: 40px auto;
      padding: 0 16px 32px;
    }
    .card {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 16px;
    }
    h1 {
      margin: 0 0 12px;
      font-size: 1.35rem;
    }
    form {
      display: grid;
      grid-template-columns: 1fr 110px 120px;
      gap: 8px;
    }
    input, button {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 10px 12px;
      font-size: 14px;
    }
    button {
      background: var(--accent);
      color: white;
      border-color: var(--accent);
      cursor: pointer;
    }
    .meta {
      margin-top: 12px;
      color: var(--muted);
      font-size: 13px;
      min-height: 18px;
    }
    .result {
      margin-top: 10px;
      padding: 10px 12px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
    }
    .title { font-weight: 600; }
    .sub { color: var(--muted); font-size: 12px; margin-top: 4px; }
    @media (max-width: 680px) {
      form { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>Coja Search</h1>
      <form id="searchForm">
        <input id="q" name="q" placeholder="Search query..." required />
        <input id="k" name="k" type="number" min="1" max="100" value="10" />
        <button type="submit">Search</button>
      </form>
      <div class="meta" id="meta"></div>
      <div id="results"></div>
    </div>
  </div>

  <script>
    const form = document.getElementById("searchForm");
    const q = document.getElementById("q");
    const k = document.getElementById("k");
    const meta = document.getElementById("meta");
    const results = document.getElementById("results");

    function esc(s) {
      return String(s)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;");
    }

    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const query = q.value.trim();
      const topK = k.value || "10";
      if (!query) return;

      results.innerHTML = "";
      meta.textContent = "Searching...";

      try {
        const resp = await fetch("/search?q=" + encodeURIComponent(query) + "&k=" + encodeURIComponent(topK));
        const text = await resp.text();
        if (!resp.ok) {
          meta.textContent = "Error: " + text;
          return;
        }

        const data = JSON.parse(text);
        const terms = (data.terms || []).join(", ");
        meta.textContent = "Results: " + data.results_count + " | Terms: [" + terms + "] | " + data.duration_ms + "ms";

        if (!data.results || data.results.length === 0) {
          results.innerHTML = "<div class='result'>No results found.</div>";
          return;
        }

        results.innerHTML = data.results.map((r, i) =>
          "<div class='result'>" +
            "<div class='title'>" + (i + 1) + ". " + esc(r.Title) + "</div>" +
            "<div class='sub'>doc=" + r.DocID + " | score=" + Number(r.Score).toFixed(4) + "</div>" +
          "</div>"
        ).join("");
      } catch (err) {
        meta.textContent = "Error: " + err.message;
      }
    });
  </script>
</body>
</html>`
