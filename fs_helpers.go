package dalle

import (
	"os"
	"path/filepath"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readTextFile(path string) string {
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return string(contents)
}

func writeTextFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o600)
}
