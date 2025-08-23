package dalle

import (
	"os"
	"path/filepath"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

// ComputeDataDir resolves a base data directory using precedence: explicit flag > env > default (under home).
// Exported for server code and tests.
func ComputeDataDir(flagVal string) (string, error) {
	dataDir := flagVal
	if dataDir == "" {
		dataDir = os.Getenv("TB_DALLE_DATA_DIR")
	}
	if dataDir == "" {
		if home, herr := os.UserHomeDir(); herr != nil {
			return "", herr
		} else {
			dataDir = filepath.Join(home, ".local", "share", "trueblocks", "dalle")
		}
	}
	dataDir = filepath.Clean(dataDir)

	if !filepath.IsAbs(dataDir) {
		if abs, aerr := filepath.Abs(dataDir); aerr == nil {
			dataDir = abs
		}
	}

	if err := EnsureWritable(dataDir); err != nil {
		if tmp, terr := os.MkdirTemp("", "dalleserver-fallback-*"); terr != nil {
			logger.Error("ERROR: cannot establish writable data dir:", err)
			dataDir = dataDir + "-unwritable"
		} else {
			logger.Error("WARNING: using fallback temp data dir due to error:", err)
			dataDir = tmp
		}
	}

	return dataDir, nil
}

// Dir helpers (pure functions) derived from a base data directory.
func OutputDir(base string) string  { return filepath.Join(base, "output") }
func SeriesDir(base string) string  { return filepath.Join(base, "series") }
func LogsDir(base string) string    { return filepath.Join(base, "logs") }
func MetricsDir(base string) string { return filepath.Join(base, "metrics") }

// EnsureWritable makes sure directory exists and is writable.
func EnsureWritable(path string) error {
	// Create (or ensure) the directory with restricted permissions; callers can relax if explicitly required.
	if err := os.MkdirAll(path, 0o750); err != nil {
		return err
	}
	sentinel := filepath.Join(path, ".write_test")
	// Use 0o600 for the write test to satisfy gosec and to avoid exposing potential sensitive data.
	if werr := os.WriteFile(sentinel, []byte("ok"), 0o600); werr != nil {
		return werr
	}
	_ = os.Remove(sentinel)
	return nil
}
