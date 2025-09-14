package utils

import (
	"os"
	"path/filepath"
)

// Clean removes all files associated with a given series/address pair from the output directory.
// Clean removes all files associated with a given series/address pair from the output directory.
// OutputDir must be provided by the caller (dependency injection).
func Clean(outputDir, series, addr string) {
	baseDir := filepath.Join(outputDir, series)
	paths := []string{
		filepath.Join(baseDir, "annotated", addr+".png"),
		filepath.Join(baseDir, "selector", addr+".json"),
		filepath.Join(baseDir, "generated", addr+".png"),
		filepath.Join(baseDir, "audio", addr+".mp3"),
	}
	for _, dir := range []string{"data", "title", "terse", "prompt", "enhanced"} {
		paths = append(paths, filepath.Join(baseDir, dir, addr+".txt"))
	}
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
