package index

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"coja/pkg/tokenizer"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75

	bodyWeight  = 1.0
	introWeight = 1.8
	titleWeight = 3.0

	phraseTitleBoost = 4.0
	phraseIntroBoost = 2.2
	phraseBodyBoost  = 1.2
)

var quotedPhraseRegex = regexp.MustCompile(`"([^"]+)"`)

// Result represents a single search result.
type Result struct {
	DocID int
	Title string
	Score float64
}

// ParsedQuery is the normalized query representation for ranking.
type ParsedQuery struct {
	Terms   []string
	Phrases [][]string
}

// ParseQuery tokenizes the input query and extracts quoted phrases.
func ParseQuery(rawQuery string) ParsedQuery {
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery == "" {
		return ParsedQuery{}
	}

	tokenized := tokenizer.Tokenize(rawQuery)
	terms := make([]string, len(tokenized))
	for i, tok := range tokenized {
		terms[i] = tok.Term
	}
	terms = uniqueTerms(terms)

	phraseMatches := quotedPhraseRegex.FindAllStringSubmatch(rawQuery, -1)
	phrases := make([][]string, 0, len(phraseMatches))
	for _, m := range phraseMatches {
		if len(m) < 2 {
			continue
		}
		pTokens := tokenizer.Tokenize(m[1])
		if len(pTokens) < 2 {
			continue
		}
		phrase := make([]string, len(pTokens))
		for i, tok := range pTokens {
			phrase[i] = tok.Term
		}
		phrases = append(phrases, phrase)
	}

	// For multi-word queries without explicit quotes, add the full query as a soft phrase.
	if len(phrases) == 0 && len(terms) >= 2 {
		soft := make([]string, len(terms))
		copy(soft, terms)
		phrases = append(phrases, soft)
	}

	return ParsedQuery{
		Terms:   terms,
		Phrases: phrases,
	}
}

// SearchQuery parses and executes a ranked query.
func (idx *Index) SearchQuery(rawQuery string, topK int) []Result {
	parsed := ParseQuery(rawQuery)
	if len(parsed.Terms) == 0 || topK <= 0 {
		return nil
	}
	return idx.searchParsed(parsed, topK)
}

// Search keeps backward compatibility when callers already have tokenized terms.
func (idx *Index) Search(queryTerms []string, topK int) []Result {
	if len(queryTerms) == 0 || topK <= 0 {
		return nil
	}
	parsed := ParsedQuery{
		Terms:   uniqueTerms(queryTerms),
		Phrases: nil,
	}
	if len(parsed.Terms) == 0 {
		return nil
	}
	return idx.searchParsed(parsed, topK)
}

func (idx *Index) searchParsed(parsed ParsedQuery, topK int) []Result {
	results := idx.searchWithMode(parsed, topK, true)
	if len(results) == 0 && len(parsed.Terms) > 1 {
		return idx.searchWithMode(parsed, topK, false)
	}
	return results
}

func (idx *Index) searchWithMode(parsed ParsedQuery, topK int, requireAllTerms bool) []Result {
	terms := parsed.Terms
	queryTermCount := len(terms)

	docScores := make(map[int]float64)
	docSeenTerms := make(map[int]map[string]struct{})

	bodyPositions := make(map[string]map[int][]int)
	titlePositions := make(map[string]map[int][]int)
	introPositions := make(map[string]map[int][]int)

	avgBodyLength := idx.AvgDocLength()
	avgTitleLength := idx.AvgTitleLength()
	avgIntroLength := idx.AvgIntroLength()
	if avgBodyLength <= 0 {
		avgBodyLength = 1
	}
	if avgTitleLength <= 0 {
		avgTitleLength = 1
	}
	if avgIntroLength <= 0 {
		avgIntroLength = 1
	}

	for _, term := range terms {
		bodyPostings := idx.PostingLists[term]
		titlePostings := idx.TitlePostingLists[term]
		introPostings := idx.IntroPostingLists[term]

		if len(bodyPostings) == 0 && len(titlePostings) == 0 && len(introPostings) == 0 {
			continue
		}

		if len(bodyPostings) > 0 {
			if bodyPositions[term] == nil {
				bodyPositions[term] = make(map[int][]int)
			}
			idfBody := bm25IDF(idx.TotalDocs, len(bodyPostings))
			for _, p := range bodyPostings {
				markTermSeen(docSeenTerms, p.DocID, term, queryTermCount)
				bodyPositions[term][p.DocID] = p.Positions

				dl := idx.docBodyLength(p.DocID)
				docScores[p.DocID] += bodyWeight * bm25TermScore(idfBody, p.Frequency, dl, avgBodyLength)
			}
		}

		if len(titlePostings) > 0 {
			if titlePositions[term] == nil {
				titlePositions[term] = make(map[int][]int)
			}
			idfTitle := bm25IDF(idx.TotalDocs, len(titlePostings))
			for _, p := range titlePostings {
				markTermSeen(docSeenTerms, p.DocID, term, queryTermCount)
				titlePositions[term][p.DocID] = p.Positions

				dl := idx.docTitleLength(p.DocID)
				docScores[p.DocID] += titleWeight * bm25TermScore(idfTitle, p.Frequency, dl, avgTitleLength)
			}
		}

		if len(introPostings) > 0 {
			if introPositions[term] == nil {
				introPositions[term] = make(map[int][]int)
			}
			idfIntro := bm25IDF(idx.TotalDocs, len(introPostings))
			for _, p := range introPostings {
				markTermSeen(docSeenTerms, p.DocID, term, queryTermCount)
				introPositions[term][p.DocID] = p.Positions

				dl := idx.docIntroLength(p.DocID)
				docScores[p.DocID] += introWeight * bm25TermScore(idfIntro, p.Frequency, dl, avgIntroLength)
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

		if matchedTerms == queryTermCount && len(parsed.Phrases) > 0 {
			score += phraseBoostForDoc(docID, parsed.Phrases, titlePositions, introPositions, bodyPositions)
		}

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

func phraseBoostForDoc(docID int, phrases [][]string, titlePositions map[string]map[int][]int, introPositions map[string]map[int][]int, bodyPositions map[string]map[int][]int) float64 {
	boost := 0.0
	for _, phrase := range phrases {
		if len(phrase) < 2 {
			continue
		}
		if containsPhraseInDocField(docID, phrase, titlePositions) {
			boost += phraseTitleBoost
			continue
		}
		if containsPhraseInDocField(docID, phrase, introPositions) {
			boost += phraseIntroBoost
			continue
		}
		if containsPhraseInDocField(docID, phrase, bodyPositions) {
			boost += phraseBodyBoost
		}
	}
	return boost
}

func containsPhraseInDocField(docID int, phrase []string, positionsByTerm map[string]map[int][]int) bool {
	if len(phrase) == 0 {
		return false
	}

	first := positionsByTerm[phrase[0]][docID]
	if len(first) == 0 {
		return false
	}

	for _, start := range first {
		match := true
		for i := 1; i < len(phrase); i++ {
			positions := positionsByTerm[phrase[i]][docID]
			if !containsInt(positions, start+i) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsInt(values []int, want int) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func markTermSeen(docSeenTerms map[int]map[string]struct{}, docID int, term string, queryTermCount int) {
	if docSeenTerms[docID] == nil {
		docSeenTerms[docID] = make(map[string]struct{}, queryTermCount)
	}
	docSeenTerms[docID][term] = struct{}{}
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

func (idx *Index) docIntroLength(docID int) float64 {
	info := idx.DocStore[docID]
	if info.IntroLength > 0 {
		return float64(info.IntroLength)
	}
	return 1
}
