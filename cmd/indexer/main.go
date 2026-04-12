package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"coja/pkg/index"
	"coja/pkg/parser"
	"coja/pkg/tokenizer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run cmd/indexer/main.go <path-to-dump.xml.bz2>")
		os.Exit(1)
	}``

	docs := make(chan parser.Document, 100)

	go func() {
		if err := parser.Parse(os.Args[1], docs); err != nil {
			fmt.Println("parse error:", err)
			os.Exit(1)
		}
	}()

	idx := index.NewIndex()
	start := time.Now()
	maxDocs := 50000

	for doc := range docs {
		clean := parser.StripWikitext(doc.Text)
		tokens := tokenizer.Tokenize(clean)

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

		if idx.TotalDocs >= maxDocs {
			break
		}
	}

	fmt.Printf("\nDone in %s\n", time.Since(start))
	fmt.Printf("Documents: %d\n", idx.TotalDocs)
	fmt.Printf("Unique terms: %d\n", len(idx.PostingLists))
	fmt.Printf("Avg doc length: %.1f tokens\n", idx.AvgDocLength())

	// Interactive search REPL
	fmt.Println("\n--- Search (type 'quit' to exit) ---")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nquery> ")
		if !scanner.Scan() {
			break
		}

		query := strings.TrimSpace(scanner.Text())
		if query == "quit" || query == "exit" {
			break
		}
		if query == "" {
			continue
		}

		queryTokens := tokenizer.Tokenize(query)
		terms := make([]string, len(queryTokens))``
		for i, t := range queryTokens {
			terms[i] = t.Term
		}

		fmt.Printf("searching for: %v\n", terms)

		searchStart := time.Now()
		results := idx.Search(terms, 10)
		elapsed := time.Since(searchStart)

		if len(results) == 0 {
			fmt.Println("no results found")
			continue
		}

		fmt.Printf("found %d results in %s\n\n", len(results), elapsed)
		for i, r := range results {
			fmt.Printf("  %d. [%.4f] %s\n", i+1, r.Score, r.Title)
		}
	}
}
