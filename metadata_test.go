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
