package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
)

// installRoleTables writes the test catalog with a shared role table and
// dalle's own tool_defaults row, so the test never reads the installed one.
func installRoleTables(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ai.DecodeModelCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	catalog.RoleDefaults = ai.RoleDefaults{
		Cheap: ai.RoleTier{Research: "claude-haiku-4-5-20251001", Compose: "claude-haiku-4-5-20251001", Image: "gemini-3.1-flash-image"},
		Pro:   ai.RoleTier{Research: "claude-opus-5", Compose: "claude-opus-5", ComposeEffort: "high", Image: "gemini-3-pro-image"},
	}
	// dalle overrides both pro slots but only the cheap image slot.
	catalog.ToolDefaults = map[string]ai.RoleDefaults{"dalle": {
		Cheap: ai.RoleTier{Image: "gpt-image-2"},
		Pro:   ai.RoleTier{Compose: "gpt-5.5", Image: "gpt-image-2"},
	}}
	if data, err = json.Marshal(catalog); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	if err := os.WriteFile(ai.ModelsPath(), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfigurationFollowsTheRegistry(t *testing.T) {
	installRoleTables(t)
	t.Setenv("TB_DALLE_ENHANCEMENT_MODEL", "")
	t.Setenv("TB_DALLE_IMAGE_MODEL", "")
	for _, test := range []struct {
		spend, text, effort, image string
	}{
		{ai.TierPro, "gpt-5.5", "", "gpt-image-2"},
		{ai.TierCheap, "claude-haiku-4-5-20251001", "", "gpt-image-2"},
	} {
		t.Setenv("TB_DALLE_SPEND", test.spend)
		c := DefaultAiConfiguration()
		if c.EnhancementModel != test.text || c.EnhancementEffort != test.effort || c.ImageModel != test.image {
			t.Errorf("%s: got %q %q %q, want %q %q %q", test.spend, c.EnhancementModel, c.EnhancementEffort, c.ImageModel, test.text, test.effort, test.image)
		}
	}
	t.Setenv("TB_DALLE_SPEND", ai.TierPro)
	t.Setenv("TB_DALLE_ENHANCEMENT_MODEL", "gpt-5.5")
	if c := DefaultAiConfiguration(); c.EnhancementModel != "gpt-5.5" || c.EnhancementEffort != "" {
		t.Errorf("override: %q %q, want gpt-5.5 with no effort", c.EnhancementModel, c.EnhancementEffort)
	}
}

func TestCheckModels(t *testing.T) {
	ok := AiConfiguration{EnhancementModel: "gpt-5.5", ImageModel: "gpt-image-2"}
	if err := ok.CheckModels(); err != nil {
		t.Fatalf("valid pair refused: %v", err)
	}
	bad := ok
	bad.ImageModel = "nope"
	if err := bad.CheckModels(); err == nil || !strings.Contains(err.Error(), "gpt-image-2") {
		t.Errorf("bad image model: %v, want the valid list", err)
	}
	bad = ok
	bad.ImageModel = "gpt-5.5"
	if err := bad.CheckModels(); err == nil {
		t.Error("a writer accepted as an image model")
	}
	bad = ok
	bad.EnhancementModel = "nope"
	if err := bad.CheckModels(); err == nil || !strings.Contains(err.Error(), "gpt-5.5") {
		t.Errorf("bad enhancement model: %v, want the valid list", err)
	}
}
