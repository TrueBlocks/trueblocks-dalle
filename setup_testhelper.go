package dalle

import (
	"os"
	"path/filepath"
	"testing"
)

// Public test helper: creates isolated data dir with series and output, sets skip image.
// Intended for consumers writing tests against the dalle package.
type SetupTestOptions struct {
	Series        []string
	ManagerConfig *ManagerOptions
}

func SetupTest(t testing.TB, opts SetupTestOptions) {
	t.Helper()
	tmp, err := os.MkdirTemp("", "dalle-public-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.Setenv("TB_DALLE_SKIP_IMAGE", "1")
	t.Cleanup(func() { _ = os.Unsetenv("TB_DALLE_SKIP_IMAGE") })
	for _, s := range opts.Series {
		_ = os.WriteFile(filepath.Join(SeriesDir(), s+".json"), []byte(`{"suffix":"`+s+`"}`), 0o600)
	}
	_ = os.MkdirAll(OutputDir(), 0o750)
	if opts.ManagerConfig != nil {
		ConfigureManager(*opts.ManagerConfig)
	}
}
