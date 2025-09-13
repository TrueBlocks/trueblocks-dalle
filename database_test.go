package dalle

import (
	"testing"
)

func TestReloadDatabases_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	TestOnlyResetDataDir()
	ConfigureDataDir(tmpDir)
	ctx := NewContext()

	// Provide DatabaseNames for the test
	DatabaseNames = []string{"nouns"}

	if err := ctx.ReloadDatabases("empty"); err != nil {
		t.Fatalf("error reloading database: %v", err)
	}

	if len(ctx.Databases) == 0 {
		t.Error("Databases not loaded")
	}
	if got := ctx.Databases["nouns"]; len(got) == 0 {
		t.Errorf("Database 'nouns' is empty: %v", got)
	}
}
