package dalle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "github.com/TrueBlocks/trueblocks-sdk/v5"
)

func TestSeries_String(t *testing.T) {
	s := &Series{
		Suffix:  "test",
		Adverbs: []string{"quickly", "slowly"},
	}
	jsonStr := s.String()
	var out Series
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		t.Fatalf("String() did not return valid JSON: %v", err)
	}
	if out.Suffix != "test" || len(out.Adverbs) != 2 {
		t.Errorf("String() mismatch: %+v", out)
	}
}

func TestSeries_GetFilter_Valid(t *testing.T) {
	s := &Series{
		Adverbs: []string{"quickly", "slowly"},
		Nouns:   []string{"cat", "dog"},
	}
	adverbs, err := s.GetFilter("Adverbs")
	if err != nil {
		t.Fatalf("GetFilter returned error: %v", err)
	}
	if !reflect.DeepEqual(adverbs, []string{"quickly", "slowly"}) {
		t.Errorf("GetFilter returned wrong slice: %v", adverbs)
	}
	nouns, err := s.GetFilter("Nouns")
	if err != nil || len(nouns) != 2 {
		t.Errorf("GetFilter failed for Nouns: %v, %v", nouns, err)
	}
}

func TestSeries_GetFilter_InvalidField(t *testing.T) {
	s := &Series{}
	_, err := s.GetFilter("NotAField")
	if err == nil {
		t.Fatal("expected error for invalid field name")
	}
	if !strings.Contains(err.Error(), "not valid") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSeries_GetFilter_NotSlice(t *testing.T) {
	s := &Series{Suffix: "notaslice"}
	_, err := s.GetFilter("Suffix")
	if err == nil || err.Error() != "field Suffix not a slice" {
		t.Errorf("expected 'not a slice' error, got %v", err)
	}
}

func TestSeries_GetFilter_NotStringSlice(t *testing.T) {
	type SeriesWithInt struct {
		Ints []int
	}
	s := &SeriesWithInt{Ints: []int{1, 2, 3}}
	ref := reflect.ValueOf(s)
	field := reflect.Indirect(ref).FieldByName("Ints")
	if field.Kind() != reflect.Slice || field.Type().Elem().Kind() == reflect.String {
		t.Skip("Test only relevant if field is a non-string slice")
	}
	// Simulate GetFilter logic
	// if field.Type().Elem().Kind() != reflect.String {
	// 	// Should error
	// }
}

func TestSeries_Model(t *testing.T) {
	s := &Series{
		Suffix:     "suf",
		Last:       3,
		Deleted:    true,
		Adverbs:    []string{"quickly"},
		Adjectives: []string{"red"},
		Nouns:      []string{"cat"},
		ModifiedAt: time.Now().UTC().Format(time.RFC3339),
	}
	m := s.Model("", "", false, nil)
	if m.Data["suffix"] != "suf" || m.Data["last"].(int) != 3 {
		t.Fatalf("model data mismatch: %#v", m.Data)
	}
	// Order should contain known keys in order (spot check first / last)
	if len(m.Order) == 0 || m.Order[0] != "suffix" || m.Order[len(m.Order)-1] != "backstyles" {
		t.Fatalf("unexpected order: %v", m.Order)
	}
}

func TestSeries_StringAndSaveSeries(t *testing.T) {
	SetupTest(t, SetupTestOptions{})
	s := &Series{Suffix: "alpha", Last: 1, Adverbs: []string{"swiftly"}}
	// ensure JSON
	js := s.String()
	if !strings.Contains(js, "\"suffix\": \"alpha\"") {
		t.Fatalf("String() missing suffix: %s", js)
	}
	// Save with different last value
	s.SaveSeries("alpha", 42)
	fn := filepath.Join(seriesDir(), "alpha.json")
	b, err := os.ReadFile(fn)
	if err != nil {
		t.Fatalf("reading saved series: %v", err)
	}
	var out Series
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal saved: %v", err)
	}
	if out.Last != 42 {
		t.Fatalf("expected Last overwritten to 42, got %d", out.Last)
	}
}

// --- Tests for series_crud.go ---

func writeSeriesFile(t *testing.T, dir, suffix string, deleted bool, last int) {
	t.Helper()
	s := Series{Suffix: suffix, Deleted: deleted, Last: last}
	data, _ := json.Marshal(s)
	if err := os.WriteFile(filepath.Join(dir, suffix+".json"), data, 0o644); err != nil {
		t.Fatalf("write series file: %v", err)
	}
}

