package dalle

import (
	"testing"
)

func TestLoadSeries_NewAndExisting(t *testing.T) {
	ctx := &Context{}
	ResetFileMocks()
	oldEstablish := establishFolder
	oldAsciiFileToString := asciiFileToString
	oldFileExists := fileExists
	defer func() {
		establishFolder = oldEstablish
		asciiFileToString = oldAsciiFileToString
		fileExists = oldFileExists
	}()

	establishFolder = MockEstablishFolder
	asciiFileToString = func(_ string) string { return "{}" }
	fileExists = func(path string) bool {
		MockFileExistsCalled = true
		return false
	}

	series, err := ctx.LoadSeries()
	if err != nil {
		t.Fatalf("LoadSeries failed: %v", err)
	}
	if series.Suffix != "simple" {
		t.Errorf("Expected Suffix 'simple', got %q", series.Suffix)
	}
	if !MockFileExistsCalled {
		t.Error("Expected file existence check to be called")
	}
}

func TestToLines_EmptyAndFiltered(t *testing.T) {
	ctx := &Context{Series: Series{Nouns: []string{"cat", "dog"}}}
	oldAsciiFileToLines := asciiFileToLines
	asciiFileToLines = func(_ string) []string { return []string{"header", "cat", "dog"} }
	defer func() { asciiFileToLines = oldAsciiFileToLines }()
	// Simulate lines with and without filtering
	lines, err := ctx.toLines("nouns")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(lines) == 0 {
		t.Error("Expected at least one line (should append 'none' if empty)")
	}
}
