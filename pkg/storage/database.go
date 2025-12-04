package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

//go:embed databases.tar.gz
var embeddedDbs []byte

// TODO: Can't we get these values from the .tar.gz file directly?

// GetAvailableDatabases returns the list of all available database names.
// This is the canonical source for database names used throughout the system.
func GetAvailableDatabases() []string {
	// Use prompt.DatabaseNames as the single source of truth
	// Note: This list may contain duplicates (e.g., "artstyles" appears twice)
	// Callers should deduplicate if needed for display purposes
	seen := make(map[string]bool)
	unique := make([]string, 0, len(prompt.DatabaseNames))
	for _, name := range prompt.DatabaseNames {
		if !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}
	return unique
}

// FormatDatabaseName converts internal database name to display name.
func FormatDatabaseName(dbName string) string {
	if len(dbName) == 0 {
		return dbName
	}
	return strings.ToUpper(dbName[:1]) + dbName[1:]
}

// TODO: Can't we get these values from the .tar.gz file directly?

// GetDatabaseDescription returns a human-readable description for a database.
func GetDatabaseDescription(dbName string) string {
	descriptions := map[string]string{
		"adverbs":      "Manner modifiers for actions",
		"adjectives":   "Descriptive attributes",
		"nouns":        "Core subjects and entities",
		"emotions":     "Emotional states and expressions",
		"occupations":  "Professional roles and vocations",
		"actions":      "Physical activities and poses",
		"artstyles":    "Artistic movements and styles",
		"litstyles":    "Literary styles and genres",
		"colors":       "Color palettes and values",
		"viewpoints":   "Camera angles and perspectives",
		"gazes":        "Gaze directions and focus",
		"backstyles":   "Background styles and treatments",
		"compositions": "Composition rules and structures",
	}
	if desc, ok := descriptions[dbName]; ok {
		return desc
	}
	return ""
}

// ReadDatabaseCSV extracts the named CSV file from the embedded .tar.gz and returns its lines.
func ReadDatabaseCSV(name string) ([]string, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(embeddedDbs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		needle := filepath.Join("databases", name)
		if hdr.Name == needle {
			// Limit decompression to 5MB per file to mitigate decompression bomb DoS (gosec G110)
			const maxDecompressedSize = 5 * 1024 * 1024
			var buf bytes.Buffer
			lr := &io.LimitedReader{R: tr, N: maxDecompressedSize + 1}
			if _, err := io.Copy(&buf, lr); err != nil {
				return nil, err
			}
			if lr.N <= 0 { // exceeded limit
				return nil, errors.New("embedded file too large: potential decompression bomb")
			}
			lines := strings.Split(strings.ReplaceAll(buf.String(), "\r\n", "\n"), "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			return lines, nil
		}
	}
	return nil, errors.New("file not found in archive: " + name)
}
