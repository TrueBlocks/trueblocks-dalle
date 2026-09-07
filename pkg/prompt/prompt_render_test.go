package prompt

import (
	"strings"
	"testing"
)

func TestEnhanceInstructionMustHold(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	out, err := EnhanceInstruction.Fill(nil)
	if err != nil {
		t.Fatalf("rendering the enhancement instruction: %v", err)
	}
	for _, want := range []string{"Enhance the following", "vivid", "narrative richness"} {
		if !strings.Contains(out, want) {
			t.Errorf("enhancement instruction is missing %q", want)
		}
	}
}
