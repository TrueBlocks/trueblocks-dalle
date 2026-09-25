package image

import (
	"os"
	"testing"
)

// TestMain pins the OpenAI models these tests exercise. The package default
// follows the registry's pro tier (Opus 5.5, Gemini Pro Image); these
// environment variables are the same override a user sets to restore the
// OpenAI path, so the tests cover it and never call a live provider.
func TestMain(m *testing.M) {
	_ = os.Setenv("TB_DALLE_ENHANCEMENT_MODEL", "gpt-5.5")
	_ = os.Setenv("TB_DALLE_IMAGE_MODEL", "gpt-image-2")
	os.Exit(m.Run())
}
