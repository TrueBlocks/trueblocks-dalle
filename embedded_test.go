package dalle

import "testing"

func TestToLines_EmptyAndFiltered(t *testing.T) {
	lines, err := readDatabaseCSV("nouns.csv")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(lines) == 0 {
		t.Error("Expected at least one line (should append 'none' if empty)")
	}
}
