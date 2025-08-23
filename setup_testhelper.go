package dalle

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Public test helper: creates isolated data dir with series and output, sets skip image.
// Intended for consumers writing tests against the dalle package.
type SetupTestOptions struct {
	Series        []string
	ManagerConfig *ManagerOptions
}

type SetupTestResult struct {
	TmpDir    string
	SeriesDir string
	OutputDir string
}

func SetupTest(t testing.TB, opts SetupTestOptions) SetupTestResult {
	t.Helper()
	tmp, err := os.MkdirTemp("", "dalle-public-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	// Reset and set base directory internally (test build tag provides reset)
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.Setenv("TB_DALLE_SKIP_IMAGE", "1")
	t.Cleanup(func() { _ = os.Unsetenv("TB_DALLE_SKIP_IMAGE") })
	seriesDir := SeriesDir() // will resolve under tmp now
	_ = os.MkdirAll(seriesDir, 0o750)
	for _, s := range opts.Series {
		_ = os.WriteFile(filepath.Join(seriesDir, s+".json"), []byte(`{"suffix":"`+s+`"}`), 0o600)
	}
	outputDir := OutputDir()
	_ = os.MkdirAll(outputDir, 0o750)
	if opts.ManagerConfig != nil {
		ConfigureManager(*opts.ManagerConfig)
	}
	return SetupTestResult{TmpDir: tmp, SeriesDir: seriesDir, OutputDir: outputDir}
}

// TestOnlyResetDataDir resets internal directory state so tests can isolate.
func TestOnlyResetDataDir() {
	dataDir = ""
	dataDirOnce = sync.Once{}
}
