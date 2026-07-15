package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCacheManager_DatabaseCache(t *testing.T) {
	// Setup isolated test environment
	tmpDir, err := os.MkdirTemp("", "dalle-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Reset global state for isolated testing
	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	// Get cache manager
	cm := GetCacheManager()

	// Test LoadOrBuild
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild failed: %v", err)
	}

	// Test that cache was created
	cacheFile := filepath.Join(tmpDir, "cache", "databases_v0.1.0.gob")
	if !fileExists(cacheFile) {
		t.Error("Expected cache file to be created")
	}

	// Test GetDatabase
	dbIndex, err := cm.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase failed: %v", err)
	}

	if dbIndex.Name != "nouns" {
		t.Errorf("Expected database name 'nouns', got '%s'", dbIndex.Name)
	}

	if len(dbIndex.Records) == 0 {
		t.Error("Expected database records, got none")
	}

	if dbIndex.Version == "" {
		t.Error("Expected version to be set")
	}

	// Test lookup functionality
	if len(dbIndex.Lookup) == 0 {
		t.Error("Expected lookup map to be populated")
	}

	// Verify lookup works
	if len(dbIndex.Records) > 0 {
		firstKey := dbIndex.Records[0].Key
		if _, exists := dbIndex.Lookup[firstKey]; !exists {
			t.Errorf("Expected key '%s' to exist in lookup map", firstKey)
		}
	}
}

func TestCacheManager_CacheReuse(t *testing.T) {
	// Setup isolated test environment
	tmpDir, err := os.MkdirTemp("", "dalle-cache-reuse-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Reset global state
	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	// Create cache manager and build cache
	cm1 := GetCacheManager()
	if err := cm1.LoadOrBuild(); err != nil {
		t.Fatalf("First LoadOrBuild failed: %v", err)
	}

	// Get database to ensure cache is populated
	_, err = cm1.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase failed: %v", err)
	}

	// Reset the singleton and create new cache manager
	cacheManagerOnce = sync.Once{}
	cacheManager = nil

	cm2 := GetCacheManager()
	if err := cm2.LoadOrBuild(); err != nil {
		t.Fatalf("Second LoadOrBuild failed: %v", err)
	}

	// Verify cache is loaded from disk (not rebuilt)
	if cm2.dbCache == nil {
		t.Error("Expected cache to be loaded from disk")
	}

	// Verify database can still be retrieved
	dbIndex, err := cm2.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase from cached data failed: %v", err)
	}

	if len(dbIndex.Records) == 0 {
		t.Error("Expected cached database to have records")
	}
}

func TestCacheManager_NounsFlattened(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-nouns-flattened-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild failed: %v", err)
	}

	nouns, err := cm.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase nouns failed: %v", err)
	}

	if len(nouns.Columns) == 0 {
		t.Fatal("Expected nouns columns to be populated")
	}

	for _, rec := range nouns.Records {
		if len(rec.Values) != len(nounColumnNames) {
			t.Errorf("Expected %d flattened values for %q, got %d", len(nounColumnNames), rec.Key, len(rec.Values))
		}
	}
}

func TestCacheManager_NounsFlattenedCompleteness(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-nouns-complete-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild failed: %v", err)
	}

	nouns, err := cm.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase nouns failed: %v", err)
	}

	for _, rec := range nouns.Records {
		if len(rec.Values) != len(nounColumnNames) {
			t.Errorf("Expected %d flattened values for %q, got %d", len(nounColumnNames), rec.Key, len(rec.Values))
			continue
		}
		if rec.Values[1] == "" || rec.Values[2] == "" || rec.Values[3] == "" || rec.Values[4] == "" {
			t.Errorf("Incomplete hierarchy for noun %q: familyCommon=%q orderCommon=%q classCommon=%q phylumCommon=%q",
				rec.Key, rec.Values[1], rec.Values[2], rec.Values[3], rec.Values[4])
		}
	}
}

func TestCacheManager_HierarchyFilesCached(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-hierarchy-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild failed: %v", err)
	}

	for _, name := range HierarchyDatabaseNames {
		idx, err := cm.GetDatabase(name)
		if err != nil {
			t.Errorf("GetDatabase %q failed: %v", name, err)
			continue
		}
		if idx.Name != name {
			t.Errorf("Expected database name %q, got %q", name, idx.Name)
		}
		if len(idx.Records) == 0 {
			t.Errorf("Expected records for %q", name)
		}
	}
}

func TestCacheManager_EnrichedValuesShape(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dalle-enriched-shape-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	TestOnlyResetDataDir(tmpDir)
	TestOnlyResetCacheManager()

	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		t.Fatalf("LoadOrBuild failed: %v", err)
	}

	nouns, err := cm.GetDatabase("nouns")
	if err != nil {
		t.Fatalf("GetDatabase nouns failed: %v", err)
	}

	if len(nouns.Columns) != len(nounColumnNames) {
		t.Fatalf("Expected %d columns, got %d", len(nounColumnNames), len(nouns.Columns))
	}
	for i, want := range nounColumnNames {
		if nouns.Columns[i] != want {
			t.Errorf("Expected column %d to be %q, got %q", i, want, nouns.Columns[i])
		}
	}

	for _, rec := range nouns.Records {
		if len(rec.Values) != len(nounColumnNames) {
			t.Errorf("Expected %d enriched values for %q, got %d", len(nounColumnNames), rec.Key, len(rec.Values))
			continue
		}
		if rec.Values[0] != rec.Key {
			t.Errorf("Expected value[0] == key for %q", rec.Key)
		}
		for i := 1; i < len(rec.Values); i++ {
			if rec.Values[i] == "" {
				t.Errorf("Expected non-empty value[%d] for %q", i, rec.Key)
			}
		}
	}
}
