package dalle

import (
	"testing"
)

// Mock file and logger dependencies for database.go
var (
	dbMockEstablishFolderCalled   bool
	dbMockAsciiFileToStringCalled bool
	dbMockFileExistsCalled        bool
)

func dbMockEstablishFolder(_ string) error {
	dbMockEstablishFolderCalled = true
	return nil
}

func dbMockAsciiFileToString(_ string) string {
	dbMockAsciiFileToStringCalled = true
	return "{\"suffix\":\"test\"}"
}

func dbMockFileExists(_ string) bool {
	dbMockFileExistsCalled = true
	return false // Simulate file does not exist
}

func dbMockAsciiFileToLines(_ string) []string {
	return []string{"header", "cat", "dog"}
}

func resetDbMocks() {
	dbMockEstablishFolderCalled = false
	dbMockAsciiFileToStringCalled = false
	dbMockFileExistsCalled = false
}

func TestLoadSeries_NewAndExisting(t *testing.T) {
	ctx := &Context{}
	oldEstablish := establishFolder
	oldAsciiFileToString := asciiFileToString
	oldFileExists := fileExists
	defer func() {
		establishFolder = oldEstablish
		asciiFileToString = oldAsciiFileToString
		fileExists = oldFileExists
	}()

	establishFolder = dbMockEstablishFolder
	asciiFileToString = dbMockAsciiFileToString
	fileExists = dbMockFileExists

	resetDbMocks()
	series, err := ctx.LoadSeries()
	if err != nil {
		t.Fatalf("LoadSeries failed: %v", err)
	}
	if series.Suffix != "simple" {
		t.Errorf("Expected Suffix 'simple', got %q", series.Suffix)
	}
	if !dbMockFileExistsCalled {
		t.Error("Expected file existence check to be called")
	}
}

func TestToLines_EmptyAndFiltered(t *testing.T) {
	ctx := &Context{Series: Series{Nouns: []string{"cat", "dog"}}}
	oldAsciiFileToLines := asciiFileToLines
	asciiFileToLines = dbMockAsciiFileToLines
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
