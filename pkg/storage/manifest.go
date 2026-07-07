package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultDatabaseVersion = "1.0.0"

type DatabaseArchiveManifest struct {
	Version     string                 `json:"version"`
	ArchiveHash string                 `json:"archiveHash"`
	Files       []DatabaseFileManifest `json:"files"`
}

type DatabaseFileManifest struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Hash    string   `json:"hash"`
	Rows    int      `json:"rows"`
	Columns []string `json:"columns"`
}

func EmbeddedArchiveHash() string {
	return hashBytes(embeddedDbs)
}

func LoadEmbeddedArchiveManifest() (DatabaseArchiveManifest, error) {
	manifest, err := readEmbeddedManifest()
	if err == nil {
		if manifest.ArchiveHash == "" {
			manifest.ArchiveHash = EmbeddedArchiveHash()
		}
		return manifest, ValidateDatabaseArchiveManifest(manifest)
	}
	return BuildEmbeddedArchiveManifest()
}

func BuildEmbeddedArchiveManifest() (DatabaseArchiveManifest, error) {
	return BuildDatabaseArchiveManifest(embeddedDbs, DefaultDatabaseVersion)
}

func BuildDatabaseArchiveManifest(archive []byte, version string) (DatabaseArchiveManifest, error) {
	manifest := DatabaseArchiveManifest{
		Version:     strings.TrimSpace(version),
		ArchiveHash: hashBytes(archive),
		Files:       []DatabaseFileManifest{},
	}
	if manifest.Version == "" {
		manifest.Version = DefaultDatabaseVersion
	}
	if err := walkArchiveFiles(archive, func(path string, body []byte) error {
		if filepath.Base(path) == "manifest.json" || filepath.Ext(path) != ".csv" {
			return nil
		}
		fileManifest, err := buildFileManifest(path, body)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, fileManifest)
		return nil
	}); err != nil {
		return DatabaseArchiveManifest{}, err
	}
	sort.Slice(manifest.Files, func(left, right int) bool {
		return manifest.Files[left].Path < manifest.Files[right].Path
	})
	return manifest, ValidateDatabaseArchiveManifest(manifest)
}

func ValidateDatabaseArchiveManifest(manifest DatabaseArchiveManifest) error {
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("database manifest version is required")
	}
	if !strings.HasPrefix(manifest.ArchiveHash, "sha256:") {
		return fmt.Errorf("database manifest archive hash must use sha256 prefix")
	}
	if len(manifest.Files) == 0 {
		return fmt.Errorf("database manifest has no files")
	}
	seen := map[string]bool{}
	for _, fileManifest := range manifest.Files {
		if fileManifest.Name == "" {
			return fmt.Errorf("database manifest file name is required")
		}
		if fileManifest.Path == "" {
			return fmt.Errorf("database manifest file path is required")
		}
		if seen[fileManifest.Path] {
			return fmt.Errorf("database manifest has duplicate file path: %s", fileManifest.Path)
		}
		seen[fileManifest.Path] = true
		if !strings.HasPrefix(fileManifest.Hash, "sha256:") {
			return fmt.Errorf("database manifest file hash must use sha256 prefix: %s", fileManifest.Path)
		}
		if fileManifest.Rows < 0 {
			return fmt.Errorf("database manifest file rows cannot be negative: %s", fileManifest.Path)
		}
		if len(fileManifest.Columns) == 0 {
			return fmt.Errorf("database manifest file columns are required: %s", fileManifest.Path)
		}
	}
	return nil
}

func readEmbeddedManifest() (DatabaseArchiveManifest, error) {
	var manifest DatabaseArchiveManifest
	err := walkArchiveFiles(embeddedDbs, func(path string, body []byte) error {
		if filepath.Base(path) != "manifest.json" {
			return nil
		}
		if err := json.Unmarshal(body, &manifest); err != nil {
			return err
		}
		return io.EOF
	})
	if err != nil && err != io.EOF {
		return DatabaseArchiveManifest{}, err
	}
	if manifest.Version == "" {
		return DatabaseArchiveManifest{}, fmt.Errorf("embedded manifest not found")
	}
	return manifest, nil
}

func buildFileManifest(path string, body []byte) (DatabaseFileManifest, error) {
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return DatabaseFileManifest{}, fmt.Errorf("read csv %s: %w", path, err)
	}
	columns := []string{}
	rows := 0
	if len(records) > 0 {
		columns = records[0]
		rows = len(records) - 1
	}
	return DatabaseFileManifest{
		Name:    strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Path:    filepath.ToSlash(path),
		Hash:    hashBytes(body),
		Rows:    rows,
		Columns: columns,
	}, nil
}

func walkArchiveFiles(archive []byte, visit func(path string, body []byte) error) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			continue
		}
		const maxDecompressedSize = 5 * 1024 * 1024
		var buffer bytes.Buffer
		limitedReader := &io.LimitedReader{R: tarReader, N: maxDecompressedSize + 1}
		if _, err := io.Copy(&buffer, limitedReader); err != nil {
			return err
		}
		if limitedReader.N <= 0 {
			return fmt.Errorf("embedded file too large: %s", header.Name)
		}
		if err := visit(filepath.ToSlash(header.Name), buffer.Bytes()); err != nil {
			return err
		}
	}
}

func hashBytes(body []byte) string {
	digest := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(digest[:])
}
