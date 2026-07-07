package storage

import "testing"

func TestLoadEmbeddedArchiveManifest(t *testing.T) {
	manifest, err := LoadEmbeddedArchiveManifest()
	if err != nil {
		t.Fatalf("LoadEmbeddedArchiveManifest: %v", err)
	}
	if manifest.Version == "" {
		t.Fatalf("expected manifest version")
	}
	if manifest.ArchiveHash == "" {
		t.Fatalf("expected archive hash")
	}
	if len(manifest.Files) == 0 {
		t.Fatalf("expected database file manifests")
	}

	foundNouns := false
	for _, file := range manifest.Files {
		if file.Name == "nouns" {
			foundNouns = true
			if file.Rows == 0 {
				t.Fatalf("expected nouns rows")
			}
			if len(file.Columns) == 0 {
				t.Fatalf("expected nouns columns")
			}
		}
	}
	if !foundNouns {
		t.Fatalf("expected nouns database in manifest: %#v", manifest.Files)
	}
}

func TestValidateDatabaseArchiveManifest(t *testing.T) {
	manifest := DatabaseArchiveManifest{
		Version:     "1.0.0",
		ArchiveHash: "sha256:archive",
		Files: []DatabaseFileManifest{{
			Name:    "nouns",
			Path:    "databases/nouns.csv",
			Hash:    "sha256:file",
			Rows:    1,
			Columns: []string{"version", "name"},
		}},
	}
	if err := ValidateDatabaseArchiveManifest(manifest); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}

	manifest.Files[0].Columns = nil
	if err := ValidateDatabaseArchiveManifest(manifest); err == nil {
		t.Fatalf("expected missing columns error")
	}
}
