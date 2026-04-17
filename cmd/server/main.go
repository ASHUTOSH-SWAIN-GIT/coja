package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"coja/pkg/index"
)

type searchResponse struct {
	Query        string         `json:"query"`
	ResultsCount int            `json:"results_count"`
	DurationMS   int64          `json:"duration_ms"`
	Results      []searchResult `json:"results"`
}

type searchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
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
	uiDir := flag.String("ui-dir", "ui", "directory containing static UI files")
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

	absUI, err := filepath.Abs(*uiDir)
	if err != nil {
		log.Fatalf("invalid ui-dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(absUI, "index.html")); err != nil {
		log.Fatalf("ui-dir %s missing index.html: %v", absUI, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(absUI, "index.html"))
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(absUI))))

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

		parsed := index.ParseQuery(rawQuery)
		if len(parsed.Terms) == 0 {
			http.Error(w, "query has no searchable terms after tokenization", http.StatusBadRequest)
			return
		}

		searchStart := time.Now()
		rawResults := idx.SearchQuery(rawQuery, k)
		results := make([]searchResult, len(rawResults))
		for i, r := range rawResults {
			results[i] = searchResult{
				Title: r.Title,
				URL:   wikipediaURL(r.Title),
			}
		}

		writeJSON(w, http.StatusOK, searchResponse{
			Query:        rawQuery,
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

func wikipediaURL(title string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
	return "https://en.wikipedia.org/wiki/" + url.PathEscape(normalized)
}
