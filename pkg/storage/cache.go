package storage

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	logger "github.com/TrueBlocks/trueblocks-dalle/v6/pkg/logging"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

// DatabaseRecord represents a single row from a CSV database
type DatabaseRecord struct {
	Key    string   `json:"key"`    // Primary identifier (e.g., "aardvark")
	Values []string `json:"values"` // All column values including version
}

// DatabaseIndex provides fast access to database records
type DatabaseIndex struct {
	Name    string           `json:"name"`    // Database name (e.g., "nouns")
	Version string           `json:"version"` // Version from CSV
	Records []DatabaseRecord `json:"records"` // All records
	Lookup  map[string]int   `json:"lookup"`  // Key -> record index mapping
}

// DatabaseCache holds all processed database indexes
type DatabaseCache struct {
	Version    string                   `json:"version"`    // Overall version
	Timestamp  int64                    `json:"timestamp"`  // Cache creation time
	Databases  map[string]DatabaseIndex `json:"databases"`  // Database name -> index
	Checksum   string                   `json:"checksum"`   // SHA256 of embedded tar.gz
	SourceHash string                   `json:"sourceHash"` // Hash of source data for validation
}

// SeriesCache holds the raw JSON bodies of built-in series.
type SeriesCache struct {
	Version    string            `json:"version"`    // Overall version
	Timestamp  int64             `json:"timestamp"`  // Cache creation time
	Builtins   map[string][]byte `json:"builtins"`   // Series suffix -> raw JSON
	Checksum   string            `json:"checksum"`   // SHA256 of embedded tar.gz
	SourceHash string            `json:"sourceHash"` // SHA256 of embedded tar.gz for validation
}

// CacheManager handles loading and building binary caches
type CacheManager struct {
	mu          sync.RWMutex
	cacheDir    string
	dbCache     *DatabaseCache
	seriesCache *SeriesCache
	loaded      bool
}

