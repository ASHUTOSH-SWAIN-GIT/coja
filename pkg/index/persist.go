package index

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
)

// SaveToFile writes the full in-memory index as a gob file.
func SaveToFile(path string, idx *Index) error {
	if idx == nil {
		return fmt.Errorf("nil index")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(idx); err != nil {
		return fmt.Errorf("encode index: %w", err)
	}

	return nil
}

// LoadFromFile reads an index gob file from disk.
func LoadFromFile(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open index file: %w", err)
	}
	defer f.Close()

	var idx Index
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&idx); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}

	if idx.PostingLists == nil {
		idx.PostingLists = make(map[string][]Posting)
	}
	if idx.TitlePostingLists == nil {
		idx.TitlePostingLists = make(map[string][]Posting)
	}
	if idx.DocStore == nil {
		idx.DocStore = make(map[int]DocInfo)
	}

	return &idx, nil
}
