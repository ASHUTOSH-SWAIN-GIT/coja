package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromManifest_MergesSegments(t *testing.T) {
	dir := t.TempDir()

	seg1 := NewIndex()
	seg1.AddDocument(1, "Doc One", []TermPosition{
		{Term: "alpha", Position: 0},
		{Term: "beta", Position: 1},
	})
	if err := SaveToFile(filepath.Join(dir, "segment_000001.gob"), seg1); err != nil {
		t.Fatalf("save segment 1: %v", err)
	}

	seg2 := NewIndex()
	seg2.AddDocument(2, "Doc Two", []TermPosition{
		{Term: "beta", Position: 0},
		{Term: "gamma", Position: 1},
	})
	if err := SaveToFile(filepath.Join(dir, "segment_000002.gob"), seg2); err != nil {
		t.Fatalf("save segment 2: %v", err)
	}

	m := Manifest{
		Source: "test",
		Segments: []SegmentMeta{
			{File: "segment_000001.gob"},
			{File: "segment_000002.gob"},
		},
	}
	if err := writeManifest(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	merged, loadedManifest, err := LoadFromManifest(dir)
	if err != nil {
		t.Fatalf("load from manifest: %v", err)
	}

	if len(loadedManifest.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(loadedManifest.Segments))
	}
	if merged.TotalDocs != 2 {
		t.Fatalf("expected 2 docs, got %d", merged.TotalDocs)
	}
	if len(merged.PostingLists["beta"]) != 2 {
		t.Fatalf("expected 2 postings for beta, got %d", len(merged.PostingLists["beta"]))
	}
}

func TestLoadFromManifest_MissingSegment(t *testing.T) {
	dir := t.TempDir()

	m := Manifest{
		Source: "test",
		Segments: []SegmentMeta{
			{File: "segment_999999.gob"},
		},
	}
	if err := writeManifest(filepath.Join(dir, "manifest.json"), m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, _, err := LoadFromManifest(dir); err == nil {
		t.Fatalf("expected error when segment file is missing")
	}
}

func writeManifest(path string, m Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
