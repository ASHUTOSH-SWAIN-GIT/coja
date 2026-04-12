package main

import (
	"fmt"
	"os"
	"time"

	"coja/pkg/index"
	"coja/pkg/parser"
	"coja/pkg/tokenizer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run cmd/indexer/main.go <path-to-dump.xml.bz2>")
		os.Exit(1)
	}

	docs := make(chan parser.Document, 100)

	go func() {
		if err := parser.Parse(os.Args[1], docs); err != nil {
			fmt.Println("parse error:", err)
			os.Exit(1)
		}
	}()

	idx := index.NewIndex()
	start := time.Now()

	for doc := range docs {
		clean := parser.StripWikitext(doc.Text)
		tokens := tokenizer.Tokenize(clean)

		// Convert tokens for the index
		indexTokens := make([]struct {
			Term     string
			Position int
		}, len(tokens))
		for i, t := range tokens {
			indexTokens[i].Term = t.Term
			indexTokens[i].Position = t.Position
		}

		idx.AddDocument(doc.ID, doc.Title, indexTokens)

		if idx.TotalDocs%10000 == 0 {
			fmt.Printf("Indexed %d docs (%s)\n", idx.TotalDocs, time.Since(start))
		}
	}

	fmt.Printf("\nDone in %s\n", time.Since(start))
	fmt.Printf("Documents: %d\n", idx.TotalDocs)
	fmt.Printf("Unique terms: %d\n", len(idx.PostingLists))
	fmt.Printf("Avg doc length: %.1f tokens\n", idx.AvgDocLength())

	// Test a lookup
	term := "anarchism"
	if postings, ok := idx.PostingLists[term]; ok {
		fmt.Printf("\nTerm '%s' appears in %d documents\n", term, len(postings))
		limit := 5
		if len(postings) < limit {
			limit = len(postings)
		}
		for _, p := range postings[:limit] {
			fmt.Printf("  doc %d (%s) — %d times\n", p.DocID, idx.DocStore[p.DocID].Title, p.Frequency)
		}
	}
}
