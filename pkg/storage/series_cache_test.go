package storage

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

func TestSeriesArchive_Basics(t *testing.T) {
	hash := SeriesArchiveHash()
	if hash == "" {
		t.Fatal("expected non-empty archive hash")
	}

	suffixes, err := ListEmbeddedSeriesSuffixes()
	if err != nil {
		t.Fatalf("ListEmbeddedSeriesSuffixes: %v", err)
	}
	if len(suffixes) == 0 {
		t.Fatal("expected at least one built-in series")
	}

	if !slices.Contains(suffixes, "empty") {
		t.Fatalf("expected 'empty' in built-in suffixes: %v", suffixes)
	}

	body, ok := ReadEmbeddedSeriesJSON("empty")
	if !ok {
		t.Fatal("expected to read embedded 'empty' series")
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty JSON body for 'empty' series")
	}

	_, ok = ReadEmbeddedSeriesJSON("definitely-not-a-series")
	if ok {
		t.Fatal("expected missing series lookup to fail")
	}
}

func TestCacheManager_SeriesCacheBuild(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-series-cache-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild: %v", err)
	}

	cacheFile := filepath.Join(tmpDir, "cache", "series_v1.0.0.gob")
	if !fileExists(cacheFile) {
		t.Fatal("expected series cache file to be created")
	}

	body, ok := cm.GetSeriesJSON("empty")
	if !ok {
		t.Fatal("expected 'empty' series in cache")
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty 'empty' series body")
	}

	builtins := cm.ListSeriesJSON()
	if len(builtins) == 0 {
		t.Fatal("expected built-in series in cache")
	}
}

func TestCacheManager_SeriesCacheReuse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-series-cache-reuse-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm1 := GetCacheManager()
	if err := cm1.LoadOrBuild(); err != nil {
		t.Fatalf("first LoadOrBuild: %v", err)
	}
	if cm1.seriesCache == nil {
		t.Fatal("expected series cache populated")
	}

	cacheManagerOnce = sync.Once{}
	cacheManager = nil

	cm2 := GetCacheManager()
	if err := cm2.LoadOrBuild(); err != nil {
		t.Fatalf("second LoadOrBuild: %v", err)
	}
	if cm2.seriesCache == nil {
		t.Fatal("expected series cache loaded from disk")
	}

	body, ok := cm2.GetSeriesJSON("empty")
	if !ok || len(body) == 0 {
		t.Fatal("expected 'empty' series after cache reuse")
	}
}

func TestCacheManager_SeriesCacheInvalidate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-series-cache-invalidate-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild: %v", err)
	}

	cacheFile := filepath.Join(tmpDir, "cache", "series_v1.0.0.gob")
	if !fileExists(cacheFile) {
		t.Fatal("expected cache file to exist")
	}

	if err := cm.InvalidateCache(); err != nil {
		t.Fatalf("InvalidateCache: %v", err)
	}

	if fileExists(cacheFile) {
		t.Error("expected series cache file removed after invalidation")
	}
	if cm.seriesCache != nil {
		t.Error("expected in-memory series cache cleared")
	}
}

func TestCacheManager_SeriesCacheRebuildOnHashMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-series-cache-mismatch-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild: %v", err)
	}

	cacheFile := filepath.Join(tmpDir, "cache", "series_v1.0.0.gob")
	if !fileExists(cacheFile) {
		t.Fatal("expected cache file to exist")
	}

	// Corrupt the stored hash to force a rebuild.
	loaded, err := cm.loadSeriesCache(cacheFile)
	if err != nil {
		t.Fatalf("loadSeriesCache: %v", err)
	}
	loaded.SourceHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := cm.saveSeriesCache(cacheFile, loaded); err != nil {
		t.Fatalf("saveSeriesCache: %v", err)
	}

	// Re-create manager so it reloads from disk.
	cacheManagerOnce = sync.Once{}
	cacheManager = nil
	cm2 := GetCacheManager()
	if err := cm2.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild after hash mismatch: %v", err)
	}

	reloaded, err := cm2.loadSeriesCache(cacheFile)
	if err != nil {
		t.Fatalf("loadSeriesCache after rebuild: %v", err)
	}
	if reloaded.SourceHash != SeriesArchiveHash() {
		t.Fatalf("expected cache hash to be rebuilt, got %s want %s", reloaded.SourceHash, SeriesArchiveHash())
	}
}
