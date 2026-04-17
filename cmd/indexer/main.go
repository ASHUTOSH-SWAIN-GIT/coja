package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"coja/pkg/index"
	"coja/pkg/parser"
	"coja/pkg/tokenizer"
)

type indexedDoc struct {
	DocID       int
	Title       string
	BodyTokens  []index.TermPosition
	TitleTokens []index.TermPosition
	IntroTokens []index.TermPosition
}

type segmentMeta struct {
	File        string `json:"file"`
	Docs        int    `json:"docs"`
	UniqueTerms int    `json:"unique_terms"`
	TotalTokens int64  `json:"total_tokens"`
}

type manifest struct {
	Source         string        `json:"source"`
	CreatedAt      time.Time     `json:"created_at"`
	Workers        int           `json:"workers"`
	CheckpointDocs int           `json:"checkpoint_docs"`
	Segments       []segmentMeta `json:"segments"`
	TotalDocs      int           `json:"total_docs"`
	TotalTokens    int64         `json:"total_tokens"`
}

func main() {
	workers := flag.Int("workers", runtime.NumCPU(), "number of text processing workers")
	checkpointDocs := flag.Int("checkpoint-docs", 100000, "documents per checkpoint segment")
	outputDir := flag.String("out-dir", "data/index", "directory where segment and manifest files are saved")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("usage: go run cmd/indexer/main.go [flags] <path-to-dump.xml[.bz2]|->")
		fmt.Println("example (parallel decompress): pbzip2 -dc dump.xml.bz2 | go run cmd/indexer/main.go -")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *workers < 1 {
		fmt.Println("workers must be >= 1")
		os.Exit(1)
	}
	if *checkpointDocs < 1 {
		fmt.Println("checkpoint-docs must be >= 1")
		os.Exit(1)
	}

	input := flag.Arg(0)

	start := time.Now()
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Println("create output directory error:", err)
		os.Exit(1)
	}

	rawDocs := make(chan parser.Document, 1024)
	processedDocs := make(chan indexedDoc, 2048)
	parseErrCh := make(chan error, 1)

	go func() {
		parseErrCh <- parser.Parse(input, rawDocs)
	}()

	var workersWG sync.WaitGroup
	for i := 0; i < *workers; i++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for doc := range rawDocs {
				clean := parser.StripWikitext(doc.Text)
				intro := parser.ExtractIntro(clean)
				bodyTokens := tokenizer.Tokenize(clean)
				titleTokens := tokenizer.Tokenize(doc.Title)
				introTokens := tokenizer.Tokenize(intro)

				indexBodyTokens := make([]index.TermPosition, len(bodyTokens))
				for i, t := range bodyTokens {
					indexBodyTokens[i] = index.TermPosition{
						Term:     t.Term,
						Position: t.Position,
					}
				}
				indexTitleTokens := make([]index.TermPosition, len(titleTokens))
				for i, t := range titleTokens {
					indexTitleTokens[i] = index.TermPosition{
						Term:     t.Term,
						Position: t.Position,
					}
				}
				indexIntroTokens := make([]index.TermPosition, len(introTokens))
				for i, t := range introTokens {
					indexIntroTokens[i] = index.TermPosition{
						Term:     t.Term,
						Position: t.Position,
					}
				}

				processedDocs <- indexedDoc{
					DocID:       doc.ID,
					Title:       doc.Title,
					BodyTokens:  indexBodyTokens,
					TitleTokens: indexTitleTokens,
					IntroTokens: indexIntroTokens,
				}
			}
		}()
	}

	go func() {
		workersWG.Wait()
		close(processedDocs)
	}()

	currentSegment := index.NewIndex()
	var savedSegments []segmentMeta
	segmentNumber := 1
	totalDocs := 0
	var totalTokens int64

	for processed := range processedDocs {
		currentSegment.AddDocument(processed.DocID, processed.Title, processed.BodyTokens, processed.TitleTokens, processed.IntroTokens)
		totalDocs++

		if totalDocs%10000 == 0 {
			fmt.Printf("Processed %d docs (%s)\n", totalDocs, time.Since(start))
		}

		if currentSegment.TotalDocs >= *checkpointDocs {
			meta, err := saveSegment(*outputDir, segmentNumber, currentSegment)
			if err != nil {
				fmt.Println("save segment error:", err)
				os.Exit(1)
			}

			savedSegments = append(savedSegments, meta)
			totalTokens += currentSegment.TotalTokens
			fmt.Printf("Saved segment %d (%d docs, %d terms) -> %s\n", segmentNumber, meta.Docs, meta.UniqueTerms, meta.File)

			segmentNumber++
			currentSegment = index.NewIndex()
		}
	}

	if currentSegment.TotalDocs > 0 {
		meta, err := saveSegment(*outputDir, segmentNumber, currentSegment)
		if err != nil {
			fmt.Println("save segment error:", err)
			os.Exit(1)
		}

		savedSegments = append(savedSegments, meta)
		totalTokens += currentSegment.TotalTokens
		fmt.Printf("Saved segment %d (%d docs, %d terms) -> %s\n", segmentNumber, meta.Docs, meta.UniqueTerms, meta.File)
	}

	parseErr := <-parseErrCh
	if parseErr != nil {
		fmt.Println("parse error:", parseErr)
		os.Exit(1)
	}

	manifestPath := filepath.Join(*outputDir, "manifest.json")
	projectManifest := manifest{
		Source:         input,
		CreatedAt:      time.Now().UTC(),
		Workers:        *workers,
		CheckpointDocs: *checkpointDocs,
		Segments:       savedSegments,
		TotalDocs:      totalDocs,
		TotalTokens:    totalTokens,
	}

	manifestBytes, err := json.MarshalIndent(projectManifest, "", "  ")
	if err != nil {
		fmt.Println("manifest encode error:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		fmt.Println("manifest write error:", err)
		os.Exit(1)
	}

	fmt.Printf("\nDone in %s\n", time.Since(start))
	fmt.Printf("Total docs indexed: %d\n", totalDocs)
	fmt.Printf("Total tokens indexed: %d\n", totalTokens)
	fmt.Printf("Segments written: %d\n", len(savedSegments))
	fmt.Printf("Manifest: %s\n", manifestPath)
}

func saveSegment(outputDir string, segmentNumber int, idx *index.Index) (segmentMeta, error) {
	fileName := fmt.Sprintf("segment_%06d.gob", segmentNumber)
	filePath := filepath.Join(outputDir, fileName)

	if err := index.SaveToFile(filePath, idx); err != nil {
		return segmentMeta{}, err
	}

	return segmentMeta{
		File:        fileName,
		Docs:        idx.TotalDocs,
		UniqueTerms: len(idx.PostingLists),
		TotalTokens: idx.TotalTokens,
	}, nil
}
