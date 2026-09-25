package prompt

import (
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
)

func TestDefaultConfigurationFollowsTheRegistry(t *testing.T) {
	t.Setenv("TB_DALLE_ENHANCEMENT_MODEL", "")
	t.Setenv("TB_DALLE_IMAGE_MODEL", "")
	for _, spend := range []string{ai.TierPro, ai.TierCheap} {
		t.Setenv("TB_DALLE_SPEND", spend)
		wantText, wantEffort, err := ai.TierCompose(spend)
		if err != nil {
			t.Fatal(err)
		}
		wantImage, err := ai.RoleModel(spend, ai.RoleImage)
		if err != nil {
			t.Fatal(err)
		}
		c := DefaultAiConfiguration()
		if c.EnhancementModel != wantText || c.EnhancementEffort != wantEffort || c.ImageModel != wantImage {
			t.Errorf("%s: got %q %q %q, want %q %q %q", spend, c.EnhancementModel, c.EnhancementEffort, c.ImageModel, wantText, wantEffort, wantImage)
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
