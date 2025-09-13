package dalle

import (
	"testing"

	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/prompt"
)

func TestReloadDatabases_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	TestOnlyResetDataDir()
	ConfigureDataDir(tmpDir)
	ctx := NewContext()

	// Use prompt.DatabaseNames for the test context
	_ = prompt.DatabaseNames

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
