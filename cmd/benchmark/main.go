package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"coja/pkg/index"
)

type JudgedDoc struct {
	Title string `json:"title"`
	Grade int    `json:"grade"`
}

type BenchmarkQuery struct {
	Query     string      `json:"query"`
	Relevance []JudgedDoc `json:"relevance"`
}

type Metrics struct {
	QueryCount int
	MRR10      float64
	NDCG10     float64
	Recall50   float64
}

func main() {
	indexDir := flag.String("index-dir", "data/index", "directory containing manifest.json and segment files")
	queriesPath := flag.String("queries", "", "path to benchmark query JSON file")
	verbose := flag.Bool("verbose", false, "print per-query metrics")
	flag.Parse()

	if strings.TrimSpace(*queriesPath) == "" {
		fmt.Println("usage: go run cmd/benchmark/main.go -index-dir data/index -queries benchmarks/queries.json")
		os.Exit(1)
	}

	idx, manifest, err := index.LoadFromManifest(*indexDir)
	if err != nil {
		fmt.Printf("failed to load index: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("loaded index: segments=%d docs=%d terms=%d\n", len(manifest.Segments), idx.TotalDocs, len(idx.PostingLists))

	queries, err := loadQueries(*queriesPath)
	if err != nil {
		fmt.Printf("failed to load queries: %v\n", err)
		os.Exit(1)
	}
	if len(queries) == 0 {
		fmt.Println("no benchmark queries found")
		os.Exit(1)
	}

	metrics := runBenchmark(idx, queries, *verbose)
	fmt.Printf("\nqueries: %d\n", metrics.QueryCount)
	fmt.Printf("MRR@10: %.4f\n", metrics.MRR10)
	fmt.Printf("nDCG@10: %.4f\n", metrics.NDCG10)
	fmt.Printf("Recall@50: %.4f\n", metrics.Recall50)
}

func loadQueries(path string) ([]BenchmarkQuery, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var queries []BenchmarkQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		return nil, err
	}
	return queries, nil
}

func runBenchmark(idx *index.Index, queries []BenchmarkQuery, verbose bool) Metrics {
	var sumMRR, sumNDCG, sumRecall float64
	count := 0

	for _, q := range queries {
		if strings.TrimSpace(q.Query) == "" || len(q.Relevance) == 0 {
			continue
		}

		relevanceByTitle := make(map[string]int, len(q.Relevance))
		for _, rel := range q.Relevance {
			title := normalizeTitle(rel.Title)
			if title == "" {
				continue
			}
			if rel.Grade > relevanceByTitle[title] {
				relevanceByTitle[title] = rel.Grade
			}
		}
		if len(relevanceByTitle) == 0 {
			continue
		}

		results := idx.SearchQuery(q.Query, 50)
		mrr10 := computeMRRAtK(results, relevanceByTitle, 10)
		ndcg10 := computeNDCGAtK(results, relevanceByTitle, 10)
		recall50 := computeRecallAtK(results, relevanceByTitle, 50)

		sumMRR += mrr10
		sumNDCG += ndcg10
		sumRecall += recall50
		count++

		if verbose {
			fmt.Printf("q=%q mrr@10=%.4f ndcg@10=%.4f recall@50=%.4f\n", q.Query, mrr10, ndcg10, recall50)
		}
	}

	if count == 0 {
		return Metrics{}
	}

	return Metrics{
		QueryCount: count,
		MRR10:      sumMRR / float64(count),
		NDCG10:     sumNDCG / float64(count),
		Recall50:   sumRecall / float64(count),
	}
}

func computeMRRAtK(results []index.Result, relevanceByTitle map[string]int, k int) float64 {
	limit := minInt(k, len(results))
	for i := 0; i < limit; i++ {
		if relevanceByTitle[normalizeTitle(results[i].Title)] > 0 {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

func computeNDCGAtK(results []index.Result, relevanceByTitle map[string]int, k int) float64 {
	limit := minInt(k, len(results))
	if limit == 0 {
		return 0
	}

	dcg := 0.0
	for i := 0; i < limit; i++ {
		grade := relevanceByTitle[normalizeTitle(results[i].Title)]
		if grade <= 0 {
			continue
		}
		dcg += (math.Pow(2, float64(grade)) - 1) / math.Log2(float64(i+2))
	}

	grades := make([]int, 0, len(relevanceByTitle))
	for _, g := range relevanceByTitle {
		if g > 0 {
			grades = append(grades, g)
		}
	}
	if len(grades) == 0 {
		return 0
	}
	sort.Slice(grades, func(i, j int) bool { return grades[i] > grades[j] })

	idealLimit := minInt(k, len(grades))
	idcg := 0.0
	for i := 0; i < idealLimit; i++ {
		idcg += (math.Pow(2, float64(grades[i])) - 1) / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func computeRecallAtK(results []index.Result, relevanceByTitle map[string]int, k int) float64 {
	totalRelevant := 0
	for _, grade := range relevanceByTitle {
		if grade > 0 {
			totalRelevant++
		}
	}
	if totalRelevant == 0 {
		return 0
	}

	limit := minInt(k, len(results))
	found := 0
	seen := make(map[string]struct{}, limit)
	for i := 0; i < limit; i++ {
		title := normalizeTitle(results[i].Title)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		if relevanceByTitle[title] > 0 {
			found++
		}
	}

	return float64(found) / float64(totalRelevant)
}

func normalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
