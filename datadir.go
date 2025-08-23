package dalle

import (
	"os"
	"path/filepath"
)

// ComputeDataDir resolves a base data directory using precedence: explicit flag > env > default (under home).
// Exported for server code and tests.
func ComputeDataDir(flagVal, envVal string) string {
	dataDir := flagVal
	if dataDir == "" {
		dataDir = envVal
	}
	if dataDir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil || home == "" {
			home = "."
		}
		dataDir = filepath.Join(home, ".local", "share", "trueblocks", "dalle")
	}
	dataDir = filepath.Clean(dataDir)
	if !filepath.IsAbs(dataDir) {
		if abs, aerr := filepath.Abs(dataDir); aerr == nil {
			dataDir = abs
		}
	}
	return dataDir
}
