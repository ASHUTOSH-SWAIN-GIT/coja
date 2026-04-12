package main

import (
	"coja/pkg/parser"
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
		if err := parser.Parse(os.Args[1],docs);err != nil {
			fmt.Println("parse error:" , err)
			os.Exit(1)
		}
	}()

	
	count := 0
	for doc := range docs {
		fmt.Printf("[%d] %s (text: %d bytes)\n", doc.ID, doc.Title, len(doc.Text))
		count++
		if count >= 10 {
			break
		}
	}

	fmt.Printf("\nParsed %d articles\n", count)
}