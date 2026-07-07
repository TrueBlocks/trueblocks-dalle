package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

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

func TestBuildDatabaseArchiveManifestSkipsAppleDoubleFiles(t *testing.T) {
	archive := buildTestDatabaseArchive(t, map[string]string{
		"databases/nouns.csv":        "version,name\n1,book\n",
		"databases/._nouns.csv":      "appledouble",
		"__MACOSX/._nouns.csv":       "appledouble",
		"databases/manifest.json":    "{}",
		"databases/adjectives.csv":   "version,name\n1,bright\n",
		"databases/._adjectives.csv": "appledouble",
	})
	manifest, err := BuildDatabaseArchiveManifest(archive, "test")
	if err != nil {
		t.Fatalf("BuildDatabaseArchiveManifest: %v", err)
	}
	if len(manifest.Files) != 2 {
		t.Fatalf("expected two real database files, got %#v", manifest.Files)
	}
	for _, file := range manifest.Files {
		if isAppleDoubleArchivePath(file.Path) || file.Name == "manifest" {
			t.Fatalf("unexpected sidecar or manifest file: %#v", file)
		}
	}
}

func buildTestDatabaseArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for path, body := range files {
		contents := []byte(body)
		if err := tarWriter.WriteHeader(&tar.Header{Name: path, Mode: 0o600, Size: int64(len(contents))}); err != nil {
			t.Fatalf("WriteHeader: %v", err)
		}
		if _, err := tarWriter.Write(contents); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buffer.Bytes()
}
