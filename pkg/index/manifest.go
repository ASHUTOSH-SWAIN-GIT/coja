package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SegmentMeta describes one persisted segment in manifest.json.
type SegmentMeta struct {
	File        string `json:"file"`
	Docs        int    `json:"docs"`
	UniqueTerms int    `json:"unique_terms"`
	TotalTokens int64  `json:"total_tokens"`
}

// Manifest describes all saved segments for one indexing run.
type Manifest struct {
	Source         string        `json:"source"`
	Workers        int           `json:"workers"`
	CheckpointDocs int           `json:"checkpoint_docs"`
	Segments       []SegmentMeta `json:"segments"`
	TotalDocs      int           `json:"total_docs"`
	TotalTokens    int64         `json:"total_tokens"`
}

// LoadManifest reads manifest.json from indexDir.
func LoadManifest(indexDir string) (*Manifest, error) {
	manifestPath := filepath.Join(indexDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if len(m.Segments) == 0 {
		return nil, fmt.Errorf("manifest has no segments")
	}

	return &m, nil
}

// LoadFromManifest loads and merges all segments referenced in manifest.json.
func LoadFromManifest(indexDir string) (*Index, *Manifest, error) {
	m, err := LoadManifest(indexDir)
	if err != nil {
		return nil, nil, err
	}

	merged := NewIndex()
	for _, seg := range m.Segments {
		segmentPath := filepath.Join(indexDir, seg.File)
		segmentIndex, err := LoadFromFile(segmentPath)
		if err != nil {
			return nil, nil, fmt.Errorf("load segment %q: %w", seg.File, err)
		}
		merged.Merge(segmentIndex)
	}

	return merged, m, nil
}
