package index

// Posting represents one occurrence of a term in a document
type Posting struct {
	DocID     int
	Frequency int
	Positions []int
}

// TermPosition represents a normalized term and where it occurred in the document.
type TermPosition struct {
	Term     string
	Position int
}

// DocInfo stores metadata about a document
type DocInfo struct {
	Title       string
	Length      int // body token count, kept for compatibility
	BodyLength  int
	TitleLength int
}

// Index is the inverted index
type Index struct {
	// body token -> list of postings
	PostingLists map[string][]Posting
	// title token -> list of postings
	TitlePostingLists map[string][]Posting

	// docID → document metadata
	DocStore map[int]DocInfo

	// corpus stats for BM25
	TotalDocs        int
	TotalTokens      int64 // body tokens, kept for compatibility
	TotalBodyTokens  int64
	TotalTitleTokens int64
}

// NewIndex creates an empty index
func NewIndex() *Index {
	return &Index{
		PostingLists:      make(map[string][]Posting),
		TitlePostingLists: make(map[string][]Posting),
		DocStore:          make(map[int]DocInfo),
	}
}

// AddDocument indexes a single document's body and title tokens.
func (idx *Index) AddDocument(docID int, title string, bodyTokens []TermPosition, titleTokens []TermPosition) {
	// Count body term frequencies and collect positions
	bodyTermFreq := make(map[string][]int)
	for _, t := range bodyTokens {
		bodyTermFreq[t.Term] = append(bodyTermFreq[t.Term], t.Position)
	}

	for term, positions := range bodyTermFreq {
		idx.PostingLists[term] = append(idx.PostingLists[term], Posting{
			DocID:     docID,
			Frequency: len(positions),
			Positions: positions,
		})
	}

	// Count title term frequencies and collect positions
	titleTermFreq := make(map[string][]int)
	for _, t := range titleTokens {
		titleTermFreq[t.Term] = append(titleTermFreq[t.Term], t.Position)
	}

	for term, positions := range titleTermFreq {
		idx.TitlePostingLists[term] = append(idx.TitlePostingLists[term], Posting{
			DocID:     docID,
			Frequency: len(positions),
			Positions: positions,
		})
	}

	// Store document metadata
	idx.DocStore[docID] = DocInfo{
		Title:       title,
		Length:      len(bodyTokens),
		BodyLength:  len(bodyTokens),
		TitleLength: len(titleTokens),
	}

	idx.TotalDocs++
	idx.TotalTokens += int64(len(bodyTokens))
	idx.TotalBodyTokens += int64(len(bodyTokens))
	idx.TotalTitleTokens += int64(len(titleTokens))
}

// AvgDocLength returns the average body length across the corpus.
func (idx *Index) AvgDocLength() float64 {
	if idx.TotalDocs == 0 {
		return 0
	}
	if idx.TotalBodyTokens > 0 {
		return float64(idx.TotalBodyTokens) / float64(idx.TotalDocs)
	}
	return float64(idx.TotalTokens) / float64(idx.TotalDocs)
}

// AvgTitleLength returns the average title length across the corpus.
func (idx *Index) AvgTitleLength() float64 {
	if idx.TotalDocs == 0 {
		return 0
	}
	if idx.TotalTitleTokens == 0 {
		return 1
	}
	return float64(idx.TotalTitleTokens) / float64(idx.TotalDocs)
}

// Merge merges another index into the receiver.
func (idx *Index) Merge(other *Index) {
	if other == nil {
		return
	}

	for term, postings := range other.PostingLists {
		idx.PostingLists[term] = append(idx.PostingLists[term], postings...)
	}
	for term, postings := range other.TitlePostingLists {
		idx.TitlePostingLists[term] = append(idx.TitlePostingLists[term], postings...)
	}

	for docID, info := range other.DocStore {
		idx.DocStore[docID] = info
	}

	idx.TotalDocs += other.TotalDocs
	idx.TotalTokens += other.TotalTokens
	idx.TotalBodyTokens += other.TotalBodyTokens
	idx.TotalTitleTokens += other.TotalTitleTokens
}