// GetCacheManager returns the singleton cache manager
func GetCacheManager() *CacheManager {
	storageStateMu.Lock()
	defer storageStateMu.Unlock()

	cacheManagerOnce.Do(func() {
		cacheManager = &CacheManager{
			cacheDir: filepath.Join(dataDirLocked(), "cache"),
		}
	})
	return cacheManager
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestOnlyResetCacheManager resets cache manager singleton for testing isolation
func TestOnlyResetCacheManager() {
	storageStateMu.Lock()
	defer storageStateMu.Unlock()

	cacheManagerOnce = sync.Once{}
	cacheManager = nil
}

// LoadOrBuild ensures caches are loaded, building them if necessary
func (cm *CacheManager) LoadOrBuild() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	logger.InfoG(fmt.Sprintf("DEBUG: LoadOrBuild called, loaded=%t", cm.loaded))
	if cm.loaded {
		logger.InfoG("DEBUG: Cache already loaded, skipping")
		return nil
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cm.cacheDir, 0o750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load or build database cache
	if err := cm.loadOrBuildDatabaseCache(); err != nil {
		logger.Error("Failed to load database cache, using embedded fallback:", err)
		// Continue without cache - fallback to embedded resources
	}

	// Load or build series cache
	if err := cm.loadOrBuildSeriesCache(); err != nil {
		logger.Error("Failed to load series cache, using embedded fallback:", err)
		// Continue without cache - fallback to embedded resources
	}

	cm.loaded = true
	return nil
}

// GetDatabase returns a database index, loading from cache or embedded resources
func (cm *CacheManager) GetDatabase(name string) (DatabaseIndex, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try cache first
	if cm.dbCache != nil {
		if idx, exists := cm.dbCache.Databases[name]; exists {
			return idx, nil
		}
	}

	// Fallback to embedded resources
	return cm.buildDatabaseIndex(name)
}

// extractVersionFromEmbedded extracts version from the first CSV in embedded databases
func (cm *CacheManager) extractVersionFromEmbedded() (string, error) {
	if len(prompt.DatabaseNames) == 0 {
		return "", fmt.Errorf("no database names configured")
	}

	// Read first CSV to extract version
	lines, err := ReadDatabaseCSV(prompt.DatabaseNames[0] + ".csv")
	if err != nil {
		return "", fmt.Errorf("failed to read first database: %w", err)
	}

	if len(lines) < 2 {
		return "", fmt.Errorf("insufficient data in database")
	}

	// Extract version from first data row (second line)
	firstDataLine := lines[1]
	if strings.HasPrefix(firstDataLine, "v") {
		parts := strings.SplitN(firstDataLine, ",", 2)
		if len(parts) > 0 {
			return parts[0], nil
		}
	}

	return "v0.1.0", nil // default version
}

// loadOrBuildDatabaseCache loads existing cache or builds new one
func (cm *CacheManager) loadOrBuildDatabaseCache() error {
	// Calculate current embedded data hash
	embeddedHash := fmt.Sprintf("%x", sha256.Sum256(embeddedDbs))
	logger.InfoG(fmt.Sprintf("DEBUG: Current embedded hash: %s (size: %d bytes)", embeddedHash[:16], len(embeddedDbs)))

	// Extract version from first database to determine cache filename
	version, err := cm.extractVersionFromEmbedded()
	if err != nil {
		logger.Error("Failed to extract version, using default:", err)
		version = "v0.1.0"
	}
	logger.InfoG(fmt.Sprintf("DEBUG: Extracted version: %s", version))

	// Try to load existing cache with versioned filename
	cacheFile := filepath.Join(cm.cacheDir, fmt.Sprintf("databases_%s.gob", version))
	logger.InfoG(fmt.Sprintf("DEBUG: Looking for cache file: %s", cacheFile))

	if fileExists(cacheFile) {
		logger.InfoG("DEBUG: Cache file exists, attempting to load")
		if cache, err := cm.loadDatabaseCache(cacheFile); err == nil {
			logger.InfoG(fmt.Sprintf("DEBUG: Loaded cache - stored hash: %s", cache.SourceHash[:16]))

			// Check if database names match (detect schema changes)
			expectedDBs := make(map[string]bool)
			for _, name := range prompt.DatabaseNames {
				expectedDBs[name] = true
			}

			schemaMismatch := false
			if len(cache.Databases) != len(expectedDBs) {
				schemaMismatch = true
				logger.InfoG(fmt.Sprintf("DEBUG: Database count mismatch - cached: %d, expected: %d", len(cache.Databases), len(expectedDBs)))
			} else {
				for cachedDB := range cache.Databases {
					if !expectedDBs[cachedDB] {
						schemaMismatch = true
						logger.InfoG(fmt.Sprintf("DEBUG: Unexpected database in cache: %s", cachedDB))
						break
					}
				}
			}

			// Verify cache is still valid (both hash and schema)
			if !schemaMismatch && cache.SourceHash == embeddedHash {
				cm.dbCache = cache
				logger.Info("Loaded database cache", "version", cache.Version, "count", len(cache.Databases))
				return nil
			}

			if schemaMismatch {
				logger.Info("Database schema changed, rebuilding cache", "cached", len(cache.Databases), "expected", len(expectedDBs))
			} else {
				logger.Info("Database cache outdated, rebuilding", "cached", cache.SourceHash[:8], "current", embeddedHash[:8])
			}
			logger.InfoG(fmt.Sprintf("DEBUG: Full hash comparison - cached: %s, current: %s", cache.SourceHash, embeddedHash))
		} else {
			logger.InfoG(fmt.Sprintf("DEBUG: Failed to load cache file: %v", err))
		}
	} else {
		logger.InfoG("DEBUG: Cache file does not exist")
	}

	// Build new cache
	logger.Info("Building database cache from embedded resources")
	cache, err := cm.buildDatabaseCache()
	if err != nil {
		return fmt.Errorf("failed to build database cache: %w", err)
	}

	cache.SourceHash = embeddedHash
	logger.InfoG(fmt.Sprintf("DEBUG: Saving cache with hash: %s", embeddedHash[:16]))

	// Save cache to disk with versioned filename
	if err := cm.saveDatabaseCache(cacheFile, cache); err != nil {
		logger.Error("Failed to save database cache:", err)
		// Continue with in-memory cache
	} else {
		logger.InfoG(fmt.Sprintf("DEBUG: Successfully saved cache to: %s", cacheFile))
	}

	cm.dbCache = cache
	logger.Info("Built database cache", "version", cache.Version, "count", len(cache.Databases))
	return nil
}

// GetSeriesJSON returns the raw JSON body for a built-in series.
func (cm *CacheManager) GetSeriesJSON(suffix string) ([]byte, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.seriesCache == nil {
		return nil, false
	}
	body, ok := cm.seriesCache.Builtins[suffix]
	return body, ok
}

// ListSeriesJSON returns a copy of the built-in series suffix-to-JSON map.
func (cm *CacheManager) ListSeriesJSON() map[string][]byte {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.seriesCache == nil {
		return nil
	}
	out := make(map[string][]byte, len(cm.seriesCache.Builtins))
	maps.Copy(out, cm.seriesCache.Builtins)
	return out
}

// loadOrBuildSeriesCache loads an existing series cache or builds one from the embedded archive.
func (cm *CacheManager) loadOrBuildSeriesCache() error {
	embeddedHash := SeriesArchiveHash()
	logger.InfoG(fmt.Sprintf("DEBUG: Current embedded series hash: %s (size: %d bytes)", embeddedHash[:16], len(embeddedSeries)))

	version, err := cm.extractSeriesVersionFromEmbedded()
	if err != nil {
		logger.Error("Failed to extract series version, using default:", err)
		version = "v1.0.0"
	}
	logger.InfoG(fmt.Sprintf("DEBUG: Extracted series version: %s", version))

	cacheFile := filepath.Join(cm.cacheDir, fmt.Sprintf("series_%s.gob", version))
	logger.InfoG(fmt.Sprintf("DEBUG: Looking for series cache file: %s", cacheFile))

	if fileExists(cacheFile) {
		logger.InfoG("DEBUG: Series cache file exists, attempting to load")
		if cache, err := cm.loadSeriesCache(cacheFile); err == nil {
			logger.InfoG(fmt.Sprintf("DEBUG: Loaded series cache - stored hash: %s", cache.SourceHash[:16]))
			if cache.SourceHash == embeddedHash {
				cm.seriesCache = cache
				logger.Info("Loaded series cache", "version", cache.Version, "count", len(cache.Builtins))
				return nil
			}
			logger.Info("Series cache outdated, rebuilding", "cached", cache.SourceHash[:8], "current", embeddedHash[:8])
		} else {
			logger.InfoG(fmt.Sprintf("DEBUG: Failed to load series cache file: %v", err))
		}
	} else {
		logger.InfoG("DEBUG: Series cache file does not exist")
	}

	logger.Info("Building series cache from embedded resources")
	cache, err := cm.buildSeriesCache(version)
	if err != nil {
		return fmt.Errorf("failed to build series cache: %w", err)
	}

	cache.SourceHash = embeddedHash
	logger.InfoG(fmt.Sprintf("DEBUG: Saving series cache with hash: %s", embeddedHash[:16]))

	if err := cm.saveSeriesCache(cacheFile, cache); err != nil {
		logger.Error("Failed to save series cache:", err)
	} else {
		logger.InfoG(fmt.Sprintf("DEBUG: Successfully saved series cache to: %s", cacheFile))
	}

	cm.seriesCache = cache
	logger.Info("Built series cache", "version", cache.Version, "count", len(cache.Builtins))
	return nil
}

func (cm *CacheManager) extractSeriesVersionFromEmbedded() (string, error) {
	var version string
	err := WalkSeriesArchive(func(path string, body []byte) error {
		var v struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			return err
		}
		version = v.Version
		return io.EOF
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if version == "" {
		return "v1.0.0", nil
	}
	return version, nil
}

func (cm *CacheManager) buildSeriesCache(version string) (*SeriesCache, error) {
	cache := &SeriesCache{
		Version:   version,
		Timestamp: time.Now().Unix(),
		Builtins:  make(map[string][]byte),
		Checksum:  SeriesArchiveHash(),
	}

	err := WalkSeriesArchive(func(path string, body []byte) error {
		suffix := seriesSuffixFromPath(path)
		cache.Builtins[suffix] = body
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func (cm *CacheManager) saveSeriesCache(filename string, cache *SeriesCache) error {
	filename = filepath.Clean(filename)
	if !strings.HasPrefix(filename, filepath.Clean(cm.cacheDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid cache path: %s", filename)
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(cache)
}

func (cm *CacheManager) loadSeriesCache(filename string) (*SeriesCache, error) {
	filename = filepath.Clean(filename)
	if !strings.HasPrefix(filename, filepath.Clean(cm.cacheDir)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid cache path: %s", filename)
	}
	file, err := os.Open(filename) // #nosec G304 path validated
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var cache SeriesCache
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// buildDatabaseCache creates a new database cache from embedded resources
func (cm *CacheManager) buildDatabaseCache() (*DatabaseCache, error) {
	cache := &DatabaseCache{
		Timestamp: time.Now().Unix(),
		Databases: make(map[string]DatabaseIndex),
		Checksum:  fmt.Sprintf("%x", sha256.Sum256(embeddedDbs)),
	}

	var version string

	// Process each database
	for _, dbName := range prompt.DatabaseNames {
		idx, err := cm.buildDatabaseIndex(dbName)
		if err != nil {
			return nil, fmt.Errorf("failed to build index for %s: %w", dbName, err)
		}

		cache.Databases[dbName] = idx

		// Use first database's version as overall version
		if version == "" {
			version = idx.Version
		}
	}

	cache.Version = version
	return cache, nil
}

// buildDatabaseIndex creates an index for a single database
func (cm *CacheManager) buildDatabaseIndex(dbName string) (DatabaseIndex, error) {
	lines, err := ReadDatabaseCSV(dbName + ".csv")
	if err != nil {
		return DatabaseIndex{}, fmt.Errorf("failed to read %s: %w", dbName, err)
	}

	if len(lines) == 0 {
		return DatabaseIndex{}, fmt.Errorf("empty database: %s", dbName)
	}

	// Skip header and process records
	records := make([]DatabaseRecord, 0, len(lines)-1)
	lookup := make(map[string]int)
	var version string

	for i, line := range lines[1:] { // Skip header
		// Remove version prefix if present
		cleanLine := strings.Replace(line, "v0.1.0,", "", 1)
		if version == "" && strings.HasPrefix(line, "v") {
			// Extract version from first record
			parts := strings.SplitN(line, ",", 2)
			if len(parts) > 0 {
				version = parts[0]
			}
		}

		// Parse CSV line (simple split for now)
		values := strings.Split(cleanLine, ",")
		if len(values) == 0 {
			continue
		}

		key := strings.TrimSpace(values[0])
		if key == "" {
			continue
		}

		record := DatabaseRecord{
			Key:    key,
			Values: values,
		}

		records = append(records, record)
		lookup[key] = i
	}

	if version == "" {
		version = "v0.1.0" // Default version
	}

	idx := DatabaseIndex{
		Name:    dbName,
		Version: version,
		Records: records,
		Lookup:  lookup,
	}

	if dbName == "nouns" {
		if err := enrichNounsWithHierarchy(&idx); err != nil {
			logger.Warn("failed to enrich nouns with hierarchy: " + err.Error())
		}
	}

	return idx, nil
}

// loadHierarchyLookup reads a hierarchy CSV and returns a map from the key column to
// the remaining columns. For example, families.csv maps family -> [order, commonName].
func loadHierarchyLookup(csvName string, keyCol int) (map[string][]string, error) {
	lines, err := ReadDatabaseCSV(csvName)
	if err != nil {
		return nil, err
	}
	lookup := make(map[string][]string, len(lines))
	for _, line := range lines[1:] {
		cleanLine := strings.Replace(line, "v0.1.0,", "", 1)
		parts := strings.Split(cleanLine, ",")
		if len(parts) <= keyCol {
			continue
		}
		key := strings.TrimSpace(parts[keyCol])
		if key != "" {
			lookup[key] = parts
		}
	}
	return lookup, nil
}

// enrichNounsWithHierarchy walks the taxonomy chain for each noun and appends
// the resolved hierarchy to its Values slice. After enrichment, each noun record
// has Values: [commonName, family, familyCommon, order, orderCommon, class, classCommon, phylum, phylumCommon].
func enrichNounsWithHierarchy(idx *DatabaseIndex) error {
	families, err := loadHierarchyLookup("families.csv", 0)
	if err != nil {
		return fmt.Errorf("loading families: %w", err)
	}
	orders, err := loadHierarchyLookup("orders.csv", 0)
	if err != nil {
		return fmt.Errorf("loading orders: %w", err)
	}
	classes, err := loadHierarchyLookup("classes.csv", 0)
	if err != nil {
		return fmt.Errorf("loading classes: %w", err)
	}
	phyla, err := loadHierarchyLookup("phyla.csv", 0)
	if err != nil {
		return fmt.Errorf("loading phyla: %w", err)
	}

	enriched := 0
	for i, rec := range idx.Records {
		if len(rec.Values) < 2 {
			continue
		}
		familyName := strings.TrimSpace(rec.Values[1])

		familyCommon := ""
		orderName := ""
		orderCommon := ""
		className := ""
		classCommon := ""
		phylumName := ""
		phylumCommon := ""

		if fam, ok := families[familyName]; ok && len(fam) >= 3 {
			orderName = fam[1]
			familyCommon = fam[2]
		}
		if ord, ok := orders[orderName]; ok && len(ord) >= 3 {
			className = ord[1]
			orderCommon = ord[2]
		}
		if cls, ok := classes[className]; ok && len(cls) >= 3 {
			phylumName = cls[1]
			classCommon = cls[2]
		}
		if phy, ok := phyla[phylumName]; ok && len(phy) >= 2 {
			phylumCommon = phy[1]
		}

		idx.Records[i].Values = []string{
			rec.Values[0],
			familyName, familyCommon,
			orderName, orderCommon,
			className, classCommon,
			phylumName, phylumCommon,
		}
		enriched++
	}

	logger.Info("enriched noun records with taxonomy hierarchy")
	return nil
}

// saveDatabaseCache saves cache to disk using GOB encoding
func (cm *CacheManager) saveDatabaseCache(filename string, cache *DatabaseCache) error {
	// Clean path, restrict to cacheDir to satisfy gosec G304
	filename = filepath.Clean(filename)
	if !strings.HasPrefix(filename, filepath.Clean(cm.cacheDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid cache path: %s", filename)
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(cache)
}

// loadDatabaseCache loads cache from disk using GOB encoding
func (cm *CacheManager) loadDatabaseCache(filename string) (*DatabaseCache, error) {
	filename = filepath.Clean(filename)
	if !strings.HasPrefix(filename, filepath.Clean(cm.cacheDir)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid cache path: %s", filename)
	}
	file, err := os.Open(filename) // #nosec G304 path validated
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var cache DatabaseCache
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// InvalidateCache removes cached files to force rebuild
func (cm *CacheManager) InvalidateCache() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.dbCache = nil
	cm.seriesCache = nil
	cm.loaded = false

	// Remove database cache files
	dbPattern := filepath.Join(cm.cacheDir, "databases_*.gob")
	matches, err := filepath.Glob(dbPattern)
	if err != nil {
		logger.Error("Failed to glob database cache files:", err)
	}

	// Remove series cache files
	seriesPattern := filepath.Join(cm.cacheDir, "series_*.gob")
	seriesMatches, err := filepath.Glob(seriesPattern)
	if err != nil {
		logger.Error("Failed to glob series cache files:", err)
	}
	matches = append(matches, seriesMatches...)

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			logger.Error("Failed to remove cache file:", err)
		}
	}

	// Also remove legacy unversioned database cache file if it exists
	legacyCacheFile := filepath.Join(cm.cacheDir, "databases.gob")
	if fileExists(legacyCacheFile) {
		if err := os.Remove(legacyCacheFile); err != nil {
			return fmt.Errorf("failed to remove legacy cache: %w", err)
		}
	}

	logger.Info("Cache invalidated")
	return nil
}