func TestLoadSeriesModelsAndVariants(t *testing.T) {
	tmp := t.TempDir()
	// valid
	writeSeriesFile(t, tmp, "one", false, 1)
	writeSeriesFile(t, tmp, "two", true, 2)
	// invalid extension
	_ = os.WriteFile(filepath.Join(tmp, "readme.txt"), []byte("ignore"), 0o644)
	// invalid json
	_ = os.WriteFile(filepath.Join(tmp, "bad.json"), []byte("{"), 0o644)

	all, err := LoadSeriesModels(tmp)
	if err != nil {
		t.Fatalf("LoadSeriesModels error: %v", err)
	}
	if len(all) != 2 { // only valid JSON files
		t.Fatalf("expected 2 valid series, got %d", len(all))
	}
	// Check ModifiedAt populated
	foundMod := false
	for _, s := range all {
		if s.Suffix == "one" && s.ModifiedAt != "" {
			foundMod = true
		}
	}
	if !foundMod {
		t.Fatalf("ModifiedAt not populated on at least one series: %#v", all)
	}

	active, _ := LoadActiveSeriesModels(tmp)
	if len(active) != 1 || active[0].Suffix != "one" {
		t.Fatalf("expected only 'one' active, got %#v", active)
	}
	deleted, _ := LoadDeletedSeriesModels(tmp)
	if len(deleted) != 1 || deleted[0].Suffix != "two" {
		t.Fatalf("expected only 'two' deleted, got %#v", deleted)
	}
}

func TestSortSeries(t *testing.T) {
	items := []Series{
		{Suffix: "b", Last: 2, ModifiedAt: "2025-01-02T00:00:00Z"},
		{Suffix: "a", Last: 3, ModifiedAt: "2025-01-01T00:00:00Z"},
		{Suffix: "c", Last: 1, ModifiedAt: "2025-01-03T00:00:00Z"},
	}
	// sort by suffix asc
	_ = SortSeries(items, sdk.SortSpec{Fields: []string{"suffix"}, Order: []sdk.SortOrder{sdk.Asc}})
	if items[0].Suffix != "a" || items[2].Suffix != "c" {
		t.Fatalf("suffix asc sort wrong: %#v", items)
	}
	// sort by last desc
	_ = SortSeries(items, sdk.SortSpec{Fields: []string{"last"}, Order: []sdk.SortOrder{sdk.Dec}})
	if items[0].Last != 3 || items[2].Last != 1 {
		t.Fatalf("last desc sort wrong: %#v", items)
	}
	// unknown field falls back to suffix
	_ = SortSeries(items, sdk.SortSpec{Fields: []string{"unknown"}})
	if items[0].Suffix != "a" {
		t.Fatalf("fallback sort expected a first: %#v", items)
	}
	// modifiedAt asc
	_ = SortSeries(items, sdk.SortSpec{Fields: []string{"modifiedAt"}, Order: []sdk.SortOrder{sdk.Asc}})
	if items[0].ModifiedAt != "2025-01-01T00:00:00Z" {
		t.Fatalf("modifiedAt asc sort wrong: %#v", items)
	}
}

func TestRemoveDeleteUndeleteSeries(t *testing.T) {
	SetupTest(t, SetupTestOptions{})
	// Prepare JSON file for suffix
	writeSeriesFile(t, seriesDir(), "s1", false, 0)
	// output dirs
	outDir := filepath.Join(OutputDir(), "s1")
	delDir := filepath.Join(OutputDir(), "s1.deleted")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.MkdirAll(delDir, 0o755); err != nil {
		t.Fatalf("mkdir del: %v", err)
	}
	if err := RemoveSeries(seriesDir(), "s1"); err != nil {
		t.Fatalf("RemoveSeries: %v", err)
	}
	if _, err := os.Stat(filepath.Join(seriesDir(), "s1.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected series file removed")
	}
	if _, err := os.Stat(outDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected output dir removed")
	}
	if _, err := os.Stat(delDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted dir removed")
	}

	// Recreate for delete / undelete cycle
	writeSeriesFile(t, seriesDir(), "s2", false, 0)
	out2 := filepath.Join(OutputDir(), "s2")
	if err := os.MkdirAll(out2, 0o755); err != nil {
		t.Fatalf("mkdir out2: %v", err)
	}
	if err := DeleteSeries(seriesDir(), "s2"); err != nil {
		t.Fatalf("DeleteSeries: %v", err)
	}
	// JSON should show Deleted true
	b, _ := os.ReadFile(filepath.Join(seriesDir(), "s2.json"))
	var s Series
	_ = json.Unmarshal(b, &s)
	if !s.Deleted {
		t.Fatalf("expected Deleted true after DeleteSeries")
	}
	if _, err := os.Stat(out2); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected original output dir moved")
	}
	if _, err := os.Stat(filepath.Join(OutputDir(), "s2.deleted")); err != nil {
		t.Fatalf("expected .deleted dir exists")
	}

	if err := UndeleteSeries(seriesDir(), "s2"); err != nil {
		t.Fatalf("UndeleteSeries: %v", err)
	}
	b, _ = os.ReadFile(filepath.Join(seriesDir(), "s2.json"))
	// Need a fresh variable because field omitted (omitempty) would not overwrite true value.
	var s2 Series
	_ = json.Unmarshal(b, &s2)
	if s2.Deleted {
		t.Fatalf("expected Deleted false after UndeleteSeries; got %+v", s2)
	}
	if _, err := os.Stat(filepath.Join(OutputDir(), "s2")); err != nil {
		t.Fatalf("expected output dir restored")
	}
	if _, err := os.Stat(filepath.Join(OutputDir(), "s2.deleted")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected .deleted dir gone")
	}
}
