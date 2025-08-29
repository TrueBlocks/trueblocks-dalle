package dalle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	sdk "github.com/TrueBlocks/trueblocks-sdk/v5"
)

// LoadSeriesModels loads all series JSON files from the series folder beneath the provided dataDir
func LoadSeriesModels(seriesDir string) ([]Series, error) {
	items := []Series{}
	entries, err := os.ReadDir(seriesDir)
	if err != nil {
		return items, nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		full := filepath.Join(seriesDir, name)
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		var s Series
		if err := json.Unmarshal(b, &s); err != nil {
			continue
		}
		fi, err := os.Stat(full)
		if err == nil {
			s.ModifiedAt = fi.ModTime().UTC().Format(time.RFC3339)
		}
		items = append(items, s)
	}
	return items, nil
}

// SortSeries sorts in place based on field in spec (suffix, modifiedAt, last)
func SortSeries(items []Series, sortSpec sdk.SortSpec) error {
	if len(items) < 2 || len(sortSpec.Fields) == 0 {
		return nil
	}
	if len(sortSpec.Order) == 0 {
		sortSpec.Order = append(sortSpec.Order, sdk.Asc)
	}
	field := sortSpec.Fields[0]
	asc := sortSpec.Order[0] == sdk.Asc
	cmp := func(i, j int) bool { return true }
	switch strings.ToLower(field) {
	case "suffix":
		cmp = func(i, j int) bool { return strings.Compare(items[i].Suffix, items[j].Suffix) < 0 }
	case "modifiedat":
		cmp = func(i, j int) bool { return items[i].ModifiedAt < items[j].ModifiedAt }
	case "last":
		cmp = func(i, j int) bool { return items[i].Last < items[j].Last }
	default:
		cmp = func(i, j int) bool { return strings.Compare(items[i].Suffix, items[j].Suffix) < 0 }
	}
	sort.SliceStable(items, func(i, j int) bool {
		if asc {
			return cmp(i, j)
		}
		return !cmp(i, j)
	})
	return nil
}

func DeleteSeries(seriesDir, suffix string) error {
	if suffix == "" {
		return errors.New("empty suffix")
	}
	fn := filepath.Join(seriesDir, suffix+".json")
	if _, err := os.Stat(fn); err != nil {
		return err
	}
	return os.Remove(fn)
}

func DuplicateSeries(seriesDir, fromSuffix, toSuffix string) (*Series, error) {
	if fromSuffix == "" || toSuffix == "" {
		return nil, errors.New("empty suffix")
	}
	if strings.EqualFold(fromSuffix, toSuffix) {
		return nil, errors.New("duplicate target equals source")
	}
	fromFile := filepath.Join(seriesDir, fromSuffix+".json")
	toFile := filepath.Join(seriesDir, toSuffix+".json")
	if _, err := os.Stat(fromFile); err != nil {
		return nil, err
	}
	if _, err := os.Stat(toFile); err == nil {
		return nil, errors.New("target exists")
	}
	b, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, err
	}
	var s Series
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	s.Suffix = toSuffix
	fi, err := os.Stat(fromFile)
	if err == nil {
		s.ModifiedAt = fi.ModTime().UTC().Format(time.RFC3339)
	}
	_ = file.EstablishFolder(seriesDir)
	if err := os.WriteFile(toFile, []byte(s.String()), 0o644); err != nil {
		return nil, err
	}
	return &s, nil
}
