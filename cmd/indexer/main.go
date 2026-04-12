package main

import (
	"coja/pkg/parser"
	"coja/pkg/tokenizer"
	"fmt"
	"os"
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

	count := 0
	for doc := range docs {
		clean := parser.StripWikitext(doc.Text)
		tokens := tokenizer.Tokenize(clean)

		fmt.Printf("[%d] %s — %d tokens\n", doc.ID, doc.Title, len(tokens))

		// Show first 20 tokens
		limit := 20
		if len(tokens) < limit {
			limit = len(tokens)
		}
		for _, t := range tokens[:limit] {
			fmt.Printf("  %d: %s\n", t.Position, t.Term)
		}
		fmt.Println()

		count++
		if count >= 3 {
			break
		}
	}
}
