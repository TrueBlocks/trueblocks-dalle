package dalle

import (
	"testing"
)

func TestLoadSeries_NewAndExisting(t *testing.T) {
	tmpDir := t.TempDir()
	TestOnlyResetDataDir()
	ConfigureDataDir(tmpDir)
	ctx := NewContext()

	series, err := ctx.LoadSeries()
	if err != nil {
		t.Fatalf("LoadSeries failed: %v", err)
	}
	if series.Suffix != "empty" {
		t.Errorf("Expected Suffix 'empty', got %q", series.Suffix)
	}
}

func TestToLines_EmptyAndFiltered(t *testing.T) {
	lines, err := readDatabaseCSV("nouns.csv")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(lines) == 0 {
		t.Error("Expected at least one line (should append 'none' if empty)")
	}
}

func TestReloadDatabases_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	TestOnlyResetDataDir()
	ConfigureDataDir(tmpDir)
	ctx := NewContext()

	// Provide DatabaseNames for the test
	DatabaseNames = []string{"nouns"}

	if err := ctx.ReloadDatabases(); err != nil {
		t.Fatalf("ReloadDatabases error: %v", err)
	}

	if len(ctx.Databases) == 0 {
		t.Error("Databases not loaded")
	}
	if got := ctx.Databases["nouns"]; len(got) == 0 {
		t.Errorf("Database 'nouns' is empty: %v", got)
	}
}
