package storage

import (
	"io"
	"path/filepath"
	"strings"

	_ "embed"
)

//go:embed series.tar.gz
var embeddedSeries []byte

// SeriesArchiveHash returns the SHA256 hash of the embedded series archive.
func SeriesArchiveHash() string {
	return hashBytes(embeddedSeries)
}

// WalkSeriesArchive visits every JSON file in the embedded series archive.
func WalkSeriesArchive(visit func(path string, body []byte) error) error {
	return walkArchiveFiles(embeddedSeries, func(path string, body []byte) error {
		if isAppleDoubleArchivePath(path) {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		return visit(path, body)
	})
}

// ListEmbeddedSeriesSuffixes returns the suffixes of all built-in series.
func ListEmbeddedSeriesSuffixes() ([]string, error) {
	var suffixes []string
	err := WalkSeriesArchive(func(path string, body []byte) error {
		suffixes = append(suffixes, seriesSuffixFromPath(path))
		return nil
	})
	return suffixes, err
}

// ReadEmbeddedSeriesJSON returns the raw JSON body for a built-in series, if present.
func ReadEmbeddedSeriesJSON(suffix string) ([]byte, bool) {
	var body []byte
	found := false
	_ = WalkSeriesArchive(func(path string, b []byte) error {
		if seriesSuffixFromPath(path) == suffix {
			body = b
			found = true
			return io.EOF
		}
		return nil
	})
	return body, found
}

func seriesSuffixFromPath(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
