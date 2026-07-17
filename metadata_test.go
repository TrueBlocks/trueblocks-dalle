package dalle

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestImageMetadataRoundTripAndList(t *testing.T) {
	dataDir := t.TempDir()
	metadata := NewImageMetadata("Person Tour Coordinates", "seed-123", "empty")
	metadata.Database = MetadataDatabase{Version: "1.0.0", ArchiveHash: "sha256:archive"}
	metadata.Series.Hash = "sha256:series"
	metadata.Series.Source = "embedded"
	metadata.SelectedRecords = []SelectedRecord{{Attribute: "noun", Database: "nouns", RowIndex: 4, Record: "person"}}
	metadata.Prompts = PromptSet{Prompt: "paint a person"}
	metadata.Artifacts = ArtifactSet{Generated: "output/empty/generated/seed-123.png"}
	metadata.Stages.Selected = StageStatus{Status: "complete"}
	metadata.Status.Completed = true

	path, err := WriteImageMetadata(dataDir, metadata)
	if err != nil {
		t.Fatalf("WriteImageMetadata: %v", err)
	}
	if path != filepath.Join(dataDir, "output", "empty", "metadata", "seed-123.json") {
		t.Fatalf("unexpected metadata path: %s", path)
	}

	loaded, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata: %v", err)
	}
	if loaded.Input != metadata.Input || loaded.Seed != metadata.Seed {
		t.Fatalf("round-trip mismatch: %#v", loaded)
	}
	if loaded.ImageID == "" {
		t.Fatalf("expected image ID to be computed")
	}

	records, err := ListImageMetadata(dataDir, ImageFilter{Series: "empty"})
	if err != nil {
		t.Fatalf("ListImageMetadata: %v", err)
	}
	if len(records) != 1 || records[0].Metadata.Seed != "seed-123" {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestCheckRegenerationCompatibility(t *testing.T) {
	metadata := NewImageMetadata("input", "seed", "empty")
	metadata.Database = MetadataDatabase{Version: "1.0.0", ArchiveHash: "sha256:old"}

	if err := CheckRegenerationCompatibility(metadata, "1.0.0", "sha256:old"); err != nil {
		t.Fatalf("expected compatibility: %v", err)
	}

	err := CheckRegenerationCompatibility(metadata, "2.0.0", "sha256:old")
	if ErrorCodeOf(err) != ErrRegenerationRefused {
		t.Fatalf("expected regeneration refusal, got %v", err)
	}
	var coded *Error
	if !errors.As(err, &coded) || coded.Code != ErrRegenerationRefused {
		t.Fatalf("expected typed regeneration error, got %v", err)
	}
}

// The gallery groups an input's variants together, so listing must order by seed
// phrase first and series second. Sorting by Path would group by series, since
// metadata lives at output/<series>/metadata/<seed>.json.
func TestListImageMetadataOrdersBySeedThenSeries(t *testing.T) {
	dataDir := t.TempDir()

	// Written in an order that is neither the input order nor the path order, so a
	// passing result cannot come from insertion order.
	for _, entry := range []struct{ input, seed, series string }{
		{"zebra phrase", "seed-z", "alpha-series"},
		{"apple phrase", "seed-a", "omega-series"},
		{"apple phrase", "seed-a", "alpha-series"},
		{"zebra phrase", "seed-z", "omega-series"},
	} {
		metadata := NewImageMetadata(entry.input, entry.seed, entry.series)
		if _, err := WriteImageMetadata(dataDir, metadata); err != nil {
			t.Fatalf("WriteImageMetadata: %v", err)
		}
	}

	records, err := ListImageMetadata(dataDir, ImageFilter{})
	if err != nil {
		t.Fatalf("ListImageMetadata: %v", err)
	}

	want := []struct{ input, series string }{
		{"apple phrase", "alpha-series"},
		{"apple phrase", "omega-series"},
		{"zebra phrase", "alpha-series"},
		{"zebra phrase", "omega-series"},
	}
	if len(records) != len(want) {
		t.Fatalf("got %d records, want %d", len(records), len(want))
	}
	for i, expected := range want {
		got := records[i]
		if got.Metadata.Input != expected.input || got.Metadata.Series.Name != expected.series {
			t.Errorf("record %d = (%q, %q), want (%q, %q)",
				i, got.Metadata.Input, got.Metadata.Series.Name, expected.input, expected.series)
		}
	}
}
