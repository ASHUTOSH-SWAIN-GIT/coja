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
	Title  string
	Length int // number of tokens
}

// Index is the inverted index
type Index struct {
	// token → list of postings
	PostingLists map[string][]Posting

	// docID → document metadata
	DocStore map[int]DocInfo

	// corpus stats for BM25
	TotalDocs   int
	TotalTokens int64
}

// NewIndex creates an empty index
func NewIndex() *Index {
	return &Index{
		PostingLists: make(map[string][]Posting),
		DocStore:     make(map[int]DocInfo),
	}
}

// AddDocument indexes a single document's tokens
func (idx *Index) AddDocument(docID int, title string, tokens []TermPosition) {
	// Count term frequencies and collect positions
	termFreq := make(map[string][]int)
	for _, t := range tokens {
		termFreq[t.Term] = append(termFreq[t.Term], t.Position)
	}

	// Add a posting for each unique term
	for term, positions := range termFreq {
		idx.PostingLists[term] = append(idx.PostingLists[term], Posting{
			DocID:     docID,
			Frequency: len(positions),
			Positions: positions,
		})
	}

	// Store document metadata
	idx.DocStore[docID] = DocInfo{
		Title:  title,
		Length: len(tokens),
	}

	idx.TotalDocs++
	idx.TotalTokens += int64(len(tokens))
}

// AvgDocLength returns the average document length across the corpus
func (idx *Index) AvgDocLength() float64 {
	if idx.TotalDocs == 0 {
		return 0
	}
	return float64(idx.TotalTokens) / float64(idx.TotalDocs)
}

// Merge merges another index into the receiver.
func (idx *Index) Merge(other *Index) {
	if other == nil {
		return
	}

	for term, postings := range other.PostingLists {
		idx.PostingLists[term] = append(idx.PostingLists[term], postings...)
	}

	for docID, info := range other.DocStore {
		idx.DocStore[docID] = info
	}

	idx.TotalDocs += other.TotalDocs
	idx.TotalTokens += other.TotalTokens
}
