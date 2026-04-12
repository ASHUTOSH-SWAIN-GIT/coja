package index

import (
	"math"
	"sort"
)

// Result represents a single search result
type Result struct {
	DocID int
	Title string
	Score float64
}

// Search takes a list of stemmed query terms and returns top-K results ranked by BM25.
func (idx *Index) Search(queryTerms []string, topK int) []Result {
	if len(queryTerms) == 0 {
		return nil
	}

	// Collect candidate documents — any doc that contains at least one query term
	docScores := make(map[int]float64)

	avgDL := idx.AvgDocLength()
	k1 := 1.2
	b := 0.75

	for _, term := range queryTerms {
		postings, ok := idx.PostingLists[term]
		if !ok {
			continue
		}

		// IDF: how rare is this term across the corpus
		n := float64(len(postings))
		idf := math.Log(1 + (float64(idx.TotalDocs)-n+0.5)/(n+0.5))

		for _, p := range postings {
			dl := float64(idx.DocStore[p.DocID].Length)
			tf := float64(p.Frequency)

			// BM25 term score
			numerator := tf * (k1 + 1)
			denominator := tf + k1*(1-b+b*(dl/avgDL))

			docScores[p.DocID] += idf * (numerator / denominator)
		}
	}

	// Convert map to sorted slice
	results := make([]Result, 0, len(docScores))
	for docID, score := range docScores {
		results = append(results, Result{
			DocID: docID,
			Title: idx.DocStore[docID].Title,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}