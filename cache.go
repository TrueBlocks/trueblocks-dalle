package dalle

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
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

// TemplateRecord holds a template name and its source string
type TemplateRecord struct {
	Name   string `json:"name"`   // Template name (e.g., "prompt")
	Source string `json:"source"` // Template source string
}

// TemplateCache holds template sources for compilation
type TemplateCache struct {
	Version    string           `json:"version"`    // Template version
	Timestamp  int64            `json:"timestamp"`  // Cache creation time
	Templates  []TemplateRecord `json:"templates"`  // Template records
	Checksum   string           `json:"checksum"`   // SHA256 of template sources
	SourceHash string           `json:"sourceHash"` // Hash of source templates
}

// CacheManager handles loading and building binary caches
type CacheManager struct {
	mu            sync.RWMutex
	cacheDir      string
	dbCache       *DatabaseCache
	templateCache *TemplateCache
	loaded        bool
}

var (
	cacheManager     *CacheManager
	cacheManagerOnce sync.Once
)

// GetCacheManager returns the singleton cache manager
func GetCacheManager() *CacheManager {
	cacheManagerOnce.Do(func() {
		cacheManager = &CacheManager{
			cacheDir: filepath.Join(DataDir(), "cache"),
		}
	})
	return cacheManager
}

// LoadOrBuild ensures caches are loaded, building them if necessary
func (cm *CacheManager) LoadOrBuild() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.loaded {
		return nil
	}

	// Ensure cache directory exists
	if err := file.EstablishFolder(cm.cacheDir); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load or build database cache
	if err := cm.loadOrBuildDatabaseCache(); err != nil {
		logger.Error("Failed to load database cache, using embedded fallback:", err)
		// Continue without cache - fallback to embedded resources
	}

	// Load or build template cache
	if err := cm.loadOrBuildTemplateCache(); err != nil {
		logger.Error("Failed to load template cache, using embedded fallback:", err)
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

// GetTemplate returns a compiled template, loading from cache or embedded resources
func (cm *CacheManager) GetTemplate(name string) (*template.Template, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try cache first
	if cm.templateCache != nil {
		if templateBytes, exists := cm.templateCache.Templates[name]; exists {
			// Deserialize template from GOB
			return cm.deserializeTemplate(name, templateBytes)
		}
	}

	// Fallback to embedded templates
	return cm.getEmbeddedTemplate(name)
}

// extractVersionFromEmbedded extracts version from the first CSV in embedded databases
func (cm *CacheManager) extractVersionFromEmbedded() (string, error) {
	if len(DatabaseNames) == 0 {
		return "", fmt.Errorf("no database names configured")
	}

	// Read first CSV to extract version
	lines, err := readDatabaseCSV(DatabaseNames[0] + ".csv")
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

	// Extract version from first database to determine cache filename
	version, err := cm.extractVersionFromEmbedded()
	if err != nil {
		logger.Error("Failed to extract version, using default:", err)
		version = "v0.1.0"
	}

	// Try to load existing cache with versioned filename
	cacheFile := filepath.Join(cm.cacheDir, fmt.Sprintf("databases_%s.gob", version))
	if file.FileExists(cacheFile) {
		if cache, err := cm.loadDatabaseCache(cacheFile); err == nil {
			// Verify cache is still valid
			if cache.SourceHash == embeddedHash {
				cm.dbCache = cache
				logger.Info("Loaded database cache", "version", cache.Version, "count", len(cache.Databases))
				return nil
			}
			logger.Info("Database cache outdated, rebuilding", "cached", cache.SourceHash[:8], "current", embeddedHash[:8])
		}
	}

	// Build new cache
	logger.Info("Building database cache from embedded resources")
	cache, err := cm.buildDatabaseCache()
	if err != nil {
		return fmt.Errorf("failed to build database cache: %w", err)
	}

	cache.SourceHash = embeddedHash

	// Save cache to disk with versioned filename
	if err := cm.saveDatabaseCache(cacheFile, cache); err != nil {
		logger.Error("Failed to save database cache:", err)
		// Continue with in-memory cache
	}

	cm.dbCache = cache
	logger.Info("Built database cache", "version", cache.Version, "count", len(cache.Databases))
	return nil
} // buildDatabaseCache creates a new database cache from embedded resources
func (cm *CacheManager) buildDatabaseCache() (*DatabaseCache, error) {
	cache := &DatabaseCache{
		Timestamp: time.Now().Unix(),
		Databases: make(map[string]DatabaseIndex),
		Checksum:  fmt.Sprintf("%x", sha256.Sum256(embeddedDbs)),
	}

	var version string

	// Process each database
	for _, dbName := range DatabaseNames {
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
	lines, err := readDatabaseCSV(dbName + ".csv")
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

	return DatabaseIndex{
		Name:    dbName,
		Version: version,
		Records: records,
		Lookup:  lookup,
	}, nil
}

// Placeholder methods for template cache (to be implemented)
func (cm *CacheManager) loadOrBuildTemplateCache() error {
	// TODO: Implement template caching
	logger.Info("Template caching not yet implemented, using embedded templates")
	return nil
}

func (cm *CacheManager) getEmbeddedTemplate(name string) (*template.Template, error) {
	// Return existing global templates for now
	switch name {
	case "prompt":
		return promptTemplate, nil
	case "data":
		return dataTemplate, nil
	case "terse":
		return terseTemplate, nil
	case "title":
		return titleTemplate, nil
	case "author":
		return authorTemplate, nil
	default:
		return nil, fmt.Errorf("unknown template: %s", name)
	}
}

func (cm *CacheManager) deserializeTemplate(name string, data []byte) (*template.Template, error) {
	// TODO: Implement template deserialization
	return cm.getEmbeddedTemplate(name)
}

// saveDatabaseCache saves cache to disk using GOB encoding
func (cm *CacheManager) saveDatabaseCache(filename string, cache *DatabaseCache) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(cache)
}

// loadDatabaseCache loads cache from disk using GOB encoding
func (cm *CacheManager) loadDatabaseCache(filename string) (*DatabaseCache, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
	cm.templateCache = nil
	cm.loaded = false

	// Remove all cache files (both versioned and unversioned for cleanup)
	pattern := filepath.Join(cm.cacheDir, "databases_*.gob")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logger.Error("Failed to glob cache files:", err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			logger.Error("Failed to remove cache file:", err)
		}
	}

	// Also remove legacy unversioned cache file if it exists
	legacyCacheFile := filepath.Join(cm.cacheDir, "databases.gob")
	if file.FileExists(legacyCacheFile) {
		if err := os.Remove(legacyCacheFile); err != nil {
			return fmt.Errorf("failed to remove legacy cache: %w", err)
		}
	}

	logger.Info("Cache invalidated")
	return nil
}
