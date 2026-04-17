package index

import (
	"math"
	"sort"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75

	bodyWeight  = 1.0
	titleWeight = 2.8
)

// Result represents a single search result.
type Result struct {
	DocID int
	Title string
	Score float64
}

// Search takes a list of stemmed query terms and returns top-K results ranked by fielded BM25.
//
// Ranking = (bodyWeight * bodyBM25) + (titleWeight * titleBM25)
// where body/title BM25 are computed from separate posting lists and length norms.
func (idx *Index) Search(queryTerms []string, topK int) []Result {
	if len(queryTerms) == 0 || topK <= 0 {
		return nil
	}

	terms := uniqueTerms(queryTerms)
	if len(terms) == 0 {
		return nil
	}

	// Stage 1: require all terms for stronger precision on multi-word queries.
	results := idx.searchWithMode(terms, topK, true)
	// Stage 2 fallback: if strict mode has no matches, allow partial matches.
	if len(results) == 0 && len(terms) > 1 {
		return idx.searchWithMode(terms, topK, false)
	}
	return results
}

func (idx *Index) searchWithMode(terms []string, topK int, requireAllTerms bool) []Result {
	docScores := make(map[int]float64)
	docSeenTerms := make(map[int]map[string]struct{})

	avgBodyLength := idx.AvgDocLength()
	avgTitleLength := idx.AvgTitleLength()
	if avgBodyLength <= 0 {
		avgBodyLength = 1
	}
	if avgTitleLength <= 0 {
		avgTitleLength = 1
	}

	queryTermCount := len(terms)

	for _, term := range terms {
		bodyPostings := idx.PostingLists[term]
		titlePostings := idx.TitlePostingLists[term]

		if len(bodyPostings) == 0 && len(titlePostings) == 0 {
			continue
		}

		if len(bodyPostings) > 0 {
			idfBody := bm25IDF(idx.TotalDocs, len(bodyPostings))
			for _, p := range bodyPostings {
				if docSeenTerms[p.DocID] == nil {
					docSeenTerms[p.DocID] = make(map[string]struct{}, queryTermCount)
				}
				docSeenTerms[p.DocID][term] = struct{}{}

				dl := idx.docBodyLength(p.DocID)
				docScores[p.DocID] += bodyWeight * bm25TermScore(idfBody, p.Frequency, dl, avgBodyLength)
			}
		}

		if len(titlePostings) > 0 {
			idfTitle := bm25IDF(idx.TotalDocs, len(titlePostings))
			for _, p := range titlePostings {
				if docSeenTerms[p.DocID] == nil {
					docSeenTerms[p.DocID] = make(map[string]struct{}, queryTermCount)
				}
				docSeenTerms[p.DocID][term] = struct{}{}

				dl := idx.docTitleLength(p.DocID)
				docScores[p.DocID] += titleWeight * bm25TermScore(idfTitle, p.Frequency, dl, avgTitleLength)
			}
		}
	}

	results := make([]Result, 0, len(docScores))
	for docID, score := range docScores {
		matchedTerms := len(docSeenTerms[docID])
		if requireAllTerms && matchedTerms < queryTermCount {
			continue
		}

		coverage := float64(matchedTerms) / float64(queryTermCount)
		score *= 0.6 + 0.4*coverage

		results = append(results, Result{
			DocID: docID,
			Title: idx.DocStore[docID].Title,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DocID < results[j].DocID
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

func uniqueTerms(terms []string) []string {
	out := make([]string, 0, len(terms))
	seen := make(map[string]struct{}, len(terms))
	for _, t := range terms {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func bm25IDF(totalDocs int, docFreq int) float64 {
	n := float64(docFreq)
	return math.Log(1 + (float64(totalDocs)-n+0.5)/(n+0.5))
}

func bm25TermScore(idf float64, frequency int, dl float64, avgDL float64) float64 {
	tf := float64(frequency)
	numerator := tf * (bm25K1 + 1)
	denominator := tf + bm25K1*(1-bm25B+bm25B*(dl/avgDL))
	return idf * (numerator / denominator)
}

func (idx *Index) docBodyLength(docID int) float64 {
	info := idx.DocStore[docID]
	if info.BodyLength > 0 {
		return float64(info.BodyLength)
	}
	if info.Length > 0 {
		return float64(info.Length)
	}
	return 1
}

func (idx *Index) docTitleLength(docID int) float64 {
	info := idx.DocStore[docID]
	if info.TitleLength > 0 {
		return float64(info.TitleLength)
	}
	return 1
}
